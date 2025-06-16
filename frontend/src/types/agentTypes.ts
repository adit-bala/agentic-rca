export enum AgentName {
  NEO4J = "Neo4jAgent",
  K8S = "K8sAgent",
  OBSERVE = "ObserveLogAgent",
  GITHUB = "GithubAgent",
  REPORT = "ReportAgent"
}

export enum AgentType {
  NEO4J = "neo4j",
  K8S = "k8s",
  OBSERVE = "observe",
  GITHUB = "github",
  REPORT = "report"
}

export enum WebSocketMessageType {
  AGENT_STARTED = "agent_started",
  AGENT_UPDATED = "agent_updated",
  TOOL_CALL = "tool_call",
  TOOL_OUTPUT = "tool_output",
  MESSAGE_OUTPUT = "message_output",
  STATUS = "status",
  ERROR = "error"
}

// Mapping from agent names to their types
export const AGENT_TYPE_MAP: Record<AgentName, AgentType> = {
  [AgentName.NEO4J]: AgentType.NEO4J,
  [AgentName.K8S]: AgentType.K8S,
  [AgentName.OBSERVE]: AgentType.OBSERVE,
  [AgentName.GITHUB]: AgentType.GITHUB,
  [AgentName.REPORT]: AgentType.REPORT
}

// Type definitions for WebSocket messages
export interface BaseMessage {
  type: WebSocketMessageType;
  agent: AgentType | null;  // null for system messages like status/error
  data: unknown;
}

export interface StatusMessage extends BaseMessage {
  type: WebSocketMessageType.STATUS;
  agent: null;
  data: string;
}

export interface ErrorMessage extends BaseMessage {
  type: WebSocketMessageType.ERROR;
  agent: null;
  data: string;
}

export interface AgentStartedMessage extends BaseMessage {
  type: WebSocketMessageType.AGENT_STARTED;
  agent: AgentType;
  data: string;  // agent name
}

export interface AgentUpdatedMessage extends BaseMessage {
  type: WebSocketMessageType.AGENT_UPDATED;
  agent: AgentType;
  data: Record<string, unknown>;  // agent data
}

export interface ToolCallMessage extends BaseMessage {
  type: WebSocketMessageType.TOOL_CALL;
  agent: AgentType;
  data: string;  // tool call description
}

export interface ToolOutputMessage extends BaseMessage {
  type: WebSocketMessageType.TOOL_OUTPUT;
  agent: AgentType;
  data: unknown;  // tool output
}

export interface MessageOutputMessage extends BaseMessage {
  type: WebSocketMessageType.MESSAGE_OUTPUT;
  agent: AgentType;
  data: string;  // message content
} 