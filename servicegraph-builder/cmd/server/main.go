package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type TraceServiceServer struct {
	coltracepb.UnimplementedTraceServiceServer
}

type SimpleSpan struct {
	Name       string
	StartTime  int64
	EndTime    int64
	Attributes map[string]interface{}
}

type EnrichedSpan struct {
	SimpleSpan
	HashableName       string
	OperationName string
	CallerService string
	CalleeService string
}

var (
	seenSpans = make(map[string]struct{})
)

func convertProtoSpan(p *tracepb.Span) SimpleSpan {
	// Copy name + timestamps
	out := SimpleSpan{
		Name:       p.Name,
		Attributes: make(map[string]interface{}, len(p.Attributes)),
	}

	for _, kv := range p.Attributes {
		out.Attributes[kv.Key] = kv.Value
	}

	return out
}

func enrichSpan(simple SimpleSpan, resourceAttrs map[string]interface{},) EnrichedSpan {
    caller := "unknown"
    callee := "unknown"

    // 1) If this is an incoming call, OTLP will carry “client.address”
    if rawClient, ok := simple.Attributes["client.address"]; ok {
        if anyVal, ok2 := rawClient.(*commonpb.AnyValue); ok2 {
            caller = anyVal.GetStringValue()
        }
        // resourceAttrs["service.name"] is also an AnyValue
        if rawSvc, ok3 := resourceAttrs["service.name"]; ok3 {
            if anySvc, ok4 := rawSvc.(*commonpb.AnyValue); ok4 {
                callee = anySvc.GetStringValue()
            }
        }
    }

    // 2) Otherwise, if this is an outgoing call, OTLP will carry “server.address”
    if rawServer, ok := simple.Attributes["server.address"]; ok {
        if anyVal, ok2 := rawServer.(*commonpb.AnyValue); ok2 {
            full := anyVal.GetStringValue()
            parts := strings.SplitN(full, ":", 2)
            callee = parts[0]
        }
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

				simple := convertProtoSpan(pspan)
				log.Info().
					Str("name", simple.Name).
					Int64("start_ns", simple.StartTime).
					Int64("end_ns", simple.EndTime).
					Interface("attrs", simple.Attributes).
					Msg("Converted SimpleSpan")

				enriched := enrichSpan(convertProtoSpan(pspan), globalAttrs)

				log.Info().
					Str("caller", enriched.CallerService).
					Str("callee", enriched.CalleeService).
					Msg("Enriched span")

				if _, ok := seenSpans[enriched.HashableName]; ok {
					log.Info().Str("span_name", enriched.HashableName).Msg("Skipping already seen span")
					continue
				} else {
					log.Info().Str("span_name", enriched.HashableName).Msg("New span, writing to database")
					seenSpans[enriched.HashableName] = struct{}{}
				}

				
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

	log.Info().Msgf("Starting trace service on 0.0.0.0:8083")
	grpcServer.Serve(lis)
}
