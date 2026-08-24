package opportunity

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeOppStore struct {
	targets []store.OpportunityTarget
	updates []store.OpportunityUpdate
	failOn  string // bu mint'te UpdateOpportunity hata → izole edilmeli
}

func (f *fakeOppStore) OpportunityScoreTargets(_ context.Context, _ int) ([]store.OpportunityTarget, error) {
	return f.targets, nil
}
func (f *fakeOppStore) UpdateOpportunity(_ context.Context, u store.OpportunityUpdate) error {
	if u.Mint == f.failOn {
		return errors.New("boom")
	}
	f.updates = append(f.updates, u)
	return nil
}

func TestWorker_ScoresAndPersists_IsolatesError(t *testing.T) {
	fs := &fakeOppStore{
		targets: []store.OpportunityTarget{
			{Mint: "ok", Safety: 80, SafetyConf: 1, Liquidity: 1000, Momentum: 60},
			{Mint: "bad", Safety: 80, SafetyConf: 1, Liquidity: 1000, Momentum: 60},
		},
		failOn: "bad",
	}
	w := NewWorker(WorkerDeps{Store: fs, Limit: 10,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:    func() time.Time { return time.Unix(1000, 0) }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fs.updates) != 1 || fs.updates[0].Mint != "ok" {
		t.Fatalf("yalnız 'ok' persist edilmeli, got %+v", fs.updates)
	}
	if fs.updates[0].ScoredTs != 1000 {
		t.Fatalf("scoredTs=%d want 1000", fs.updates[0].ScoredTs)
	}
}
