package persist

import (
	"flag"
	"testing"

	"heim/backend"
)

var dsn = flag.String("dsn", "postgres://heimtest:heimtest@localhost/heimtest", "")

func TestBackend(t *testing.T) {
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

	factory := func() backend.Backend {
		return b
	}

	backend.IntegrationTest(t, factory)
}
