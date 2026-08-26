package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/walletgraph"
)

func authorityGraphHandler(ts store.TokenStore, minCluster, maxDegree int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := ts.AuthorityGraphClusters(r.Context(), minCluster, maxDegree)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "authority graph unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, walletgraph.BuildAuthorityGraph(rows))
	}
}
