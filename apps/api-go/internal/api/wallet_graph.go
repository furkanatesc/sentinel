package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/walletgraph"
)

func walletGraphHandler(ts store.TokenStore, minCluster, maxDegree int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := ts.WalletGraphClusters(r.Context(), minCluster, maxDegree)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "wallet graph unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, walletgraph.BuildGraph(rows))
	}
}
