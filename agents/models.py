from pydantic import BaseModel, Field
from datetime import datetime
from typing import List, Dict, Optional

### Alert Models ###

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

### Service Graph Models ###

class K8sMetadata(BaseModel):
    """Relevant Kubernetes object metadata."""    
    namespace: str
    labels: Optional[Dict[str, str]] = None
    annotations: Optional[Dict[str, str]] = None
    owner_kind: Optional[str] = None
    owner_name: Optional[str] = None
    owner_uid: Optional[str] = None


class ServiceNode(BaseModel):
    """A node in the service graph."""    
    name: str = Field(..., description="Logical service name")
    k8s: K8sMetadata


class ServiceGraph(BaseModel):
    """
    Response model returned by your API:
    - current: the focal service
    - upstream: callers that invoke `current`
    - downstream: callees invoked by `current`
    """    
    current: ServiceNode
    upstream: List[ServiceNode]
    downstream: List[ServiceNode]

class ServiceGraphResponse(BaseModel):    
    services: List[ServiceGraph]

class ServiceDependencies(BaseModel):
    """Model representing a service's upstream and downstream dependencies."""    
    upstream: List[str] = Field(..., description="List of service names that call this service")
    downstream: List[str] = Field(..., description="List of service names that this service calls")