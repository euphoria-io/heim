package psql

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"heim/backend"
	"heim/backend/cluster"
	"heim/backend/cluster/clustertest"
	"heim/proto"

	"github.com/rubenv/sql-migrate"

	. "github.com/smartystreets/goconvey/convey"
)

var dsn = flag.String("dsn", "postgres://heimtest:heimtest@localhost/heimtest", "")

func TestBackend(t *testing.T) {
	// for running in CI container
	dsn := *dsn
	if env := os.Getenv("DSN"); env != "" {
		dsn = env
	}

	etcd, err := clustertest.StartEtcd()
	if err != nil {
		t.Fatal(err)
	}
	if etcd == nil {
		t.Fatal("etcd not available in PATH, can't test backend")
	}
	defer etcd.Shutdown()

	// Set up a backend in order to instantiate the DB.
	c := etcd.Join("/test", "bootstrap", "")
	desc := &cluster.PeerDesc{ID: "bootstrap"}
	b, err := NewBackend(dsn, c, desc)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	if err := b.DbMap.DropTablesIfExists(); err != nil {
		t.Fatal(err)
	}

	if _, err := b.DbMap.Exec("DROP TABLE IF EXISTS gorp_migrations"); err != nil {
		t.Fatal(err)
	}

	src := migrate.FileMigrationSource{"migrations"}
	if _, err := migrate.Exec(b.DB, "postgres", src, migrate.Up); err != nil {
		t.Fatal(err)
	}

	// Factory for test cases to generate fresh backends.
	iter := 0
	factory := func() proto.Backend {
		iter++
		c := etcd.Join("/test", "testcase", fmt.Sprintf("iter%d", iter))
		desc := &cluster.PeerDesc{
			ID:      "testcase",
			Era:     fmt.Sprintf("iter%d", iter),
			Version: "testver",
		}
		b, err := NewBackend(dsn, c, desc)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	Convey("Integration test suite", t, func() {
		// Run test suite.
		backend.IntegrationTest(factory)
	})
}
