# agentic root cause analysis
A platform for automated root cause analysis using AI agents

- demo test architecture: simple microservice design instrumented with Prometheus and AlertManager
- eBPF & OpenTelemetry for distributed tracing to build a service dependency graph
- agentic analysis that extracts context from neo4j, k8s, observe logs, and the codebase
- TODO: tests

# Running the app 

> [!NOTE]  
> There are probably some env vars you have to set up


In first terminal
```
./scripts/run.sh # Make sure docker is running
```

This will start up a minikube environment and will deploy the microservices, observability infra, and the infra required to build the service-dependency graph using [this  helm chart](https://github.com/adit-bala/agentic-rca/tree/main/servicegraph-helm)

In second terminal run
```
./run_agents.sh --fresh --codebase=<codebase_path> # embeddings will be created from the codebase that is passed in for the github agent
```
Ideally, the codebase path has the code that is deployed into the minikube cluster.

In third terminal
```
./scripts/quick-test.sh test
```

This will create multiple e2e requests through the microservices, sending the network calls that create the edges for our service depedency graph. If you want to see the graph, go to `http://localhost:7474` and query `MATCH(N) RETURN (N)`

In third terminal 
```
npm run dev
```
to start the frontend. There are two pages, one to visualize the service dependency graph, and the second shows the alerts.

Right now, Alertmanager has not been configured properly to receive alerts from Prometheus, so you can trigger an alert manually.

In fourth terminal
```
./scripts/test_alert.sh --frontend
```
This sends a mock alert payload to the frontend. If you navigate to the alerts page and press "Start Root Cause Analysis", it will kick off the agentic rca

