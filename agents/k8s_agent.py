from agents import Agent, ModelSettings, function_tool, AgentOutputSchema
import os
import subprocess
from typing import Dict, List, Optional
from kubernetes import client, config
from kubernetes.client.rest import ApiException
import json

PROMPT = (
    "You are a Kubernetes exploration agent. "
    "Your goal is to examine the cluster in depth until you either:\n"
    "• Detect an anomaly (e.g., Deployment unavailable, pod NotReady, "
    "CrashLoopBackOff, repeated restarts, failing Events) OR\n"
    "• Conclude the cluster looks healthy.\n\n"
    "You will be given a list of services and their k8s metadata. "
    "You have a tool that allows you to run k8s commands. "
    "Only run read-only commands. and do not run any commands that would change the state of the cluster. "
    "When you are satisfied, respond with ONE JSON object:\n"
    '{"conclusion":"healthy|issues_found","details":[...]}'

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
def run_k8s_command(k8s_command: str) -> str:
    """Run a k8s command and return the output."""
    return k8s_client.run_k8s_command(k8s_command)

# Define the K8s agent with its specialized instructions and tools
k8s_agent = Agent(
    name="K8sAgent",
    instructions=PROMPT,
    tools=[run_k8s_command],
    model_settings=ModelSettings(
        tool_choice="required"
    )
)
