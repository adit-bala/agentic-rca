from __future__ import annotations

import os
from typing import Dict, List, Optional
from agents import Agent, ModelSettings, function_tool
from agent_types import AgentName
from indexing.app import answer_query

PROMPT = (
    "You are a **Codebase Analysis agent** for on-call engineers.\n\n"
    "Inputs\n"
    "------\n"
    "• **Alert JSON** → one or more firing alerts (label key-values, description, etc.)\n"
    "• **Service graph JSON** → for each alert-impacted service:\n"
    "  ▸ `current`  : focal service (name + k8s metadata)\n"
    "  ▸ `upstream` : services this node calls\n"
    "  ▸ `downstream`: services that call this node\n"
    "• **K8s Exploration** → findings from Kubernetes analysis including:\n"
    "  ▸ Pod status and conditions\n"
    "  ▸ Deployment/ReplicaSet health\n"
    "  ▸ Recent events and warnings\n"
    "• **Observe Logs** → findings from log analysis including:\n"
    "  ▸ Error patterns and stack traces\n"
    "  ▸ Performance metrics and anomalies\n"
    "  ▸ Recent log events and warnings\n\n"
    "Goal\n"
    "----\n"
    "Form and test hypotheses about potential root causes by:\n"
    "1. Analyzing K8s/Observe findings to form initial hypotheses\n"
    "2. Using the query_codebase tool to test these hypotheses:\n"
    "   • The tool sends your hypothesis to a specialized codebase agent\n"
    "   • This agent has deep knowledge of the codebase and recent commits\n"
    "   • It returns relevant code snippets, recent changes, and its analysis\n"
    "3. Refining hypotheses based on the codebase agent's findings\n"
    "4. Correlating code patterns with observed issues\n"
    "5. Providing actionable insights and recommendations\n\n"
    "When composing your answer:\n"
    "  1. Map alert labels → service graph nodes → potential code paths\n"
    "  2. Form specific hypotheses about what might be wrong\n"
    "  3. Use query_codebase to test each hypothesis:\n"
    "     • Frame queries that would confirm/deny your hypothesis\n"
    "     • Example: 'Find code related to error handling in user authentication'\n"
    "  4. Analyze the codebase agent's response\n"
    "  5. Propose the most plausible root cause and next steps\n"
    " LIMIT TO 3 tool calls per response"
)

@function_tool
def query_codebase(query: str) -> str:
    """
    Query the codebase for relevant code snippets and recent changes.
    The query should be specific and focused on the potential issue.
    Example: "Find code related to error handling in user authentication"
    """
    print(f"query_codebase: {query}")
    return answer_query(query)

codebase_agent = Agent(
    name=AgentName.GITHUB,
    instructions=PROMPT,
    tools=[query_codebase],
    model_settings=ModelSettings(
        tool_choice="required"
    )
)
