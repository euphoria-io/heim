package psql

import (
	"flag"
	"os"
	"testing"

	"heim/backend"
	"heim/proto"

	. "github.com/smartystreets/goconvey/convey"
)

var dsn = flag.String("dsn", "postgres://heimtest:heimtest@localhost/heimtest", "")

func TestBackend(t *testing.T) {
	// for running in CI container
	dsn := *dsn
	if env := os.Getenv("DSN"); env != "" {
		dsn = env
	}

	Convey("Integration test suite", t, func() {
		b, err := NewBackend(dsn, "testver")
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
