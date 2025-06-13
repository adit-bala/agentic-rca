'use client';

import { useEffect, useState } from 'react';
import { BaseMessage, WebSocketMessageType } from '@/types/agentTypes';
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

interface AlertMessages {
  [alertName: string]: BaseMessage[];
}

export default function Alerts() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [websocketIds, setWebsocketIds] = useState<Record<string, string>>({});
  const [websockets, setWebsockets] = useState<Record<string, WebSocket>>({});
  const [messages, setMessages] = useState<AlertMessages>({});
  const [expandedAlerts, setExpandedAlerts] = useState<Record<string, boolean>>({});

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

    // Initial fetch
    fetchAlerts();

    // Poll for updates every 10 seconds
    const interval = setInterval(fetchAlerts, 10000);

    return () => {
      clearInterval(interval);
      // Close all WebSocket connections
      Object.values(websockets).forEach(ws => ws.close());
    };
  }, [websockets]);

  const startRCA = async (alert: Alert) => {
    try {
      // Create alert group payload
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

      // Step 1: Send to /alerts endpoint to get WebSocket ID
      const response = await fetch('http://localhost:8001/alerts', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(alertGroup),
      });

      if (!response.ok) throw new Error('Failed to start RCA');
      
      const data = await response.json();
      const wsId = data.websocket_id;

      if (!wsId) {
        throw new Error('No WebSocket ID received from server');
      }

      // Store websocket ID for this alert
      setWebsocketIds(prev => ({
        ...prev,
        [alert.labels.alertname]: wsId
      }));

      // Step 2: Create WebSocket connection to receive updates
      // The WebSocket connection itself initiates the processing
      const ws = new WebSocket(`ws://localhost:8001/process/${wsId}`);
      
      ws.onmessage = (event) => {
        try {
          const message: BaseMessage = JSON.parse(event.data);
          setMessages(prev => ({
            ...prev,
            [alert.labels.alertname]: [...(prev[alert.labels.alertname] || []), message]
          }));
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        // Clean up the websocket ID on error
        setWebsocketIds(prev => {
          const newIds = { ...prev };
          delete newIds[alert.labels.alertname];
          return newIds;
        });
      };

      ws.onclose = () => {
        console.log('WebSocket connection closed');
        // Clean up the websocket ID when connection closes
        setWebsocketIds(prev => {
          const newIds = { ...prev };
          delete newIds[alert.labels.alertname];
          return newIds;
        });
      };

      setWebsockets(prev => ({
        ...prev,
        [alert.labels.alertname]: ws
      }));

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start RCA');
      // Clean up the websocket ID on error
      setWebsocketIds(prev => {
        const newIds = { ...prev };
        delete newIds[alert.labels.alertname];
        return newIds;
      });
    }
  };

  // Helper to get badge color and label based on message type or likelihood
  const getStatusBadge = (message: BaseMessage) => {
    // You can expand this logic if you have more granular likelihoods in your data
    if (typeof message.data === 'string') {
      if (message.data.toLowerCase().includes('very likely')) {
        return { label: 'Very Likely', color: 'bg-[#FF6B00] text-white' };
      }
      if (message.data.toLowerCase().includes('unlikely')) {
        return { label: 'Unlikely', color: 'bg-[#B6E900] text-black' };
      }
      if (message.data.toLowerCase().includes('very unlikely')) {
        return { label: 'Very Unlikely', color: 'bg-[#B6E900] text-black' };
      }
    }
    if (message.type === WebSocketMessageType.ERROR) {
      return { label: 'Error', color: 'bg-red-500 text-white' };
    }
    if (message.type === WebSocketMessageType.STATUS) {
      return { label: 'Status', color: 'bg-gray-200 text-black' };
    }
    return { label: message.type, color: 'bg-gray-200 text-black' };
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
              className="p-0 rounded-lg border border-gray-200 shadow-lg bg-white max-w-2xl mx-auto"
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
                  disabled={!!websocketIds[alert.labels.alertname]}
                  className={`px-4 py-2 rounded font-semibold transition-colors duration-200 shadow ${
                    websocketIds[alert.labels.alertname]
                      ? 'bg-gray-300 cursor-not-allowed text-gray-600'
                      : 'bg-[#B6E900] text-black hover:bg-[#D4FF3D]'
                  }`}
                >
                  {websocketIds[alert.labels.alertname] ? 'RCA in Progress...' : 'Start Root Cause Analysis'}
                </button>

                {/* Messages Dropdown */}
                {messages[alert.labels.alertname]?.length > 0 && (
                  <div className="mt-4 space-y-3">
                    {messages[alert.labels.alertname].map((message, idx) => {
                      const badge = getStatusBadge(message);
                      return (
                        <div
                          key={idx}
                          className="rounded-lg border border-gray-200 bg-white shadow flex flex-col"
                          style={{ borderLeft: `6px solid ${badge.label === 'Very Likely' ? '#FF6B00' : badge.label.includes('Unlikely') ? '#B6E900' : '#D1D5DB'}` }}
                        >
                          <div className="flex items-center justify-between px-4 py-3 cursor-pointer select-none"
                            onClick={() => setExpandedAlerts(prev => ({
                              ...prev,
                              [`${alert.labels.alertname}-${idx}`]: !prev[`${alert.labels.alertname}-${idx}`]
                            }))}
                          >
                            <div className="flex items-center gap-2">
                              <span className={`px-2 py-1 rounded text-xs font-bold uppercase ${badge.color}`}>{badge.label}</span>
                              <span className="font-medium text-black">
                                {message.type === WebSocketMessageType.STATUS ? 'Status Update' : 
                                 message.type === WebSocketMessageType.ERROR ? 'Error' : 
                                 message.type}
                              </span>
                            </div>
                            <button
                              className="ml-2 px-3 py-1 rounded bg-[#B6E900] text-[#F3FF3D] font-semibold text-xs shadow hover:bg-[#D4FF3D] transition-colors duration-200"
                              tabIndex={-1}
                            >
                              {expandedAlerts[`${alert.labels.alertname}-${idx}`] ? 'Hide Details' : 'Show Details'}
                            </button>
                          </div>
                          {expandedAlerts[`${alert.labels.alertname}-${idx}`] && (
                            <div className="px-4 pb-4">
                              {message.type === WebSocketMessageType.MESSAGE_OUTPUT ? (
                                <div className="prose prose-sm max-w-none bg-gray-50 rounded p-3 border border-gray-100 mt-2 text-gray-800">
                                  <ReactMarkdown>{String(message.data)}</ReactMarkdown>
                                </div>
                              ) : (
                                <pre className="whitespace-pre-wrap text-sm text-gray-800 bg-gray-50 rounded p-3 border border-gray-100 mt-2">
                                  {JSON.stringify(message.data, null, 2)}
                                </pre>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    })}
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