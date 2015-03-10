package mock

import (
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTestBackend(t *testing.T) {
	Convey("Integration test suite", t, func() {
		backend.IntegrationTest(func() proto.Backend { return &TestBackend{} })
	})
}
