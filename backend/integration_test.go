package backend

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTestBackend(t *testing.T) {
	Convey("Integration test suite", t, func() {
		IntegrationTest(func() Backend { return &TestBackend{} })
	})
}
