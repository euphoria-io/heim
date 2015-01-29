package backend

import (
	"testing"

	"heim/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTestBackend(t *testing.T) {
	Convey("Integration test suite", t, func() {
		IntegrationTest(func() proto.Backend { return &TestBackend{} })
	})
}
