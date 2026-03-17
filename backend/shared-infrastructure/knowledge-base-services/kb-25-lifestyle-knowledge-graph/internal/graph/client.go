package graph

import (
	"context"
	"fmt"

	"kb-25-lifestyle-knowledge-graph/internal/config"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

// GraphClient is the interface both Client and NoOpClient implement.
type GraphClient interface {
	Run(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error)
	Write(ctx context.Context, cypher string, params map[string]any) error
	HealthCheck(ctx context.Context) error
	Close(ctx context.Context) error
}

type Client struct {
	driver   neo4j.DriverWithContext
	database string
	logger   *zap.Logger
}

var _ GraphClient = (*Client)(nil)
var _ GraphClient = (*NoOpClient)(nil)

func NewClient(cfg *config.Config, logger *zap.Logger) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4j.URI,
		neo4j.BasicAuth(cfg.Neo4j.User, cfg.Neo4j.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	logger.Info("Neo4j connection established",
		zap.String("uri", cfg.Neo4j.URI),
		zap.String("database", cfg.Neo4j.Database),
	)

	return &Client{
		driver:   driver,
		database: cfg.Neo4j.Database,
		logger:   logger,
	}, nil
}

func (c *Client) Run(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	records, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return result.Collect(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("cypher read failed: %w", err)
	}
	return records.([]*neo4j.Record), nil
}

func (c *Client) Write(ctx context.Context, cypher string, params map[string]any) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, cypher, params)
		return nil, err
	})
	return err
}

func (c *Client) HealthCheck(ctx context.Context) error {
	return c.driver.VerifyConnectivity(ctx)
}

func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// --- NoOp client for graceful degradation ---

type NoOpClient struct{}

func NewNoOpClient() *NoOpClient { return &NoOpClient{} }

func (n *NoOpClient) Run(_ context.Context, _ string, _ map[string]any) ([]*neo4j.Record, error) {
	return nil, nil
}

func (n *NoOpClient) Write(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func (n *NoOpClient) HealthCheck(_ context.Context) error { return nil }

func (n *NoOpClient) Close(_ context.Context) error { return nil }
