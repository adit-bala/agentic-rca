import asyncio

from rich.console import Console

from agents import Runner, custom_span, gen_trace_id, trace
from printer import Printer
from neo4j_agent import neo4j_agent
from k8s_agent import k8s_agent
from models import AlertGroup, ServiceGraphResponse

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

            with custom_span("Kubernetes Cluster Analysis"):
                k8s_exploration = await self.explore_k8s(affected_services_metadata.final_output)

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
    
    async def explore_k8s(self, service_graph: ServiceGraphResponse) -> str:
        """
        Explore the k8s metadata for the affected services.
        """
        self.printer.update_item("exploring", f"Exploring k8s metadata for relevant services...")
        k8s_exploration = await Runner().run(
            k8s_agent,
            service_graph.model_dump_json()
        )
        self.printer.mark_item_done("exploring")
        return k8s_exploration

rca_manager = RCA_Manager()