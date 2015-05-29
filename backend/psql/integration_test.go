package psql

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/backend/cluster/clustertest"
	"euphoria.io/heim/proto"
	"euphoria.io/scope"

	"github.com/rubenv/sql-migrate"
)

var dsn = flag.String("dsn", "postgres://heimtest:heimtest@localhost/heimtest", "")

func TestBackend(t *testing.T) {
	etcd, err := clustertest.StartEtcd()
	if err != nil {
		t.Fatal(err)
	}
	if etcd == nil {
		t.Fatal("etcd not available in PATH, can't test backend")
	}
	defer etcd.Shutdown()

	dsn := *dsn
	if env := os.Getenv("DSN"); env != "" {
		// for running in CI container
		dsn = env
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %s", err)
	}

	// Drop all tables.
	for _, item := range schema {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + item.Name); err != nil {
			t.Fatalf("failed to drop table %s: %s", item.Name, err)
		}
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS gorp_migrations"); err != nil {
		t.Fatal(err)
	}

	// Recreate all tables.
	src := migrate.FileMigrationSource{"migrations"}
	if _, err := migrate.Exec(db, "postgres", src, migrate.Up); err != nil {
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
		b, err := NewBackend(scope.New(), dsn, c, desc)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	// Run test suite.
	backend.IntegrationTest(t, factory)
}
