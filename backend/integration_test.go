package backend_test

import (
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/mock"
	"euphoria.io/heim/proto"
)

func TestBackend(t *testing.T) {
	backend.IntegrationTest(t, func() proto.Backend { return &mock.TestBackend{} })
}
