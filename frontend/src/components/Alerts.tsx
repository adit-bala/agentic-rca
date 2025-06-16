'use client';

import { useEffect, useState } from 'react';
import { AgentType, WebSocketMessageType } from '@/types/agentTypes';
import ReactMarkdown from 'react-markdown';

interface Alert {
  status: string;
  labels: {
    alertname: string;
    severity: string;
    service: string;
    [key: string]: string;
  };
  annotations: {
    summary: string;
    description: string;
  };
  receivedAt: string;
}

interface AlertGroup {
  version: string;
  groupKey: string;
  status: string;
  receiver: string;
  groupLabels: Record<string, string>;
  commonLabels: Record<string, string>;
  commonAnnotations: Record<string, string>;
  externalURL: string;
  alerts: Alert[];
}

// Types for the graph data
interface K8sInfo {
  namespace: string;
  labels: Record<string, string> | null;
  annotations: Record<string, string> | null;
  owner_kind: string;
  owner_name: string;
  owner_uid: string;
}
interface ServiceNode {
  name: string;
  k8s: K8sInfo;
  operation: string | null;
  attributes: Record<string, string> | null;
}
interface ServiceGraph {
  current: ServiceNode;
  upstream: ServiceNode[];
  downstream: ServiceNode[];
}
interface Neo4jGraphData {
  services: ServiceGraph[];
}

// Add after Neo4jGraphData
interface MessageOutputEntry {
  agent: string;
  message: string;
}

// Restore ToolCallEntry and toolCalls state
interface ToolCallEntry {
  agent: string;
  function_name: string;
  arguments: string;
  status: 'pending' | 'done';
  output?: string;
}

// Add a simple graph rendering component
function ServiceDependencyGraph({ services }: { services: ServiceGraph[] }) {
  if (!services || services.length === 0) return null;

  // For now, only support a single service object as in the example
  const { current, upstream, downstream } = services[0];

  // Build nodes array: upstream(s), current, downstream(s)
  // Layout: upstream on left, current center, downstream right
  const nodeRadius = 32;
  const width = 800;
  const height = 220;
  const centerY = height / 2;
  const leftX = 160;
  const centerX = width / 2;
  const rightX = width - 160;

  // Assign positions
  const upstreamNodes = upstream.map((node, i) => ({
    ...node,
    x: leftX,
    y: centerY + (i - (upstream.length - 1) / 2) * 80,
    type: 'upstream',
  }));
  const currentNode = {
    ...current,
    x: centerX,
    y: centerY,
    type: 'current',
  };
  const downstreamNodes = downstream.map((node, i) => ({
    ...node,
    x: rightX,
    y: centerY + (i - (downstream.length - 1) / 2) * 80,
    type: 'downstream',
  }));

  // Helper to render a node
  const renderNode = (node: ServiceNode & { x: number; y: number; type: string }) => {
    let stroke, fill, ringStroke;
    if (node.type === 'current') {
      stroke = '#FF6B00';
      fill = '#fff';
      ringStroke = '#FF6B00';
    } else {
      stroke = '#B6E900';
      fill = '#fff';
      ringStroke = '#F3FF3D';
    }
    return (
      <g key={node.name}>
        {/* Outer ring */}
        <circle cx={node.x} cy={node.y} r={nodeRadius + 6} fill="none" stroke={ringStroke} strokeWidth={6} />
        {/* Inner ring */}
        <circle cx={node.x} cy={node.y} r={nodeRadius} fill={fill} stroke={stroke} strokeWidth={3} />
        {/* Node name */}
        <text
          x={node.x}
          y={node.y + nodeRadius + 22}
          textAnchor="middle"
          fontWeight={node.type === 'current' ? 'bold' : 'normal'}
          fontSize={16}
          fill="#222"
        >
          {node.name}
        </text>
      </g>
    );
  };

  // Helper to render a directed edge
  const renderEdge = (
    from: ServiceNode & { x: number; y: number; type: string },
    to: ServiceNode & { x: number; y: number; type: string }
  ) => (
    <g key={`${from.name}->${to.name}`}>
      <line
        x1={from.x + (to.x > from.x ? nodeRadius : -nodeRadius)}
        y1={from.y}
        x2={to.x + (from.x > to.x ? nodeRadius : -nodeRadius)}
        y2={to.y}
        stroke="#FF6B00"
        strokeWidth={2}
        markerEnd="url(#arrow-red)"
      />
      <text
        x={(from.x + to.x) / 2}
        y={(from.y + to.y) / 2 - 10}
        textAnchor="middle"
        fontSize={12}
        fill="#FF6B00"
        fontWeight="bold"
      >
        CALLS
      </text>
    </g>
  );

  return (
    <svg width={width} height={height} style={{ display: 'block', margin: '0 auto' }}>
      {/* Edges: upstream -> current, current -> downstream */}
      {upstreamNodes.map(node => renderEdge(node, currentNode))}
      {downstreamNodes.map(node => renderEdge(currentNode, node))}
      {/* Nodes */}
      {upstreamNodes.map(renderNode)}
      {renderNode(currentNode)}
      {downstreamNodes.map(renderNode)}
      {/* Arrow marker definition */}
      <defs>
        <marker id="arrow-red" markerWidth="10" markerHeight="10" refX="10" refY="5" orient="auto" markerUnits="strokeWidth">
          <path d="M0,0 L10,5 L0,10 Z" fill="#FF6B00" />
        </marker>
      </defs>
    </svg>
  );
}

export default function Alerts() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [websocketIds, setWebsocketIds] = useState<Record<string, string>>({});
  const [websockets, setWebsockets] = useState<Record<string, WebSocket>>({});
  const [graphData, setGraphData] = useState<Neo4jGraphData | null>(null);
  const [investigating, setInvestigating] = useState<Record<string, boolean>>({});
  const [messageOutputs, setMessageOutputs] = useState<MessageOutputEntry[]>([]);
  const [expandedMessageOutput, setExpandedMessageOutput] = useState<number | null>(null);
  const [toolCalls, setToolCalls] = useState<ToolCallEntry[]>([]);
  const [expandedToolCall, setExpandedToolCall] = useState<number | null>(null);

  useEffect(() => {
    const fetchAlerts = async () => {
      try {
        const response = await fetch('/api/alerts/webhook');
        if (!response.ok) throw new Error('Failed to fetch alerts');
        const data = await response.json();
        setAlerts(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch alerts');
      }
    };
    fetchAlerts();
    const interval = setInterval(fetchAlerts, 10000);
    return () => {
      clearInterval(interval);
      Object.values(websockets).forEach(ws => ws.close());
    };
  }, [websockets]);

  const startRCA = async (alert: Alert) => {
    setInvestigating(prev => ({ ...prev, [alert.labels.alertname]: true }));
    try {
      const alertGroup: AlertGroup = {
        version: "4",
        groupKey: `{}:{alertname="${alert.labels.alertname}"}`,
        status: alert.status,
        receiver: "webhook-receiver",
        groupLabels: { alertname: alert.labels.alertname },
        commonLabels: alert.labels,
        commonAnnotations: alert.annotations,
        externalURL: "",
        alerts: [alert]
      };
      const response = await fetch('http://localhost:8001/alerts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(alertGroup),
      });
      if (!response.ok) throw new Error('Failed to start RCA');
      const data = await response.json();
      const wsId = data.websocket_id;
      if (!wsId) throw new Error('No WebSocket ID received from server');
      setWebsocketIds(prev => ({ ...prev, [alert.labels.alertname]: wsId }));
      const ws = new WebSocket(`ws://localhost:8001/process/${wsId}`);
      ws.onmessage = (event) => {
        console.log('WebSocket message:', event);
        try {
          const message = JSON.parse((event as MessageEvent).data);
          if (message.type === WebSocketMessageType.TOOL_CALL) {
            setToolCalls(prev => [
              ...prev,
              {
                agent: message.agent,
                function_name: message.data.function_name,
                arguments: message.data.arguments,
                status: 'pending',
              }
            ]);
          }
          if (message.type === WebSocketMessageType.TOOL_OUTPUT) {
            setToolCalls(prev => prev.map(tc =>
              tc.agent === message.agent && tc.function_name === (message.data?.function_name || tc.function_name)
                ? { ...tc, status: 'done', output: typeof message.data === 'string' ? message.data : JSON.stringify(message.data) }
                : tc
            ));
          }
          if (message.type === WebSocketMessageType.MESSAGE_OUTPUT) {
            setMessageOutputs(prev => [
              ...prev,
              {
                agent: message.agent,
                message: message.data,
              }
            ]);
          }
          // Only display MESSAGE_OUTPUT from neo4j agent with 'services' array
          if (
            message.type === WebSocketMessageType.MESSAGE_OUTPUT &&
            message.agent === AgentType.NEO4J &&
            message.data &&
            typeof message.data === 'string'
          ) {
            let parsed;
            try {
              parsed = JSON.parse(message.data);
            } catch {
              // Not JSON, ignore
              return;
            }
            if (parsed && Array.isArray(parsed.services) && parsed.services.length > 0) {
              setGraphData(parsed);
            }
          } else {
            // For all other messages, just log
            console.log('WebSocket message:', message);
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };
      ws.onerror = (error: Event) => {
        console.error('WebSocket error:', error);
        setWebsocketIds(prev => {
          const newIds = { ...prev };
          delete newIds[alert.labels.alertname];
          return newIds;
        });
        setInvestigating(prev => {
          const newInv = { ...prev };
          delete newInv[alert.labels.alertname];
          return newInv;
        });
      };
      ws.onclose = () => {
        setWebsocketIds(prev => {
          const newIds = { ...prev };
          delete newIds[alert.labels.alertname];
          return newIds;
        });
        setInvestigating(prev => {
          const newInv = { ...prev };
          delete newInv[alert.labels.alertname];
          return newInv;
        });
      };
      setWebsockets(prev => ({ ...prev, [alert.labels.alertname]: ws }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start RCA');
      setWebsocketIds(prev => {
        const newIds = { ...prev };
        delete newIds[alert.labels.alertname];
        return newIds;
      });
      setInvestigating(prev => {
        const newInv = { ...prev };
        delete newInv[alert.labels.alertname];
        return newInv;
      });
    }
  };

  if (error) {
    return <div className="text-red-500">Error: {error}</div>;
  }

  return (
    <div className="min-h-screen py-8 px-4" style={{ background: '#F3FF3D' }}>
      <h2 className="text-2xl font-bold mb-4 text-black">System Alerts</h2>
      <div className="space-y-4">
        {alerts.length === 0 ? (
          <p className="text-gray-700">No active alerts</p>
        ) : (
          alerts.map((alert, index) => (
            <div
              key={index}
              className="p-0 rounded-lg border border-gray-200 shadow-lg bg-white w-full"
              style={{ borderLeft: `8px solid ${alert.labels.severity === 'critical' ? '#FF6B00' : '#B6E900'}` }}
            >
              <div className="flex justify-between items-start p-4">
                <div>
                  <h3 className="font-semibold text-lg text-black">
                    {alert.labels.alertname}
                  </h3>
                  <p className="text-sm text-gray-600">
                    Service: {alert.labels.service}
                  </p>
                  <p className="mt-2 text-gray-800">{alert.annotations.description}</p>
                </div>
                <div className="text-right">
                  <span
                    className={`px-2 py-1 rounded text-sm font-semibold ${
                      alert.labels.severity === 'critical'
                        ? 'bg-[#FF6B00] text-white'
                        : 'bg-[#B6E900] text-black'
                    }`}
                  >
                    {alert.labels.severity}
                  </span>
                  <p className="text-xs text-gray-500 mt-1">
                    {new Date(alert.receivedAt).toLocaleString()}
                  </p>
                </div>
              </div>
              {/* RCA Button */}
              <div className="p-4 pt-0">
                <button
                  onClick={() => startRCA(alert)}
                  disabled={!!websocketIds[alert.labels.alertname] || investigating[alert.labels.alertname]}
                  className={`px-4 py-2 rounded font-semibold transition-colors duration-200 shadow ${
                    (!!websocketIds[alert.labels.alertname] || investigating[alert.labels.alertname])
                      ? 'bg-gray-300 cursor-not-allowed text-gray-600'
                      : 'bg-[#B6E900] text-black hover:bg-[#D4FF3D]'
                  }`}
                >
                  {investigating[alert.labels.alertname] || websocketIds[alert.labels.alertname]
                    ? 'Investigating Root Cause Analysis'
                    : 'Start Root Cause Analysis'}
                </button>
                 {/* Service Dependency Graph */}
                 {graphData && graphData.services && graphData.services.length > 0 && (
                  <div className="mt-6">
                    <h4 className="font-semibold text-black mb-2 text-xl">Service Dependency Graph</h4>
                    <ServiceDependencyGraph services={graphData.services} />
                  </div>
                )}
                {/* Final Report (agent === AgentType.REPORT) below the graph */}
                {messageOutputs.filter(msg => msg.agent === AgentType.REPORT).length > 0 && (
                  <div className="mt-6">
                    {messageOutputs.filter(msg => msg.agent === AgentType.REPORT).map((msg, i) => (
                      <div key={i} className="bg-gray-50 p-6 rounded text-black mt-4">
                        <div className="prose prose-2xl max-w-none text-black" style={{ fontSize: '1.35rem' }}>
                          <ReactMarkdown>{msg.message}</ReactMarkdown>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
                {/* Tool Call Dropdowns */}
                {toolCalls.length > 0 && (
                  <div className="mt-4 space-y-2">
                    {toolCalls.map((toolCall, i) => (
                      <div key={i} className="border border-gray-300 rounded">
                        <button
                          className="w-full text-left px-4 py-2 font-mono bg-gray-100 hover:bg-gray-200 rounded-t focus:outline-none text-black flex items-center justify-between"
                          onClick={() => setExpandedToolCall(expandedToolCall === i ? null : i)}
                        >
                          <span>
                            {toolCall.agent} called {toolCall.function_name}(
                            {toolCall.arguments}
                            )
                          </span>
                          {toolCall.status === 'pending' && (
                            <svg className="animate-spin ml-2" width="18" height="18" viewBox="0 0 24 24">
                              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="black" strokeWidth="4" fill="none" />
                              <path className="opacity-75" fill="black" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
                            </svg>
                          )}
                        </button>
                        {expandedToolCall === i && toolCall.status === 'done' && (
                          <pre className="bg-gray-50 text-xs p-4 overflow-x-auto rounded-b text-black">
                            {toolCall.output}
                          </pre>
                        )}
                      </div>
                    ))}
                  </div>
                )}
                {/* Message Output Dropdowns (other agents) */}
                {messageOutputs.filter(msg => msg.agent !== AgentType.REPORT).length > 0 && (
                  <div className="mt-4 space-y-2">
                    {messageOutputs.map((msg, i) => (
                      msg.agent === AgentType.REPORT ? null : (
                        <div key={i} className="border border-gray-300 rounded">
                          <button
                            className="w-full text-left px-4 py-2 bg-gray-100 hover:bg-gray-200 rounded-t focus:outline-none text-black font-bold text-lg"
                            onClick={() => setExpandedMessageOutput(expandedMessageOutput === i ? null : i)}
                          >
                            {msg.agent} report
                          </button>
                          {expandedMessageOutput === i && (
                            <div className="bg-gray-50 text-black p-4 rounded-b">
                              <div className="prose prose-xl max-w-none text-black" style={{ fontSize: '1.15rem' }}>
                                <ReactMarkdown>{msg.message}</ReactMarkdown>
                              </div>
                            </div>
                          )}
                        </div>
                      )
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
} 