package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"servicegraph-builder/pkg/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

type Neo4jClient struct {
	driver neo4j.DriverWithContext
}

// NewNeo4jClient creates and verifies a Neo4j driver using environment variables.
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

// WriteSpan upserts the caller & callee service nodes and a CALLS relationship
// uniquely identified by span.HashableName, setting all desired properties.
func (c *Neo4jClient) WriteSpan(ctx context.Context, span *models.EnrichedSpan) error {
	// Create a write-mode session tied to the caller's context
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	// Convert attributes map to JSON string
	attributesJSON, err := json.Marshal(span.Attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 1) Merge caller and callee service nodes (requires unique constraint on :Service(name))
		_, err := tx.Run(ctx, `
			MERGE (:Service { name: $caller })
			MERGE (:Service { name: $callee })
		`, map[string]interface{}{
			"caller": span.CallerService,
			"callee": span.CalleeService,
		})
		if err != nil {
			return nil, err
		}

		// 2) merge the single CALLS relationship and overwrite its props
        _, err = tx.Run(
            ctx,
            `MATCH  (c:Service {name:$caller}),
                   (d:Service {name:$callee})
             MERGE  (c)-[r:CALLS]->(d)
             SET    r.operation       = $operation,
                    r.attributesJson  = $attributesJson,
                    r.k8s_namespace   = $k8sNamespace,
                    r.k8s_owner_kind  = $k8sOwnerKind,
                    r.k8s_owner_name  = $k8sOwnerName,
                    r.k8s_owner_uid   = $k8sOwnerUID,
                    r.last_seen       = datetime()`,
            map[string]any{
                "caller":        span.CallerService,
                "callee":        span.CalleeService,
                "operation":     span.OperationName,
                "attributesJson": string(attributesJSON),
                "k8sNamespace":  span.K8sMetadata.Namespace,
                "k8sOwnerKind":  span.K8sMetadata.OwnerKind,
                "k8sOwnerName":  span.K8sMetadata.OwnerName,
                "k8sOwnerUID":   span.K8sMetadata.OwnerUID,
            },
        )
		if err != nil {
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to write span to Neo4j")
		return err
	}

	return nil
}

// getEnv returns the value of the environment variable or a default.
func getEnv(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultValue
}
