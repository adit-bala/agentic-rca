"""
A minimal FastAPI receiver for Prometheus Alertmanager webhooks.

• POST  /alerts   – accepts the JSON payload defined in
  https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
"""
from fastapi import FastAPI, status, HTTPException
from typing import Any
import logging
from datetime import datetime
from typing import List, Dict, Optional

from models import AlertGroup
from rca_manager import RCA_Manager

app = FastAPI(title="Alertmanager Webhook")

log = logging.getLogger("uvicorn.error")


@app.post("/alerts", status_code=status.HTTP_202_ACCEPTED)
async def receive_alerts(payload: AlertGroup) -> dict[str, Any]:
    """
    Receive a batch of alerts from Alertmanager.
    """
    try:
        count = len(payload.alerts)
        log.info("✅ received %s alerts | groupKey=%s | status=%s",
                 count, payload.groupKey, payload.status)
        
        report = await RCA_Manager().run(payload)

        return {"report": report}
    except Exception as exc:
        log.exception("❌ failed to process alerts")
        raise HTTPException(status_code=500, detail=str(exc))