package psql

import (
	"database/sql"
	"flag"
	"os"
	"testing"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"

	"github.com/rubenv/sql-migrate"
)

var dsn = flag.String("dsn", "postgres://heimtest:heimtest@localhost/heimtest", "")

func TestBackend(t *testing.T) {
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
		if _, err := db.Exec("DROP TABLE IF EXISTS " + item.Name + " CASCADE"); err != nil {
			t.Fatalf("failed to drop table %s: %s", item.Name, err)
		}
	}
	for _, table := range []string{"gorp_migrations", "stats_sessions_analyzed", "stats_sessions_global", "stats_sessions_per_room"} {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
			t.Fatal(err)
		}
	}
	for _, function := range []string{
		"stats_sessions_analyze()",
		"stats_sessions_global_find(timestamp with time zone, timestamp with time zone)",
		"stats_sessions_global_extend(timestamp with time zone, timestamp with time zone)",
		"stats_sessions_per_room_find(timestamp with time zone, timestamp with time zone)",
		"stats_sessions_per_room_extend(timestamp with time zone, timestamp with time zone)",
		"job_claim(text, text)",
		"job_steal(text, text)",
		"job_complete(bigint, integer, bytea)",
		"job_fail(bigint, integer, text, bytea)",
		"job_cancel(bigint)",
	} {
		if _, err := db.Exec("DROP FUNCTION IF EXISTS " + function); err != nil {
			t.Fatal(err)
		}
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
			heim.Cluster = &cluster.TestCluster{}
			heim.PeerDesc = &cluster.PeerDesc{
				ID:      "testcase",
				Era:     "era",
				Version: "testver",
			}

			b, err = NewBackend(heim, dsn)
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
