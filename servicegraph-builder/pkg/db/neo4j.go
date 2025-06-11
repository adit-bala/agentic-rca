package db

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"servicegraph-builder/pkg/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

type Neo4jClient struct {
	driver neo4j.DriverWithContext
}

func NewNeo4jClient() (*Neo4jClient, error) {
	uri := getEnv("NEO4J_URI", "bolt://host.docker.internal:7687")
	username := getEnv("NEO4J_USERNAME", "neo4j")
	password := getEnv("NEO4J_PASSWORD", "password")

	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth(username, password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity (with timeout)
	verifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(verifyCtx); err != nil {
		driver.Close(verifyCtx)
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return &Neo4jClient{driver: driver}, nil
}

// Close shuts down the underlying Neo4j driver.
func (c *Neo4jClient) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// WriteSpan upserts caller & callee nodes and a single CALLS
// relationship, now safe against duplicates.
func (c *Neo4jClient) WriteSpan(ctx context.Context, span *models.EnrichedSpan) error {
	// Normalise service names (trim, lowercase)
	caller := normaliseServiceName(span.CallerService)
	callee := normaliseServiceName(span.CalleeService)

	// Skip self-calls, unknown or IP-literal services
	if caller == callee || caller == "" || callee == "" {
		return nil
	}

	// Convert attributes map to JSON string
	attributesJSON, err := json.Marshal(span.Attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create / ensure nodes
		_, e := tx.Run(ctx, `
			MERGE (caller:Service {name:$caller})
			SET   caller.k8s_namespace  = $k8sNamespace,
			      caller.k8s_owner_kind = $k8sOwnerKind,
			      caller.k8s_owner_name = $k8sOwnerName,
			      caller.k8s_owner_uid  = $k8sOwnerUID,
			      caller.operation      = $operation,
			      caller.attributesJson = $attributesJson
			MERGE (callee:Service {name:$callee})
			SET   callee.k8s_namespace  = $k8sNamespace,
			      callee.k8s_owner_kind = $k8sOwnerKind,
			      callee.k8s_owner_name = $k8sOwnerName,
			      callee.k8s_owner_uid  = $k8sOwnerUID,
			      callee.operation      = $operation,
			      callee.attributesJson = $attributesJson
		`, map[string]any{
			"caller":         caller,
			"callee":         callee,
			"k8sNamespace":   span.K8sMetadata.Namespace,
			"k8sOwnerKind":   span.K8sMetadata.OwnerKind,
			"k8sOwnerName":   span.K8sMetadata.OwnerName,
			"k8sOwnerUID":    span.K8sMetadata.OwnerUID,
			"operation":      span.OperationName,
			"attributesJson": string(attributesJSON),
		})
		if e != nil {
			return nil, e
		}

		// Upsert relationship
		_, e = tx.Run(ctx, `
			MATCH (c:Service {name:$caller}),
			      (d:Service {name:$callee})
			MERGE (c)-[r:CALLS]->(d)
		`, map[string]any{
			"caller": caller,
			"callee": callee,
		})
		return nil, e
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to write span to Neo4j")
		return err
	}
	return nil
}

// normaliseServiceName trims, lower-cases, and rejects IP literals.
func normaliseServiceName(raw string) string {
	svc := strings.ToLower(strings.TrimSpace(raw))
	if svc == "" {
		return ""
	}
	if net.ParseIP(svc) != nil {
		return "" // treat as unknown service
	}
	return svc
}

// getEnv returns the env var or default.
func getEnv(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultValue
}
