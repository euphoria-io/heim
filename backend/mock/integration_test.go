package mock

import (
	"testing"

	"heim/backend"
	"heim/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTestBackend(t *testing.T) {
	Convey("Integration test suite", t, func() {
		backend.IntegrationTest(func() proto.Backend { return &TestBackend{} })
	})
}
