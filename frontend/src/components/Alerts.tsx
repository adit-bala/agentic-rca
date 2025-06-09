'use client';

import { useEffect, useState } from 'react';

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

export default function Alerts() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [error, setError] = useState<string | null>(null);

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

    return () => clearInterval(interval);
  }, []);

  if (error) {
    return <div className="text-red-500">Error: {error}</div>;
  }

  return (
    <div className="p-4">
      <h2 className="text-2xl font-bold mb-4">System Alerts</h2>
      <div className="space-y-4">
        {alerts.length === 0 ? (
          <p className="text-gray-500">No active alerts</p>
        ) : (
          alerts.map((alert, index) => (
            <div
              key={index}
              className={`p-4 rounded-lg border ${
                alert.labels.severity === 'critical'
                  ? 'bg-red-50 border-red-200'
                  : 'bg-yellow-50 border-yellow-200'
              }`}
            >
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="font-semibold text-lg">
                    {alert.labels.alertname}
                  </h3>
                  <p className="text-sm text-gray-600">
                    Service: {alert.labels.service}
                  </p>
                  <p className="mt-2">{alert.annotations.description}</p>
                </div>
                <div className="text-right">
                  <span
                    className={`px-2 py-1 rounded text-sm ${
                      alert.labels.severity === 'critical'
                        ? 'bg-red-100 text-red-800'
                        : 'bg-yellow-100 text-yellow-800'
                    }`}
                  >
                    {alert.labels.severity}
                  </span>
                  <p className="text-xs text-gray-500 mt-1">
                    {new Date(alert.receivedAt).toLocaleString()}
                  </p>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
} 