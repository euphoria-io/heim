package persist

import (
	"testing"

	"heim/backend"
)

func TestBackend(t *testing.T) {
	// TODO: get from environment somehow
	dsn := "postgres://heimtest:heimtest@localhost/heimtest"
	b, err := NewBackend(dsn)
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
