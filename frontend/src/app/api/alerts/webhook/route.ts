import { NextResponse } from 'next/server';

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

interface AlertData {
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
}

// In-memory store for alerts (in a real app, you'd use a database)
let alerts: Alert[] = [];

export async function POST(request: Request) {
  try {
    const data = await request.json();
    
    // Handle Alertmanager webhook format
    const alertsData = Array.isArray(data.alerts) ? data.alerts : [data];
    
    // Add timestamp to each alert
    const alertsWithTimestamp = alertsData.map((alert: AlertData) => ({
      ...alert,
      receivedAt: new Date().toISOString(),
    }));
    
    // Add to alerts array (keep last 100 alerts)
    alerts = [...alertsWithTimestamp, ...alerts].slice(0, 100);
    
    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Error processing webhook:', error);
    return NextResponse.json({ error: 'Invalid request' }, { status: 400 });
  }
}

export async function GET() {
  return NextResponse.json(alerts);
} 