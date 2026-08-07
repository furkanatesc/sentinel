package market

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingLimiter, Wait çağrılarını ve her Wait anında sunucunun gördüğü istek
// sayısını kaydeder (sıralamayı — Wait HTTP'den önce mi — doğrulamak için).
type countingLimiter struct {
	mu        sync.Mutex
	calls     int
	err       error
	serverReq *atomic.Int32
	reqAtWait []int32
}

func (l *countingLimiter) Wait(context.Context) error {
	l.mu.Lock()
	l.calls++
	if l.serverReq != nil {
		l.reqAtWait = append(l.reqAtWait, l.serverReq.Load())
	}
	l.mu.Unlock()
	return l.err
}

// statusServer, ilk failStatus'u status kadar döner, sonra 200 + new_pools gövdesi.
func statusServer(t *testing.T, failStatus, failTimes int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	body, err := os.ReadFile("testdata/new_pools.json")
	if err != nil {
		t.Fatal(err)
	}
	var reqs atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := reqs.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if failTimes < 0 || int(n) <= failTimes {
			w.WriteHeader(failStatus)
			return
		}
		w.Write(body)
	}))
	return srv, &reqs
}

func TestGetJSONWaitsBeforeEachRequest(t *testing.T) {
	srv, reqs := statusServer(t, 0, 0) // hep 200
	defer srv.Close()
	lim := &countingLimiter{serverReq: reqs}
	c := NewGeckoTerminalClient(srv.URL, srv.Client(), WithLimiter(lim))

	if _, err := c.NewPools(context.Background()); err != nil {
		t.Fatal(err)
	}
	if lim.calls != 1 {
		t.Fatalf("limiter.Wait çağrı=%d, want 1", lim.calls)
	}
	if len(lim.reqAtWait) != 1 || lim.reqAtWait[0] != 0 {
		t.Fatalf("Wait, HTTP isteğinden ÖNCE çağrılmalı; reqAtWait=%v", lim.reqAtWait)
	}
}

func TestGetJSONLimiterErrorSkipsRequest(t *testing.T) {
	srv, reqs := statusServer(t, 0, 0)
	defer srv.Close()
	wantErr := errors.New("limit blok")
	lim := &countingLimiter{err: wantErr, serverReq: reqs}
	c := NewGeckoTerminalClient(srv.URL, srv.Client(), WithLimiter(lim))

	_, err := c.NewPools(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("limiter hatası yayılmalı: %v", err)
	}
	if got := reqs.Load(); got != 0 {
		t.Fatalf("limiter hata verince HTTP isteği YAPILMAMALI, got=%d", got)
	}
}

func TestGetJSONRetriesOn429ThenSucceeds(t *testing.T) {
	srv, reqs := statusServer(t, http.StatusTooManyRequests, 2) // ilk 2 → 429, 3. → 200
	defer srv.Close()
	lim := &countingLimiter{serverReq: reqs}
	c := NewGeckoTerminalClient(srv.URL, srv.Client(), WithLimiter(lim))
	c.sleep = func(context.Context, time.Duration) error { return nil } // gerçek beklemeyi atla

	pools, err := c.NewPools(context.Background())
	if err != nil {
		t.Fatalf("429 sonrası retry başarılı olmalı: %v", err)
	}
	if len(pools) == 0 {
		t.Fatal("başarılı yanıt parse edilmeli")
	}
	if got := reqs.Load(); got != 3 {
		t.Fatalf("istek sayısı=%d, want 3 (2×429 + 1×200)", got)
	}
	if lim.calls != 3 {
		t.Fatalf("her deneme limiter'dan token almalı; calls=%d want 3", lim.calls)
	}
}

func TestGetJSONExhausts429ReturnsError(t *testing.T) {
	srv, reqs := statusServer(t, http.StatusTooManyRequests, -1) // hep 429
	defer srv.Close()
	c := NewGeckoTerminalClient(srv.URL, srv.Client())
	c.sleep = func(context.Context, time.Duration) error { return nil }

	if _, err := c.NewPools(context.Background()); err == nil {
		t.Fatal("kalıcı 429 hata dönmeli")
	}
	if got := reqs.Load(); got != gtMaxAttempts {
		t.Fatalf("istek sayısı=%d, want %d (bounded retry)", got, gtMaxAttempts)
	}
}

func TestGetJSONBackoffCancelAborts(t *testing.T) {
	srv, reqs := statusServer(t, http.StatusTooManyRequests, -1) // hep 429
	defer srv.Close()
	wantErr := errors.New("ctx iptal")
	c := NewGeckoTerminalClient(srv.URL, srv.Client())
	c.sleep = func(context.Context, time.Duration) error { return wantErr } // backoff sırasında iptal

	_, err := c.NewPools(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("backoff iptali yayılmalı: %v", err)
	}
	if got := reqs.Load(); got != 1 {
		t.Fatalf("iptalden sonra yeni deneme YAPILMAMALI; istek=%d want 1", got)
	}
}

func TestGetJSONNoRetryOnNon429(t *testing.T) {
	srv, reqs := statusServer(t, http.StatusInternalServerError, -1) // hep 500
	defer srv.Close()
	c := NewGeckoTerminalClient(srv.URL, srv.Client())
	c.sleep = func(context.Context, time.Duration) error { return nil }

	if _, err := c.NewPools(context.Background()); err == nil {
		t.Fatal("500 hata dönmeli")
	}
	if got := reqs.Load(); got != 1 {
		t.Fatalf("429 dışı statü retry ETMEMELİ; istek=%d want 1", got)
	}
}
