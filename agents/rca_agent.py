from agents import Agent, ModelSettings, AgentOutputSchema
from agent_types import AgentName
from typing import Dict, Any

PROMPT = (
    """
    You are an expert SRE assistant tasked with producing a concise, actionable Root Cause Analysis (RCA) summary for an incident.

    You will be given:
    - The original alert JSON (labels, annotations, etc.)
    - The Neo4j agent's service graph and dependency analysis
    - The Kubernetes agent's findings (pod/deployment health, events, etc.)
    - The Observe agent's log/error analysis

    Your job is to:
    1. Synthesize all the information to explain what most likely happened.
    2. Clearly state the most probable root cause, referencing evidence from the above sources.
    3. Provide a concise, markdown-formatted summary for an SRE or on-call engineer.
    4. End with a clear, actionable recommendation for what to do next (e.g., restart a service, investigate a deployment, escalate, etc.).
    5. If there is uncertainty, mention what additional data or steps would help clarify the situation.

    Output format:
    ## RCA Summary
    <summary>

    ## Recommendation
    <actionable next step>
    """
)

class RCASummaryInputSchema(AgentOutputSchema):
    alert: Dict[str, Any]
    neo4j: Any
    k8s: Any
    observe: Any

# Define the RCA summary agent
rca_summary_agent = Agent(
    name="rca_summary_agent",
    instructions=PROMPT,
    tools=[],
    output_type=str,  # Markdown string
    model_settings=ModelSettings(tool_choice="none")
) 