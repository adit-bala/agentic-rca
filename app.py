from fastapi import FastAPI, HTTPException
from typing import Dict, Any, List
import uvicorn
from agents.base import AgentConfig
from agents.sample_agent import SampleAgent, SampleAgentConfig

app = FastAPI(title="Agent Infrastructure API")

# Store active agents
active_agents: Dict[str, Any] = {}

@app.on_event("startup")
async def startup_event():
    """Initialize agents on startup"""
    # Initialize sample agent
    sample_config = SampleAgentConfig(
        name="sample-agent",
        description="A sample agent implementation",
        model_name="gpt-3.5-turbo",
        temperature=0.7
    )
    sample_agent = SampleAgent(sample_config)
    await sample_agent.initialize()
    active_agents["sample-agent"] = sample_agent

@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup agents on shutdown"""
    for agent in active_agents.values():
        await agent.cleanup()

@app.get("/agents")
async def list_agents() -> List[Dict[str, Any]]:
    """List all available agents"""
    return [
        {
            "name": agent.name,
            "description": agent.description,
            "enabled": agent.is_enabled()
        }
        for agent in active_agents.values()
    ]

@app.post("/agents/{agent_name}/process")
async def process_with_agent(agent_name: str, input_data: Dict[str, Any]) -> Dict[str, Any]:
    """Process input data using the specified agent"""
    if agent_name not in active_agents:
        raise HTTPException(status_code=404, detail=f"Agent {agent_name} not found")
    
    agent = active_agents[agent_name]
    if not agent.is_enabled():
        raise HTTPException(status_code=400, detail=f"Agent {agent_name} is disabled")
    
    try:
        result = await agent.process(input_data)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    uvicorn.run("app:app", host="0.0.0.0", port=8000, reload=True) 