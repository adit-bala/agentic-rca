from enum import Enum

class AgentName(str, Enum):
    """Names of agents in the system."""
    NEO4J = "Neo4jAgent"
    K8S = "K8sAgent"
    OBSERVE = "ObserveLogAgent"
    GITHUB = "GithubAgent"
    REPORT = "ReportAgent"

class AgentType(str, Enum):
    """Types of agents in the system."""
    NEO4J = "neo4j"
    K8S = "k8s"
    OBSERVE = "observe"
    GITHUB = "github"
    REPORT = "report"

# Mapping from agent names to their types
AGENT_TYPE_MAP = {
    AgentName.NEO4J: AgentType.NEO4J,
    AgentName.K8S: AgentType.K8S,
    AgentName.OBSERVE: AgentType.OBSERVE,
    AgentName.GITHUB: AgentType.GITHUB,
    AgentName.REPORT: AgentType.REPORT
} 