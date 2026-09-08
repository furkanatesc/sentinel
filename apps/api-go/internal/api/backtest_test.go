package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBacktestHandlerNotConfigured(t *testing.T) {
	h := backtestHandler("", time.Second)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodPost, "/api/backtest", strings.NewReader(`{"strategyId":"s1"}`)))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("serviceURL boş → 503 beklenir, got %d", rr.Code)
	}
}

func TestBacktestHandlerProxiesToUpstream(t *testing.T) {
	var gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/backtest" || r.Method != http.MethodPost {
			t.Errorf("beklenmeyen upstream isteği: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"metrics":{"trades":2}}`))
	}))
	defer upstream.Close()

	h := backtestHandler(upstream.URL, 5*time.Second)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodPost, "/api/backtest", strings.NewReader(`{"strategyId":"s1","maxPositions":4}`)))

	if rr.Code != http.StatusOK {
		t.Fatalf("passthrough status 200 beklenir, got %d", rr.Code)
	}
	if !strings.Contains(gotBody, "maxPositions") {
		t.Fatalf("istek gövdesi upstream'e iletilmeli, got %q", gotBody)
	}
	if !strings.Contains(rr.Body.String(), `"trades":2`) {
		t.Fatalf("upstream cevabı passthrough edilmeli, got %q", rr.Body.String())
	}
}
