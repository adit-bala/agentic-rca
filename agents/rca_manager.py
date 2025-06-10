import asyncio

from rich.console import Console

from agents import Runner, custom_span, gen_trace_id, trace
from printer import Printer
from neo4j_agent import neo4j_agent

from models import AlertGroup

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

            affected_services_metadata = await self.get_affected_services_metadata(alert)
            self.printer.update_item("fetching", "Fetching downstream and upstream services...", is_done=True)
            

    async def get_affected_services_metadata(self, alert: AlertGroup) -> list[dict]:
        """
        Get the metadata for the affected services.
        """
        self.printer.update_item("fetching", "Fetching downstream and upstream services...")
        graph_metadata = await Runner().run(
            neo4j_agent,
            alert.json()
        )
        self.printer.update_item("fetching", "Fetching downstream and upstream services...", is_done=True)
        return graph_metadata
        


rca_manager = RCA_Manager()