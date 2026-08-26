package walletgraph

import (
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// BuildGraph, küme satırlarından wallet graph kurar (saf; ağsız/DB'siz). CEX funder'ları dışlar.
// funding_wallet hub + funded/created edge'leri kümeyi temsil eder (shares_funder YAGNI).
func BuildGraph(rows []store.ClusterRow) store.WalletGraphResult {
	nodes := map[string]store.GraphNode{}
	edges := map[string]store.GraphEdge{}
	// funder başına distinct creator sayısı (risk için).
	degree := map[string]map[string]struct{}{}
	for _, r := range rows {
		if IsCEX(r.Funder) {
			continue // borsa çekimi, bundler değil.
		}
		if degree[r.Funder] == nil {
			degree[r.Funder] = map[string]struct{}{}
		}
		degree[r.Funder][r.Creator] = struct{}{}
	}
	for _, r := range rows {
		if IsCEX(r.Funder) {
			continue
		}
		ts := rfc3339(r.FirstSeenTs)
		fundID, walID, tokID := "fund:"+r.Funder, "wal:"+r.Creator, "tok:"+r.Mint
		deg := len(degree[r.Funder])
		nodes[fundID] = store.GraphNode{ID: fundID, Type: "funding_wallet", Label: shortAddr(r.Funder),
			Address: r.Funder, RiskLevel: store.ScoreToLevel(float64(100 - min(deg*20, 100))), FirstSeen: ts, LastSeen: ts}
		nodes[walID] = store.GraphNode{ID: walID, Type: "creator_wallet", Label: shortAddr(r.Creator),
			Address: r.Creator, RiskLevel: store.ScoreToLevel(r.ReputationScore), FirstSeen: ts, LastSeen: ts}
		nodes[tokID] = store.GraphNode{ID: tokID, Type: "token", Label: r.Symbol,
			Address: r.Mint, RiskLevel: store.ScoreToLevel(r.SafetyScore), FirstSeen: ts, LastSeen: ts}
		fE := "e:funded:" + r.Funder + ":" + r.Creator
		edges[fE] = store.GraphEdge{ID: fE, Source: fundID, Target: walID, Type: "funded"}
		cE := "e:created:" + r.Creator + ":" + r.Mint
		edges[cE] = store.GraphEdge{ID: cE, Source: walID, Target: tokID, Type: "created"}
	}
	return store.WalletGraphResult{Nodes: mapNodes(nodes), Edges: mapEdges(edges)}
}

func rfc3339(ts int64) string {
	if ts <= 0 {
		return "—"
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
func shortAddr(a string) string {
	if len(a) <= 10 {
		return a
	}
	return a[:4] + "…" + a[len(a)-4:]
}

func mapNodes(m map[string]store.GraphNode) []store.GraphNode {
	out := make([]store.GraphNode, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
func mapEdges(m map[string]store.GraphEdge) []store.GraphEdge {
	out := make([]store.GraphEdge, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
