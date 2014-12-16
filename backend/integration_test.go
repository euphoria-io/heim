package backend

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTestBackend(t *testing.T) {
	Convey("Integration test", t,
		IntegrationTest(func() Backend { return &TestBackend{} }))
}
