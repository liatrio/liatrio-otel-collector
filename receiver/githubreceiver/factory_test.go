package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	if factory == nil {
		t.Error("NewFactory() should not return nil")
	}

	cfg := factory.CreateDefaultConfig()
	if cfg == nil {
		t.Error("CreateDefaultConfig() should not return nil")
	}

	typ := factory.Type()
	if typ != typeStr {
		t.Errorf("factory.Type() should return %s, got %s", typeStr, typ)
	}
}

func TestCreateMetricsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	r, err := factory.CreateMetricsReceiver(context.Background(), receiver.CreateSettings{}, cfg, consumertest.NewNop())

	if err != nil {
		t.Errorf("failed to create metrics receiver: %v", err)
	}

	if r == nil {
		t.Error("CreateMetricsReceiver() should not return nil")
	}
}

func TestCreateMetricsReceiverNilNextConsumer(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	_, err := factory.CreateMetricsReceiver(context.Background(), receiver.CreateSettings{}, cfg, nil)

	if err == nil {
		t.Error("CreateMetricsReceiver() should return an error when next consumer is nil")
	}
}
