from pydantic import BaseModel, Field
from datetime import datetime
from typing import List, Dict

class Alert(BaseModel):
    status: str
    labels: Dict[str, str]
    annotations: Dict[str, str]
    startsAt: datetime
    endsAt: datetime
    generatorURL: str

class AlertGroup(BaseModel):
    version: str = Field(..., pattern=r"^\d+$")
    groupKey: str
    status: str
    receiver: str
    groupLabels: Dict[str, str]
    commonLabels: Dict[str, str]
    commonAnnotations: Dict[str, str]
    externalURL: str
    alerts: List[Alert] 