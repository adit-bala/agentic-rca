import asyncio
import json
from typing import AsyncGenerator

from rich.console import Console
from fastapi import WebSocket
from pydantic import BaseModel
from dataclasses import is_dataclass, asdict

from agents import Runner, custom_span, gen_trace_id, trace, Agent, ItemHelpers, RunResultStreaming
from printer import Printer
from neo4j_agent import neo4j_agent
from k8s_agent import k8s_agent
from observe_agent import observe_agent
from rca_agent import rca_summary_agent
from codebase_agent import codebase_agent
from models import AlertGroup, ServiceGraphResponse
from agent_types import AgentName, AgentType, AGENT_TYPE_MAP
from websocket_types import (
    WebSocketMessageType,
    StatusMessage,
    ErrorMessage,
    AgentStartedMessage,
    AgentUpdatedMessage,
    ToolCallMessage,
    ToolOutputMessage,
    MessageOutputMessage
)

def _json_safe(obj):
    if isinstance(obj, BaseModel):
        return obj.model_dump()
    if is_dataclass(obj):
        return asdict(obj)
    return obj

class RCA_Manager:
    def __init__(self, websocket: WebSocket):
        self.console = Console()
        self.printer = Printer(self.console)
        self.websocket = websocket

    def _get_agent_type(self, agent_name: str) -> AgentType:
        """Get the AgentType enum value for a given agent name."""
        try:
            agent_name_enum = AgentName(agent_name)
            return AGENT_TYPE_MAP[agent_name_enum]
        except (ValueError, KeyError):
            return AgentType.NEO4J  # Default to NEO4J if unknown

    async def wrap_agent_stream(self, agent: Agent, *inputs: str) -> RunResultStreaming:
        """
        Wrap the agent stream to send the output to the websocket.
        Takes variable number of input strings and formats them as agent-compatible messages.
        """
        formatted_inputs = [{"role": "user", "content": content} for content in inputs]
        agent_type = self._get_agent_type(agent.name)
        
        result = Runner.run_streamed(agent, formatted_inputs)
        await self.websocket.send_json(AgentStartedMessage(
            type=WebSocketMessageType.AGENT_STARTED,
            agent=agent_type,
            data=agent.name
        ))
        print(f"=== {agent.name} starting ===")
        print(f"Sent AGENT_STARTED message for {agent.name}")

        async for event in result.stream_events():
            # Ignore raw responses
            if event.type == "raw_response_event":
                continue
            # When the agent updates, print that
            elif event.type == "agent_updated_stream_event":
                print(f"Agent updated: {event.new_agent.name}")
                await self.websocket.send_json(AgentUpdatedMessage(
                    type=WebSocketMessageType.AGENT_UPDATED,
                    agent=agent_type,
                    data={"name": event.new_agent.name}
                ))
                print(f"Sent AGENT_UPDATED message for {event.new_agent.name}")
                continue
            # When items are generated, print them
            elif event.type == "run_item_stream_event":
                if event.item.type == "tool_call_item":
                    print("-- Tool was called")
                    await self.websocket.send_json(ToolCallMessage(
                        type=WebSocketMessageType.TOOL_CALL,
                        agent=agent_type,
                        data="-- Tool was called"
                    ))
                    print(f"Sent TOOL_CALL message for {agent.name}")
                elif event.item.type == "tool_call_output_item":
                    print(f"-- Tool output: {event.item.output}")
                    await self.websocket.send_json(ToolOutputMessage(
                        type=WebSocketMessageType.TOOL_OUTPUT,
                        agent=agent_type,
                        data=_json_safe(event.item.output)
                    ))
                    print(f"Sent TOOL_OUTPUT message for {agent.name}")
                elif event.item.type == "message_output_item":
                    # skip neo4j output since it's just a json string
                    if agent.name == AgentName.NEO4J:
                        continue
                    message = ItemHelpers.text_message_output(event.item)
                    print(f"-- Message output:\n {message}")
                    await self.websocket.send_json(MessageOutputMessage(
                        type=WebSocketMessageType.MESSAGE_OUTPUT,
                        agent=agent_type,
                        data=message
                    ))
                    print(f"Sent MESSAGE_OUTPUT message for {agent.name}")
                else:
                    print(f"Ignoring event type: {event.item.type}")

        print("=== Run complete ===")
        return result

    async def run(self, alert: AlertGroup) -> None:
        trace_id = gen_trace_id()
        with trace("Root Cause Analysis", trace_id=trace_id):
            self.printer.update_item(
                "trace_id",
                f"View trace: https://platform.openai.com/traces/trace?trace_id={trace_id}",
                is_done=True,
                hide_checkmark=True,
            )

            self.printer.update_item(
                "starting",
                "Starting root cause analysis...",
                is_done=True,
                hide_checkmark=True,
            )

            with custom_span("Neo4j Service Graph Analysis"):
                affected_services_metadata: ServiceGraphResponse = await self.get_affected_services_metadata(alert)
                print(f"affected_services_metadata: {affected_services_metadata.final_output}")

            with custom_span("Kubernetes and Observe Log Analysis"):
                k8s_exploration, observe_exploration = await asyncio.gather(
                    self.explore_k8s(alert, affected_services_metadata.final_output),
                    self.explore_observe_logs(alert, affected_services_metadata.final_output)
                )
                print(f"k8s_exploration: {k8s_exploration.final_output}")
                print(f"observe_exploration: {observe_exploration.final_output}")

            with custom_span("Codebase Analysis"):
                codebase_analysis = await self.codebase_analysis(alert, affected_services_metadata.final_output, k8s_exploration.final_output, observe_exploration.final_output)
                print(f"codebase_analysis: {codebase_analysis.final_output}")

            with custom_span("RCA Summary"):
                rca_summary = await self.rca_summary(alert, affected_services_metadata.final_output, k8s_exploration.final_output, observe_exploration.final_output, codebase_analysis.final_output)
                print(f"rca_summary: {rca_summary.final_output}")

    async def get_affected_services_metadata(self, alert: AlertGroup) -> ServiceGraphResponse:
        """
        Get the metadata for the affected services.
        """
        self.printer.update_item("fetching", "Fetching downstream and upstream services...")
        graph_metadata = await self.wrap_agent_stream(neo4j_agent, alert.json())
        self.printer.mark_item_done("fetching")
        return graph_metadata

    async def explore_k8s(self, alert: AlertGroup, service_graph: ServiceGraphResponse) -> str:
        """
        Explore the k8s metadata for the affected services.
        """
        self.printer.update_item("k8s", "Exploring Kubernetes resources...")
        k8s_exploration = await self.wrap_agent_stream(k8s_agent, alert.json(), service_graph.json())
        self.printer.mark_item_done("k8s")
        return k8s_exploration

    async def explore_observe_logs(self, alert: AlertGroup, service_graph: ServiceGraphResponse) -> str:
        """
        Explore the Observe logs for the affected services.
        """
        self.printer.update_item("observe", "Exploring Observe logs...")
        observe_exploration = await self.wrap_agent_stream(observe_agent, alert.json(), service_graph.json())
        self.printer.mark_item_done("observe")
        return observe_exploration
    
    async def codebase_analysis(self, alert: AlertGroup, service_graph: ServiceGraphResponse, k8s_exploration: str, observe_exploration: str) -> str:
        """
        Analyze the codebase for the affected services.
        """
        self.printer.update_item("codebase", "Analyzing codebase...")
        codebase_analysis = await self.wrap_agent_stream(codebase_agent, alert.json(), service_graph.json(), k8s_exploration, observe_exploration)
        self.printer.mark_item_done("codebase")
        return codebase_analysis
    
    async def rca_summary(self, alert: AlertGroup, service_graph: ServiceGraphResponse, k8s_exploration: str, observe_exploration: str, codebase_analysis: str) -> str:
        """
        Generate a summary of the RCA.
        """
        self.printer.update_item("summary", "Generating RCA summary...")
        rca_summary = await self.wrap_agent_stream(rca_summary_agent, alert.json(), service_graph.json(), k8s_exploration, observe_exploration, codebase_analysis)
        self.printer.mark_item_done("summary")
        return rca_summary