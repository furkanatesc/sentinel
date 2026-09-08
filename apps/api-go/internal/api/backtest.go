package api

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// backtestHandler, POST /api/backtest'i Python backtest servisine (FastAPI) proxy'ler.
// serviceURL boşsa graceful 503 döner (frontend notReady dalı) — Railway 2. servis + env
// BACKTEST_SERVICE_URL kurulana dek prod davranışını bozmaz.
func backtestHandler(serviceURL string, timeout time.Duration) http.HandlerFunc {
	client := &http.Client{Timeout: timeout}
	return func(w http.ResponseWriter, r *http.Request) {
		if serviceURL == "" {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backtest service not configured"})
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, serviceURL+"/backtest", bytes.NewReader(body))
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream request build"})
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "backtest upstream unreachable"})
			return
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode) // upstream durumunu passthrough
		_, _ = w.Write(out)
	}
}
