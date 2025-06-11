from __future__ import annotations

import os, time, httpx, json
from typing import List, Dict, Any, Optional
from agents import Agent, ModelSettings, function_tool

OBSERVE_BASE   = os.getenv("OBSERVE_BASE_URL", "https://119137983744.observeinc.com")
BEARER_TOKEN   = os.getenv("OBSERVE_API_TOKEN")
DATASET_ID     = os.getenv("OBSERVE_DATASET", "42492215")

PROMPT = (
    "You are an **Observe Log Analysis agent**.\n\n"
    "Inputs\n"
    "------\n"
    "• **Alert JSON** → one or more firing alerts (label key-values, description, etc.)\n"
    "• **Service graph JSON** → for each alert-impacted service:\n"
    "  ▸ `current`  : focal service (name + k8s metadata)\n"
    "  ▸ `upstream` : services this node calls\n"
    "  ▸ `downstream`: services that call this node\n\n"
    "Service metadata (from Neo4j):\n"
    "  • `k8s_namespace`, `k8s_owner_kind`, `k8s_owner_name`, `k8s_owner_uid`\n"
    "  • `operation` and `attributes` for additional context\n\n"
    "Allowed tools (read-only):\n"
    "• `opl_get_logs`   – recent raw logs for a service.\n"
    "• `opl_get_errors` – recent error/warning lines.\n\n"
    "Goal\n"
    "----\n"
    "Given the service-graph JSON and alert labels, Query all relevant service "
    "names & namespaces. Keep OPAL windows small (≤30 m) and "
    "limit rows to stay under token limits. Summarise key patterns "
    "(frequent stack traces, HTTP 500 spikes, OOMKilled events, etc.).\n\n"
    "When composing your answer:\n"
)



class ObserveClient:
    """Read-only OPAL queries using a pre-generated SSO bearer token."""
    def __init__(self):
        token = BEARER_TOKEN.strip('"') if BEARER_TOKEN else None
        if not token:
            raise RuntimeError("OBSERVE_API_TOKEN env var is missing or quoted")
        self._headers = {
            "Authorization": f"Bearer {BEARER_TOKEN}",
            "Content-Type":  "application/json",
        }

    async def query(self, opal: str, minutes: int = 15, limit: int = 1000) -> List[Dict[str, Any]]:
        body = {
            "query": {
                "stages": [
                    {
                        "input": [
                            {
                                "inputName": "Logs",
                                "datasetId": DATASET_ID
                            }
                        ],
                        "stageID": "main",
                        "pipeline": f"{opal}"
                    }
                ]
            }
        }
        url = f"/v1/meta/export/query?interval={minutes}m"
        print(f"\nObserve HTTP Request:")
        print(f"POST {OBSERVE_BASE}{url}")
        print("Headers:")
        for key, value in self._headers.items():
            print(f"  {key}: {value}")
        print("Body:")
        print(json.dumps(body, indent=2))
        print()
        async with httpx.AsyncClient(base_url=OBSERVE_BASE, headers=self._headers) as cli:
            try:
                resp = await cli.post(url, json=body, timeout=60)
                resp.raise_for_status()
                rows = [
                    json.loads(line)
                    for line in resp.text.splitlines()
                    if line.strip()
                ]
                return rows
            except httpx.HTTPStatusError as e:
                print(f"\nHTTP Error {e.response.status_code}:")
                print(f"Response Headers: {dict(e.response.headers)}")
                print(f"Response Body: {e.response.text}")
                raise
            except httpx.RequestError as e:
                print(f"\nRequest Error: {str(e)}")
                raise
            except json.JSONDecodeError as e:
                print(f"\nJSON Decode Error: {str(e)}")
                print(f"Response Text: {resp.text}")
                raise


observe = ObserveClient()

@function_tool
async def opl_get_logs(
    service: str,
    namespace: str = "default",
    minutes: int = 15,
    limit: int = 800
) -> str:
    """
    Fetch raw logs for (service, namespace) over the last `minutes`.
    """
    pipeline = (
        f'filter namespace = "{namespace}"\n'
        f'filter container = "{service}"\n'
        f'limit {limit}'
    )
    rows = await observe.query(pipeline, minutes=minutes, limit=limit)
    return json.dumps(rows, indent=2)[:15_000]


observe_agent = Agent(
    name="ObserveLogAgent",
    instructions=PROMPT,
    tools=[opl_get_logs],
    output_type=str,
    model_settings=ModelSettings(tool_choice="required"),
)