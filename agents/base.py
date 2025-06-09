from typing import Any, Dict, List, Optional
from pydantic import BaseModel
from agents import Agent, Runner, function_tool

class AgentConfig(BaseModel):
    """Base configuration for agents"""
    name: str
    description: str
    enabled: bool = True
    config: Dict[str, Any] = {}

class BaseAgent:
    """Base class for all agents using OpenAI Agents SDK"""
    
    def __init__(self, config: AgentConfig):
        self.config = config
        self.name = config.name
        self.description = config.description
        self.enabled = config.enabled
        self.agent = self._create_agent()

    def _create_agent(self) -> Agent:
        """Create the OpenAI Agent instance"""
        return Agent(
            name=self.name,
            instructions=self.description,
            tools=self._get_tools()
        )

    def _get_tools(self) -> List[Any]:
        """Get the tools for this agent"""
        return []

    async def process(self, input_data: Dict[str, Any]) -> Dict[str, Any]:
        """Process input data using the OpenAI Agent"""
        if not self.enabled:
            return {"status": "disabled"}
        
        result = await Runner.run(self.agent, str(input_data))
        return {"status": "success", "result": result.final_output}

    def is_enabled(self) -> bool:
        """Check if the agent is enabled"""
        return self.enabled 