package main

import (
	"context"
	"fmt"
	"net"
	"os"
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

type TraceServiceServer struct {
	coltracepb.UnimplementedTraceServiceServer
}

type SimpleSpan struct {
	Name       string
	Attributes map[string]string
}

type k8sMeta struct {
	Namespace string
	OwnerKind string
	OwnerName string
	OwnerUID  string
}

type EnrichedSpan struct {
	SimpleSpan
	K8sMeta       k8sMeta
	HashableName  string
	OperationName string
	CallerService string
	CalleeService string
}

var (
	seenSpans = make(map[string]struct{})
	k8sClient kubernetes.Interface
)

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

func convertProtoSpan(p *tracepb.Span) SimpleSpan {
	out := SimpleSpan{
		Name:       p.Name,
		Attributes: make(map[string]string, len(p.Attributes)),
	}

	for _, kv := range p.Attributes {
		out.Attributes[kv.Key] = kv.Value.GetStringValue()
	}

	return out
}

func addK8sMeta(span *EnrichedSpan, resourceAttrs map[string]interface{}) error {
	if ns, ok := resourceAttrs["k8s.namespace.name"]; ok {
		if any, ok := ns.(*commonpb.AnyValue); ok {
			span.K8sMeta.Namespace = any.GetStringValue()
		}
	}

	// Prefer Deployment info; fall back to ReplicaSet if needed
	if dep, ok := resourceAttrs["k8s.deployment.name"]; ok {
		if any, ok := dep.(*commonpb.AnyValue); ok {
			span.K8sMeta.OwnerKind = "Deployment"
			span.K8sMeta.OwnerName = any.GetStringValue()
		}
	} else if rs, ok := resourceAttrs["k8s.replicaset.name"]; ok {
		if any, ok := rs.(*commonpb.AnyValue); ok {
			span.K8sMeta.OwnerKind = "ReplicaSet"
			span.K8sMeta.OwnerName = any.GetStringValue()
		}
	}

	// If we already discovered a workload kind/name, we’re done
	if span.K8sMeta.OwnerKind != "" {
		return nil
	}

	ns := span.K8sMeta.Namespace // may be empty; empty means “all namespaces”
	svcs, err := k8sClient.CoreV1().Services(ns).List(
		context.TODO(),
		metav1.ListOptions{
			FieldSelector: "metadata.name=" + span.Name,
		},
	)
	if err != nil || len(svcs.Items) == 0 {
		return err // nothing found
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

	// Walk one Pod's ownerReferences up to the top-level controller
	owner := pods.Items[0].OwnerReferences
	for len(owner) > 0 && owner[0].Kind != "" {
		ref := owner[0]
		span.K8sMeta = k8sMeta{
			Namespace: svc.Namespace,
			OwnerKind: ref.Kind,
			OwnerName: ref.Name,
			OwnerUID:  string(ref.UID),
		}

		// Stop at the first controller (Deployment, DaemonSet, Job, etc.)
		if ref.Controller != nil && *ref.Controller {
			break
		}

		// Otherwise follow ReplicaSet → Deployment chain if present
		rs, err := k8sClient.AppsV1().
			ReplicaSets(svc.Namespace).
			Get(context.TODO(), ref.Name, metav1.GetOptions{})
		if err != nil || rs == nil || len(rs.OwnerReferences) == 0 {
			break
		}
		owner = rs.OwnerReferences
	}
	return nil
}

func enrichSpan(simple SimpleSpan, resourceAttrs map[string]interface{}) EnrichedSpan {
	caller := "unknown"
	callee := "unknown"

	// 1) If this is an incoming call, OTLP will carry "client.address"
	if rawClient, ok := simple.Attributes["client.address"]; ok {
		caller = rawClient
		if rawSvc, ok3 := resourceAttrs["service.name"]; ok3 {
			if anySvc, ok4 := rawSvc.(*commonpb.AnyValue); ok4 {
				callee = anySvc.GetStringValue()
			}
		}
	}

	// 2) Otherwise, if this is an outgoing call, OTLP will carry "server.address"
	if rawServer, ok := simple.Attributes["server.address"]; ok {
		full := rawServer
		parts := strings.SplitN(full, ":", 2)
		callee = parts[0]
		if rawSvc, ok3 := resourceAttrs["service.name"]; ok3 {
			if anySvc, ok4 := rawSvc.(*commonpb.AnyValue); ok4 {
				caller = anySvc.GetStringValue()
			}
		}
	}

	return EnrichedSpan{
		SimpleSpan:    simple,
		HashableName:  fmt.Sprintf("%s-%s-%s", simple.Name, caller, callee),
		OperationName: simple.Name,
		CallerService: caller,
		CalleeService: callee,
	}
}

func (s *TraceServiceServer) Export(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	log.Info().Msg("Received trace data")

	for _, resource := range req.ResourceSpans {
		globalAttrs := make(map[string]interface{}, len(resource.Resource.Attributes))
		for _, attr := range resource.Resource.Attributes {
			globalAttrs[attr.Key] = attr.Value
		}
		log.Info().
			Interface("global_attributes", globalAttrs).
			Msg("Resource-level attributes")

		for _, scope := range resource.ScopeSpans {
			for _, pspan := range scope.Spans {
				log.Info().Str("span_name", pspan.Name).Msg("Processing span")
				enriched := enrichSpan(convertProtoSpan(pspan), globalAttrs)

				// if _, ok := seenSpans[enriched.HashableName]; ok {
				// 	log.Info().Str("span_name", enriched.HashableName).Msg("Skipping already seen span")
				// 	continue
				// } else {
				// 	log.Info().Str("span_name", enriched.HashableName).Msg("New span, writing to database")
				// 	seenSpans[enriched.HashableName] = struct{}{}
				// }

				if err := addK8sMeta(&enriched, globalAttrs); err != nil {
					log.Error().Err(err).Msg("cannot add k8s meta")
				}

				log.Info().
					Str("span_name", enriched.HashableName).
					Str("caller", enriched.CallerService).
					Str("callee", enriched.CalleeService).
					Str("k8s_namespace", enriched.K8sMeta.Namespace).
					Str("k8s_owner_kind", enriched.K8sMeta.OwnerKind).
					Str("k8s_owner_name", enriched.K8sMeta.OwnerName).
					Str("k8s_owner_uid", enriched.K8sMeta.OwnerUID).
					Msg("Enriched span")

				// TODO: Write to database
			}
		}
	}

	return &coltracepb.ExportTraceServiceResponse{}, nil
}

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

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
