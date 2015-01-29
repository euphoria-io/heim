package persist

import (
	"flag"
	"testing"

	"heim/backend"
	"heim/proto"

	. "github.com/smartystreets/goconvey/convey"
)

var dsn = flag.String("dsn", "postgres://heimtest:heimtest@localhost/heimtest", "")

func TestBackend(t *testing.T) {
	Convey("Integration test suite", t, func() {
		b, err := NewBackend(*dsn, "testver")
		if err != nil {
			t.Fatal(err)
		}

		if err := b.DbMap.DropTablesIfExists(); err != nil {
			t.Fatal(err)
		}

		t.Logf("creating schema")
		if err := b.createSchema(); err != nil {
			t.Fatal(err)
		}

		backend.IntegrationTest(func() proto.Backend { return b })
	})
}
