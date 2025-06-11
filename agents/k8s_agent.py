from agents import Agent, ModelSettings, function_tool, AgentOutputSchema
import os
import subprocess
from typing import Dict, List, Optional
from kubernetes import client, config
from kubernetes.client.rest import ApiException
import json

PROMPT = (
    "You are a **Kubernetes exploration agent** for on-call engineers.\n\n"
    "Inputs\n"
    "------\n"
    "• **Alert JSON** → one or more firing alerts (label key-values, description, etc.)\n"
    "• **Service graph JSON** → for each alert-impacted service:\n"
    "  ▸ `current`  : focal service (name + k8s metadata)\n"
    "  ▸ `upstream` : services this node calls\n"
    "  ▸ `downstream`: services that call this node\n\n"
    "Cluster layout hints (from Neo4j):\n"
    "  • `k8s_namespace`, `k8s_owner_kind`, `k8s_owner_name`, `k8s_owner_uid`\n"
    "  • `last_seen` timestamp on CALLS edges\n\n"
    "Goal\n"
    "----\n"
    "Triangulate **root cause** by inspecting Pods, Deployments, Events and Nodes that belong to the impacted services.\n\n"
    "⚠️  **Allowed kubectl commands only**\n"
    "• `kubectl get <TYPE> [NAME] [-n NAMESPACE] [--selector ...]`\n"
    "    – quick list, wide output, optional label selectors  📜\n"
    "• `kubectl describe <TYPE> [NAME] [-n NAMESPACE]`\n"
    "    – deep dive into spec, status, recent Events  🔬\n"
    "Use `get` to locate resources and `describe` to investigate details (conditions, container restarts, warnings, etc.).\n"
    "Never modify cluster state (no delete / apply / exec). Only read-only operations are permitted.\n\n"
    "When composing your answer:\n"
    "  1. Map alert labels → service graph nodes → k8s namespace/owner.\n"
    "  2. Issue the minimal set of `get`/`describe` calls to confirm health:\n"
    "     • Pods (phase, restarts, image tag)\n"
    "     • Deployments / ReplicaSets (ready vs desired)\n"
    "     • Events (Warnings, FailedScheduling, CrashLoopBackOff)\n"
    "  3. Propose the most plausible root cause and next diagnostic step.\n"
)

def get_minikube_kubeconfig() -> str:
    """Get the path to the minikube kubeconfig file."""
    try:
        result = subprocess.run(
            ["minikube", "kubectl", "--", "config", "view", "--raw"],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        raise Exception(f"Failed to get minikube kubeconfig: {e.stderr}")

def is_minikube_running() -> bool:
    """Check if minikube is running."""
    try:
        result = subprocess.run(
            ["minikube", "status"],
            capture_output=True,
            text=True,
            check=True
        )
        return "Running" in result.stdout
    except subprocess.CalledProcessError:
        return False

class K8sClient:
    def __init__(self):
        try:
            if is_minikube_running():
                config.load_kube_config()
            else:
                config.load_incluster_config()
        except Exception as e:
            raise Exception(f"Could not configure kubernetes client: {str(e)}")

    def run_k8s_command(self, command: str) -> str:
        """
        Execute a kubectl command and return its output.
        """
        try:
            # Split the command into parts
            cmd_parts = command.split()

            # Execute the command
            result = subprocess.run(
                cmd_parts,
                capture_output=True,
                text=True,
                check=True
            )
            
            return result.stdout
        except subprocess.CalledProcessError as e:
            return f"Error executing command: {e.stderr}"
        except Exception as e:
            return f"Error: {str(e)}"

# Initialize K8s client
k8s_client = K8sClient()

@function_tool
def kubectl_get(resource: str, namespace: Optional[str] = None,
                selector: Optional[str] = None, output: str = "-o wide") -> str:
    """
    Read-only wrapper around `kubectl get`.
    Example: kubectl_get("pods", namespace="default", selector="app=my-svc")
    """
    cmd = "kubectl get " + resource
    if namespace:
        cmd += f" -n {namespace}"
    if selector:
        cmd += f" -l {selector}"
    if output:
        cmd += f" {output}"
    return k8s_client.run_k8s_command(cmd)


@function_tool
def kubectl_describe(resource: str, name: Optional[str] = None,
                     namespace: Optional[str] = None) -> str:
    """
    Read-only wrapper around `kubectl describe`.
    Example: kubectl_describe("deployment", "user-svc", namespace="prod")
    """
    cmd = f"kubectl describe {resource}"
    if name:
        cmd += f" {name}"
    if namespace:
        cmd += f" -n {namespace}"
    return k8s_client.run_k8s_command(cmd)

# Define the K8s agent with its specialized instructions and tools
k8s_agent = Agent(
    name="K8sAgent",
    instructions=PROMPT,
    tools=[kubectl_get, kubectl_describe],
    model_settings=ModelSettings(
        tool_choice="required"
    )
)
