package main

import (
	"context"
	"fmt"
	"net"
	"os"

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

func convertProtoSpan(p *tracepb.Span) SimpleSpan {
	// Copy name + timestamps
	out := SimpleSpan{
		Name:       p.Name,
		StartTime:  int64(p.StartTimeUnixNano),
		EndTime:    int64(p.EndTimeUnixNano),
		Attributes: make(map[string]interface{}, len(p.Attributes)),
	}

	for _, kv := range p.Attributes {
		out.Attributes[kv.Key] = kv.Value
	}

	return out
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

				// ──► If you only care about edges/caller→callee:
				//    You can inspect simple.Attributes["service.name"] and simple.Attributes["peer.service"],
				//    then upsert (caller,callee) into your SQLite edge table as shown previously.
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
	coltracepb.RegisterTraceServiceServer(grpcServer, &TraceServiceServer{})

	log.Info().Msgf("Starting trace service on 0.0.0.0:8083")
	grpcServer.Serve(lis)
}
