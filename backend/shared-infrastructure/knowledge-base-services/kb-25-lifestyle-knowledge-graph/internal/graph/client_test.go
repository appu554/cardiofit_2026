package graph

import (
	"context"
	"testing"
)

func TestNoOpClientReturnsNil(t *testing.T) {
	c := NewNoOpClient()
	result, err := c.Run(context.Background(), "RETURN 1 AS n", nil)
	if err != nil {
		t.Fatalf("NoOp Run() should not error: %v", err)
	}
	if result != nil {
		t.Error("NoOp Run() should return nil result")
	}
}

func TestNoOpClientWrite(t *testing.T) {
	c := NewNoOpClient()
	err := c.Write(context.Background(), "CREATE (n:Test)", nil)
	if err != nil {
		t.Errorf("NoOp Write() should not error: %v", err)
	}
}

func TestNoOpClientHealthCheck(t *testing.T) {
	c := NewNoOpClient()
	if err := c.HealthCheck(context.Background()); err != nil {
		t.Errorf("NoOp HealthCheck should succeed: %v", err)
	}
}
