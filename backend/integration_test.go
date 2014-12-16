package backend

import (
	"testing"
)

func TestTestBackend(t *testing.T) {
	IntegrationTest(t, func() Backend { return &TestBackend{} })
}
