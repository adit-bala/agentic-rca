package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type TraceServiceServer struct {
	coltracepb.UnimplementedTraceServiceServer
	traceFile *os.File
	fileMutex sync.Mutex
}

type SimpleSpan struct {
	Name       string
	StartTime  int64
	EndTime    int64
	Attributes map[string]interface{}
}

type TraceRecord struct {
	Timestamp     time.Time              `json:"timestamp"`
	ServiceName   string                 `json:"service_name"`
	SpanName      string                 `json:"span_name"`
	StartTime     int64                  `json:"start_time_ns"`
	EndTime       int64                  `json:"end_time_ns"`
	DurationNs    int64                  `json:"duration_ns"`
	Attributes    map[string]interface{} `json:"attributes"`
	ResourceAttrs map[string]interface{} `json:"resource_attributes"`
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

// writeTraceToFile writes a trace record to the JSON file
func (s *TraceServiceServer) writeTraceToFile(record TraceRecord) error {
	s.fileMutex.Lock()
	defer s.fileMutex.Unlock()

	if s.traceFile == nil {
		return fmt.Errorf("trace file not initialized")
	}

	// Convert to JSON and write to file
	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal trace record: %v", err)
	}

	// Write JSON line to file
	_, err = s.traceFile.Write(append(jsonData, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write to trace file: %v", err)
	}

	// Flush to ensure data is written
	return s.traceFile.Sync()
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

				// Extract service name from attributes
				serviceName := "unknown"
				if svcName, ok := globalAttrs["service.name"]; ok {
					if svcNameStr, ok := svcName.(string); ok {
						serviceName = svcNameStr
					}
				}

				// Create trace record for file storage
				traceRecord := TraceRecord{
					Timestamp:     time.Now(),
					ServiceName:   serviceName,
					SpanName:      simple.Name,
					StartTime:     simple.StartTime,
					EndTime:       simple.EndTime,
					DurationNs:    simple.EndTime - simple.StartTime,
					Attributes:    simple.Attributes,
					ResourceAttrs: globalAttrs,
				}

				// Write trace to file
				if err := s.writeTraceToFile(traceRecord); err != nil {
					log.Error().Err(err).Msg("Failed to write trace to file")
				} else {
					log.Debug().Str("service", serviceName).Str("span", simple.Name).Msg("Trace written to file")
				}

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

	// Create trace data directory if it doesn't exist
	traceDir := "/data/traces"
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		log.Fatal().Msgf("failed to create trace directory: %v", err)
	}

	// Open trace file for writing
	traceFileName := fmt.Sprintf("%s/traces_%s.jsonl", traceDir, time.Now().Format("2006-01-02_15-04-05"))
	traceFile, err := os.OpenFile(traceFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal().Msgf("failed to open trace file: %v", err)
	}
	defer traceFile.Close()

	log.Info().Msgf("Writing traces to: %s", traceFileName)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 8083))
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	traceServer := &TraceServiceServer{
		traceFile: traceFile,
	}
	coltracepb.RegisterTraceServiceServer(grpcServer, traceServer)

	log.Info().Msgf("Starting trace service on 0.0.0.0:8083")
	grpcServer.Serve(lis)
}
