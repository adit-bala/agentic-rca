from typing import Any, Dict, List
from agents import Agent, Runner, function_tool
from .base import BaseAgent, AgentConfig

class OrchestratorAgentConfig(AgentConfig):
    """Configuration for the orchestrator agent"""
    max_retries: int = 3
    retry_delay: int = 5

class OrchestratorAgent(BaseAgent):
    """Orchestrator agent that manages task creation and monitoring"""
    
    def __init__(self, config: OrchestratorAgentConfig):
        super().__init__(config)
        self.max_retries = config.max_retries
        self.retry_delay = config.retry_delay
        
        # Initialize sub-agents
        self.git_agent = GitAgent(GitAgentConfig(
            name="git_agent",
            description="Analyzes repository commits and changes"
        ))
        self.log_agent = LogAgent(LogAgentConfig(
            name="log_agent",
            description="Analyzes service logs using Loki/Observe"
        ))
        self.k8s_agent = K8sAgent(K8sAgentConfig(
            name="k8s_agent",
            description="Checks pod health using client-go"
        ))

    def _get_tools(self) -> List[Any]:
        """Get the tools for this agent"""
        return [
            self.create_graph_task,
            self.create_git_task,
            self.create_log_task,
            self.create_k8s_task,
            self.monitor_tasks
        ]

    @function_tool
    def create_graph_task(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create task for Graph agent"""
        return {
            "type": "graph_analysis",
            "services": input_data.get("affected_services", []),
            "time_range": input_data.get("time_range", "1h")
        }

    @function_tool
    def create_git_task(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create task for Git agent"""
        return {
            "type": "git_analysis",
            "repos": input_data.get("repos", []),
            "time_range": input_data.get("time_range", "1h")
        }

    @function_tool
    def create_log_task(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create task for Log agent"""
        return {
            "type": "log_analysis",
            "services": input_data.get("affected_services", []),
            "time_range": input_data.get("time_range", "1h")
        }

    @function_tool
    def create_k8s_task(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create task for K8s agent"""
        return {
            "type": "pod_health_check",
            "namespaces": input_data.get("namespaces", []),
            "services": input_data.get("affected_services", [])
        }

    @function_tool
    async def monitor_tasks(self, tasks: Dict[str, Dict[str, Any]]) -> Dict[str, Any]:
        """Monitor task execution and collect results"""
        results = {}
        for agent_type, task in tasks.items():
            if agent_type == "graph":
                results[agent_type] = await self.graph_agent.process(task)
            elif agent_type == "git":
                results[agent_type] = await self.git_agent.process(task)
            elif agent_type == "log":
                results[agent_type] = await self.log_agent.process(task)
            elif agent_type == "k8s":
                results[agent_type] = await self.k8s_agent.process(task)
        return results

    async def process(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        """Process alert and create tasks for other agents"""
        alert_id = input_data.get("alert_id")
        if not alert_id:
            raise ValueError("alert_id is required in input_data")

        # Create tasks for each agent
        tasks = {
            "graph": self.create_graph_task(input_data),
            "git": self.create_git_task(input_data),
            "log": self.create_log_task(input_data),
            "k8s": self.create_k8s_task(input_data)
        }

        # Monitor task execution
        results = await self.monitor_tasks(tasks)

        return {
            "status": "success",
            "alert_id": alert_id,
            "tasks_created": list(tasks.keys()),
            "results": results
        } 