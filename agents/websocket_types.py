from enum import Enum, auto
from typing import TypedDict, Any, Literal

class AgentType(str, Enum):
    """Types of agents in the system."""
    NEO4J = "neo4j"
    OBSERVE = "observe"
    REPORT = "report"

class WebSocketMessageType(str, Enum):
    """Types of messages that can be sent over the WebSocket."""
    AGENT_STARTED = "agent_started"
    AGENT_UPDATED = "agent_updated"
    TOOL_CALL = "tool_call"
    TOOL_OUTPUT = "tool_output"
    MESSAGE_OUTPUT = "message_output"
    STATUS = "status"
    ERROR = "error"

class BaseMessage(TypedDict):
    """Base structure for all WebSocket messages."""
    type: WebSocketMessageType
    agent: AgentType | None  # None for system messages like status/error
    data: Any

class StatusMessage(BaseMessage):
    """Message indicating processing status."""
    type: Literal[WebSocketMessageType.STATUS]
    agent: None
    data: str

class ErrorMessage(BaseMessage):
    """Message indicating an error occurred."""
    type: Literal[WebSocketMessageType.ERROR]
    agent: None
    data: str

class AgentStartedMessage(BaseMessage):
    """Message indicating an agent has started."""
    type: Literal[WebSocketMessageType.AGENT_STARTED]
    agent: AgentType
    data: str  # agent name

class AgentUpdatedMessage(BaseMessage):
    """Message indicating an agent has been updated."""
    type: Literal[WebSocketMessageType.AGENT_UPDATED]
    agent: AgentType
    data: dict  # agent data

class ToolCallMessage(BaseMessage):
    """Message indicating a tool was called."""
    type: Literal[WebSocketMessageType.TOOL_CALL]
    agent: AgentType
    data: str  # tool call description

class ToolOutputMessage(BaseMessage):
    """Message containing tool output."""
    type: Literal[WebSocketMessageType.TOOL_OUTPUT]
    agent: AgentType
    data: Any  # tool output

class MessageOutputMessage(BaseMessage):
    """Message containing agent message output."""
    type: Literal[WebSocketMessageType.MESSAGE_OUTPUT]
    agent: AgentType
    data: str  # message content 