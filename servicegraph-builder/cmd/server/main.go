package main

import (
	"context"
	"fmt"
	"net"
	"os"
	cache "servicegraph-builder/pkg/cache"
	"servicegraph-builder/pkg/db"
	"servicegraph-builder/pkg/models"
	"strings"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	CACHE_TTL = 600
)

var (
	seenSpans   = cache.New()
	k8sClient   kubernetes.Interface
	neo4jClient *db.Neo4jClient
)

type TraceServiceServer struct {
	coltracepb.UnimplementedTraceServiceServer
}

type Span struct {
	OperationName string
	Attributes    map[string]string
}

type K8sMetadata struct {
	Namespace string
	OwnerKind string
	OwnerName string
	OwnerUID  string
}

type EnrichedSpan struct {
	Span
	ServiceName   string
	HashableName  string
	CallerService string
	CalleeService string
	K8sMetadata   K8sMetadata
}

func initK8sClient() error {
	if k8sClient != nil {
		return nil
	}

	// First try in-cluster config (when running inside Kubernetes)
	cfg, err := rest.InClusterConfig()
	if err == nil {
		log.Info().Msg("Using in-cluster Kubernetes configuration")
	} else {
		log.Info().Msg("Not running in cluster, falling back to kubeconfig")
		// Fall back to kubeconfig for local development
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = clientcmd.RecommendedHomeFile
			log.Info().Str("kubeconfig", kubeconfig).Msg("Using default kubeconfig location")
		}
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatal().Err(err).Msg("cannot build kubeconfig")
		}
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create kubernetes client")
	}
	k8sClient = client
	log.Info().Msg("Successfully initialized Kubernetes client")
	return nil
}

func addK8sMeta(span *models.EnrichedSpan, resourceAttrs map[string]interface{}) error {
	// namespace from OTLP
	if ns, ok := resourceAttrs["k8s.namespace.name"]; ok {
		if any, ok := ns.(*commonpb.AnyValue); ok {
			span.K8sMetadata.Namespace = any.GetStringValue()
		}
	}

	// Deployment / ReplicaSet from OTLP
	switch {
	case resourceAttrs["k8s.deployment.name"] != nil:
		any := resourceAttrs["k8s.deployment.name"].(*commonpb.AnyValue)
		span.K8sMetadata.OwnerKind, span.K8sMetadata.OwnerName = "Deployment", any.GetStringValue()
	case resourceAttrs["k8s.replicaset.name"] != nil:
		any := resourceAttrs["k8s.replicaset.name"].(*commonpb.AnyValue)
		span.K8sMetadata.OwnerKind, span.K8sMetadata.OwnerName = "ReplicaSet", any.GetStringValue()
	}

	// If we already have workload data, stop here
	if span.K8sMetadata.OwnerKind != "" {
		return nil
	}

	// ---- Fallback: Service → Pods → ownerReferences chain ----
	svcName := span.ServiceName
	if svcName == "unknown" {
		svcName = span.OperationName // worst-case fallback
	}
	ns := span.K8sMetadata.Namespace // empty string = all namespaces

	svcs, err := k8sClient.CoreV1().Services(ns).List(
		context.TODO(),
		metav1.ListOptions{FieldSelector: "metadata.name=" + svcName},
	)
	if err != nil || len(svcs.Items) == 0 {
		return err
	}
	svc := svcs.Items[0]

	selector := labels.SelectorFromSet(svc.Spec.Selector).String()
	pods, err := k8sClient.CoreV1().Pods(svc.Namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: selector},
	)
	if err != nil || len(pods.Items) == 0 {
		return err
	}

	// Walk ownerReferences
	owner := pods.Items[0].OwnerReferences
	for len(owner) > 0 {
		ref := owner[0]
		span.K8sMetadata = models.K8sMetadata{
			Namespace: svc.Namespace,
			OwnerKind: ref.Kind,
			OwnerName: ref.Name,
			OwnerUID:  string(ref.UID),
		}
		if ref.Controller != nil && *ref.Controller {
			break // reached Deployment / DaemonSet / Job
		}
		rs, _ := k8sClient.AppsV1().
			ReplicaSets(svc.Namespace).
			Get(context.TODO(), ref.Name, metav1.GetOptions{})
		if rs == nil || len(rs.OwnerReferences) == 0 {
			break
		}
		owner = rs.OwnerReferences
	}
	return nil
}

func enrichSpan(p *tracepb.Span, resourceAttrs map[string]interface{}) models.EnrichedSpan {
	span := models.Span{
		OperationName: p.Name,
		Attributes:    make(map[string]string, len(p.Attributes)),
	}
	for _, kv := range p.Attributes {
		span.Attributes[kv.Key] = kv.Value.GetStringValue()
	}

	// Service name from resource attrs
	serviceName := "unknown"
	if raw, ok := resourceAttrs["service.name"]; ok {
		if any, ok2 := raw.(*commonpb.AnyValue); ok2 {
			serviceName = any.GetStringValue()
		}
	}

	caller, callee := "unknown", "unknown"

	if c, ok := span.Attributes["client.address"]; ok {
		caller, callee = c, serviceName
	}
	if s, ok := span.Attributes["server.address"]; ok {
		parts := strings.SplitN(s, ":", 2)
		callee, caller = parts[0], serviceName
	}

	hash := fmt.Sprintf("%s-%s-%s", serviceName, caller, callee)

	return models.EnrichedSpan{
		Span:          span,
		ServiceName:   serviceName,
		HashableName:  hash,
		CallerService: caller,
		CalleeService: callee,
	}
}

func (s *TraceServiceServer) Export(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	for _, resource := range req.ResourceSpans {
		globalAttrs := make(map[string]interface{}, len(resource.Resource.Attributes))
		for _, attr := range resource.Resource.Attributes {
			globalAttrs[attr.Key] = attr.Value
		}

		for _, scope := range resource.ScopeSpans {
			for _, pspan := range scope.Spans {
				enriched := enrichSpan(pspan, globalAttrs)

				if isHealthSpan(enriched) {
					continue
				}

				if _, ok := seenSpans.Get(enriched.HashableName); ok {
					continue
				} else {
					log.Info().Str("span_name", enriched.HashableName).Msg("New span, writing to database")
					seenSpans.Set(enriched.HashableName, enriched, CACHE_TTL)
				}

				if err := addK8sMeta(&enriched, globalAttrs); err != nil {
					log.Error().Err(err).Msg("cannot add k8s meta")
				}

				log.Info().Any("enriched_span", enriched).Msg("Enriched span")

				// Write to Neo4j
				if err := neo4jClient.WriteSpan(ctx, &enriched); err != nil {
					log.Error().Err(err).Msg("Failed to write span to Neo4j")
				}
			}
		}
	}

	return &coltracepb.ExportTraceServiceResponse{}, nil
}

// isHealthSpan returns true for common k8s health/liveness/readiness probes.
func isHealthSpan(span models.EnrichedSpan) bool {
	// 1) Check the http.route attribute if present
	if route, ok := span.Attributes["http.route"]; ok {
		if strings.HasPrefix(route, "/health") ||
			strings.HasPrefix(route, "/live") ||
			strings.HasPrefix(route, "/ready") {
			return true
		}
	}

	// 2) Fallback to inspecting the operation name
	op := strings.ToLower(span.OperationName)
	if strings.Contains(op, "health") ||
		strings.Contains(op, "liveness") ||
		strings.Contains(op, "ready") {
		return true
	}

	return false
}

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Initialize Neo4j client
	var err error
	neo4jClient, err = db.NewNeo4jClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Neo4j client")
	}
	defer neo4jClient.Close(context.TODO())

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 8083))
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	traceServer := &TraceServiceServer{}
	coltracepb.RegisterTraceServiceServer(grpcServer, traceServer)

	// Initialize Kubernetes client
	if err := initK8sClient(); err != nil {
		log.Fatal().Err(err).Msg("cannot initialize kubernetes client")
		return
	}

	log.Info().Msgf("Starting trace service on 0.0.0.0:8083")
	grpcServer.Serve(lis)
}
