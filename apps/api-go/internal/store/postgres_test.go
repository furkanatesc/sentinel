package store

import (
	"context"
	"os"
	"testing"
)

func TestPostgresListSeeded(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL unset — postgres integration test atlandı")
	}
	ctx := context.Background()
	b, cleanup, err := OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	defer cleanup()

	rows, err := b.Strategies.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 6 {
		t.Fatalf("rows = %d, want 6 (seed)", len(rows))
	}
}
