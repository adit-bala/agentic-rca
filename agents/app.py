"""
A minimal FastAPI receiver for Prometheus Alertmanager webhooks.

• POST  /alerts   – accepts the JSON payload defined in
  https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
"""
from fastapi import FastAPI, status, HTTPException, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from typing import Any
from datetime import datetime
from typing import List, Dict, Optional
import uuid
import json
import asyncio

from models import AlertGroup
from rca_manager import RCA_Manager
from websocket_types import (
    WebSocketMessageType,
    StatusMessage,
    ErrorMessage
)

app = FastAPI(title="Alertmanager Webhook")

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Store alerts temporarily with their websocket IDs
alert_store: Dict[str, AlertGroup] = {}

@app.post("/alerts", status_code=status.HTTP_202_ACCEPTED)
async def receive_alerts(payload: AlertGroup) -> dict[str, Any]:
    """
    Receive a batch of alerts from Alertmanager and return a websocket ID.
    """
    try:
        count = len(payload.alerts)
        print(f"✅ received {count} alerts | groupKey={payload.groupKey} | status={payload.status}")
        
        # Generate a unique websocket ID
        ws_id = str(uuid.uuid4())
        
        # Store the payload with the websocket ID
        alert_store[ws_id] = payload
        
        return {"websocket_id": ws_id}
    except Exception as exc:
        print("❌ failed to process alerts")
        raise HTTPException(status_code=500, detail=str(exc))

@app.websocket("/process/{ws_id}")
async def process_alerts(websocket: WebSocket, ws_id: str):
    """
    Process alerts for a given websocket ID and send updates through WebSocket.
    """
    try:
        await websocket.accept()
        
        if ws_id not in alert_store:
            await websocket.send_json(ErrorMessage(
                type=WebSocketMessageType.ERROR,
                data="Websocket ID not found"
            ))
            await websocket.close()
            return
        
        payload = alert_store[ws_id]
        
        # Send initial status
        await websocket.send_json(StatusMessage(
            type=WebSocketMessageType.STATUS,
            data="Starting RCA analysis"
        ))
        
        try:
            report = await RCA_Manager(websocket).run(payload)
            
            # Send the final report
            await websocket.send_json(StatusMessage(
                type=WebSocketMessageType.STATUS,
                data="Analysis complete"
            ))
            
        except Exception as exc:
            # Send error status
            await websocket.send_json(ErrorMessage(
                type=WebSocketMessageType.ERROR,
                data=str(exc)
            ))
            
        finally:
            # Clean up the stored payload
            del alert_store[ws_id]
            await websocket.close()
            
    except WebSocketDisconnect:
        print("WebSocket disconnected")
    except Exception as exc:
        print("❌ failed to process alerts")
        try:
            await websocket.send_json(ErrorMessage(
                type=WebSocketMessageType.ERROR,
                data=str(exc)
            ))
        except:
            pass
        finally:
            await websocket.close()