package psql

import (
	"database/sql"
	"flag"
	"os"
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/cluster/clustertest"
	"euphoria.io/heim/proto"

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

	// Define backend factory.
	var b *Backend
	defer func() {
		if b != nil {
			b.Close()
		}
	}()
	factory := func(heim *proto.Heim) (proto.Backend, error) {
		if b == nil {
			// Use a real etcd cluster.
			// TODO: do we have to?
			heim.Cluster = etcd.Join("/test", "testcase", "era")
			desc := &cluster.PeerDesc{
				ID:      "testcase",
				Era:     "era",
				Version: "testver",
			}

			b, err = NewBackend(heim, dsn, desc)
			if err != nil {
				return nil, err
			}
		}
		return &nonClosingBackend{b}, nil
	}

	// Run test suite.
	backend.IntegrationTest(t, factory)
}

type nonClosingBackend struct {
	proto.Backend
}

func (nonClosingBackend) Close() {}
