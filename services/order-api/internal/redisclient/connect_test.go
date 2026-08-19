package redisclient

import (
	"context"
	"testing"
)

func TestConnectRejectsInvalidURL(t *testing.T) {
	_, err := Connect(context.Background(), "://not-a-url")
	if err == nil {
		t.Fatal("expected error for invalid redis url")
	}
}
