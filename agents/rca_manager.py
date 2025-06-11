import asyncio

from rich.console import Console

from agents import Runner, custom_span, gen_trace_id, trace
from printer import Printer
from neo4j_agent import neo4j_agent
from k8s_agent import k8s_agent
from observe_agent import observe_agent
from models import AlertGroup, ServiceGraphResponse

def gather_inputs(*contents: str) -> list[dict]:
    """
    Helper to format multiple strings as a list of agent-compatible user message dicts.
    """
    return [{"role": "user", "content": content} for content in contents]

class RCA_Manager:
    def __init__(self):
        self.console = Console()
        self.printer = Printer(self.console)

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
                print(affected_services_metadata)

            with custom_span("Kubernets and Observe Log Analysis"):
                k8s_exploration, observe_exploration = await asyncio.gather(
                    self.explore_k8s(alert, affected_services_metadata.final_output),
                    self.explore_observe_logs(alert, affected_services_metadata.final_output)
                )
                print(k8s_exploration.final_output, observe_exploration.final_output)

    async def get_affected_services_metadata(self, alert: AlertGroup) -> ServiceGraphResponse:
        """
        Get the metadata for the affected services.
        """
        self.printer.update_item("fetching", "Fetching downstream and upstream services...")
        graph_metadata = await Runner().run(
            neo4j_agent,
            alert.json()
        )
        self.printer.mark_item_done("fetching")
        return graph_metadata

    async def explore_k8s(self, alert: AlertGroup, service_graph: ServiceGraphResponse) -> str:
        """
        Explore the k8s metadata for the affected services.
        """
        self.printer.update_item("exploring", f"Exploring k8s metadata for relevant services...")
        k8s_exploration = await Runner().run(k8s_agent, gather_inputs(alert.json(), service_graph.model_dump_json()))
        self.printer.mark_item_done("exploring")
        return k8s_exploration

    async def explore_observe_logs(self, alert: AlertGroup, service_graph: ServiceGraphResponse) -> str:
        """
        Explore the logs for the affected services.
        """
        self.printer.update_item("exploring", f"Exploring logs for relevant services...")
        logs_exploration = await Runner().run(
            observe_agent,
            gather_inputs(alert.json(), service_graph.model_dump_json())
        )
        self.printer.mark_item_done("exploring")
        return logs_exploration

rca_manager = RCA_Manager()
