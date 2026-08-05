package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/ws"
)

func wsHandler(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hub == nil {
			http.Error(w, "ws unavailable", http.StatusServiceUnavailable)
			return
		}
		hub.ServeWS(w, r)
	}
}
