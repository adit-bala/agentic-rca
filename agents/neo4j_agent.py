from agents import Agent, ModelSettings, function_tool, AgentOutputSchema
from neo4j import GraphDatabase
import os
from typing import Dict, List
from models import ServiceNode, K8sMetadata, ServiceGraph, ServiceGraphResponse, ServiceDependencies
from agent_types import AgentName
import json

PROMPT = (
    "You are a Neo4j agent specialized in service dependency lookups. "
    "You are given a list of alerts. "
    "You need to identify the services that are affected by the alert. "
    "The service graph is stored in Neo4j with the following schema:\n"
    "- Nodes labeled as 'Service' with property 'name'\n"
    "- Relationships labeled as 'CALLS' with properties:\n"
    "  * operation: The operation name\n"
    "  * attributesJson: JSON string of span attributes\n"
    "  * k8s_namespace: Kubernetes namespace\n"
    "  * k8s_owner_kind: Kubernetes owner kind\n"
    "  * k8s_owner_name: Kubernetes owner name\n"
    "  * k8s_owner_uid: Kubernetes owner UID\n"
    "  * last_seen: Timestamp of last observation\n\n"
    "Given a service name, query the dependency graph for that service. "
    "Return the list of upstream services that depend on this service, and the downstream services that this service depends on."
)

class Neo4jClient:
    def __init__(self):
        uri = os.getenv("NEO4J_URI", "bolt://neo4j:7687")
        username = os.getenv("NEO4J_USER", "neo4j")
        password = os.getenv("NEO4J_PASSWORD", "password")
        self.driver = GraphDatabase.driver(uri, auth=(username, password))

    def close(self):
        self.driver.close()

    def get_service_dependencies(self, service_name: str) -> Dict[str, List[str]]:
        with self.driver.session() as session:
            # Query for upstream services (services that call this service)
            upstream = session.run("""
                MATCH (caller:Service)-[r:CALLS]->(callee:Service {name: $service_name})
                RETURN caller.name as service
            """, service_name=service_name).value("service")

            # Query for downstream services (services that this service calls)
            downstream = session.run("""
                MATCH (caller:Service {name: $service_name})-[r:CALLS]->(callee:Service)
                RETURN callee.name as service
            """, service_name=service_name).value("service")

            return {
                "upstream": upstream,
                "downstream": downstream
            }

    def get_service_health(self, service_name: str) -> Dict:
        with self.driver.session() as session:
            # Get the most recent calls and their status
            result = session.run("""
                MATCH (caller:Service {name: $service_name})-[r:CALLS]->(callee:Service)
                RETURN callee.name as service,
                       r.operation as operation,
                       r.k8s_namespace as namespace,
                       r.k8s_owner_kind as owner_kind,
                       r.k8s_owner_name as owner_name,
                       r.last_seen as last_seen
                ORDER BY r.last_seen DESC
                LIMIT 10
            """, service_name=service_name)
            
            calls = [dict(record) for record in result]
            return {
                "service": service_name,
                "recent_calls": calls
            }

# Initialize Neo4j client
neo4j_client = Neo4jClient()

def get_current_node_metadata(service_name: str) -> ServiceNode:
    """Get the current node's metadata including Kubernetes information."""
    print(f"Getting metadata for service: {service_name}")
    try:
        with neo4j_client.driver.session() as session:
            result = session.run("""
                MATCH (s:Service {name: $service_name})
                RETURN s.name as name,
                       s.k8s_namespace as namespace,
                       s.k8s_owner_kind as owner_kind,
                       s.k8s_owner_name as owner_name,
                       s.k8s_owner_uid as owner_uid,
                       s.operation as operation,
                       s.attributesJson as attributesJson
            """, service_name=service_name).single()
            
            if not result:
                print(f"Service {service_name} not found")
                raise ValueError(f"Service {service_name} not found")
            
            # Parse attributes JSON if present
            attributes = {}
            if result["attributesJson"]:
                try:
                    attributes = json.loads(result["attributesJson"])
                except json.JSONDecodeError:
                    print(f"Could not parse attributes JSON for service {service_name}")
            
            return ServiceNode(
                name=result["name"],
                k8s=K8sMetadata(
                    namespace=result["namespace"] or "unknown",
                    owner_kind=result["owner_kind"] or "Unknown",
                    owner_name=result["owner_name"] or service_name,
                    owner_uid=result["owner_uid"] or "unknown"
                ),
                operation=result["operation"],
                attributes=attributes
            )
    except Exception as e:
        print(f"Error getting node metadata: {str(e)}")
        raise

def get_service_dependencies(service_name: str) -> ServiceDependencies:
    """Get the names of upstream and downstream services for a given service."""
    print(f"Getting dependencies for service: {service_name}")
    try:
        with neo4j_client.driver.session() as session:
            # Query for upstream services (services that call this service)
            upstream = session.run("""
                MATCH (caller:Service)-[r:CALLS]->(callee:Service {name: $service_name})
                RETURN caller.name as service
            """, service_name=service_name).value("service")

            # Query for downstream services (services that this service calls)
            downstream = session.run("""
                MATCH (caller:Service {name: $service_name})-[r:CALLS]->(callee:Service)
                RETURN callee.name as service
            """, service_name=service_name).value("service")

            return ServiceDependencies(
                upstream=upstream,
                downstream=downstream
            )
    except Exception as e:
        print(f"Error getting service dependencies: {str(e)}")
        raise

@function_tool
def get_service_graph(service_name: List[str]) -> ServiceGraphResponse:
    """Get the complete service graph for a given service, including its metadata and dependencies."""
    print(f"Building service graph for: {service_name}")
    graphs = []
    for service in service_name:
        try:
            node = get_current_node_metadata(service)
            deps = get_service_dependencies(service)
            graph = ServiceGraph(
                current=node,
                upstream=[ServiceNode(name=s, k8s=node.k8s) for s in deps.upstream],
                downstream=[ServiceNode(name=s, k8s=node.k8s) for s in deps.downstream],
            )
            graphs.append(graph)
        except Exception as e:
            print(f"Error building service graph: {str(e)}")
            raise
    return ServiceGraphResponse(services=graphs)

# Define the Neo4j agent with its specialized instructions and tools
neo4j_agent = Agent(
    name=AgentName.NEO4J,
    instructions=(
        PROMPT
    ),
    tools=[get_service_graph],
    output_type=AgentOutputSchema(ServiceGraphResponse, strict_json_schema=False),
    model_settings=ModelSettings(
        tool_choice="required"
    )
)
