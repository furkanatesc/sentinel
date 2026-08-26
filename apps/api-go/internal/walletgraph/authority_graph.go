package walletgraph

import "github.com/furkanatesc/sentinel/apps/api-go/internal/store"

// BuildAuthorityGraph, controls_authority kümelerini kurar (saf; ağsız/DB'siz). Bilinen-program
// authority'lerini dışlar. authority_wallet hub + controls_authority edge (rol: mint/freeze/both).
func BuildAuthorityGraph(rows []store.AuthorityRow) store.WalletGraphResult {
	nodes := map[string]store.GraphNode{}
	edges := map[string]store.GraphEdge{}
	degree := map[string]map[string]struct{}{} // authority → distinct mint (risk)
	roleByEdge := map[string]string{}          // "auth|mint" → birleşik rol
	for _, r := range rows {
		if IsProgramAuthority(r.Authority) {
			continue
		}
		if degree[r.Authority] == nil {
			degree[r.Authority] = map[string]struct{}{}
		}
		degree[r.Authority][r.Mint] = struct{}{}
		key := r.Authority + "|" + r.Mint
		roleByEdge[key] = mergeRole(roleByEdge[key], r.Role)
	}
	for _, r := range rows {
		if IsProgramAuthority(r.Authority) {
			continue
		}
		ts := rfc3339(r.FirstSeenTs)
		authID, tokID := "auth:"+r.Authority, "tok:"+r.Mint
		deg := len(degree[r.Authority])
		nodes[authID] = store.GraphNode{ID: authID, Type: "authority_wallet", Label: shortAddr(r.Authority),
			Address: r.Authority, RiskLevel: store.ScoreToLevel(float64(100 - min(deg*20, 100))), FirstSeen: ts, LastSeen: ts}
		nodes[tokID] = store.GraphNode{ID: tokID, Type: "token", Label: r.Symbol,
			Address: r.Mint, RiskLevel: store.ScoreToLevel(r.SafetyScore), FirstSeen: ts, LastSeen: ts}
		eID := "e:ctrl:" + r.Authority + ":" + r.Mint
		edges[eID] = store.GraphEdge{ID: eID, Source: authID, Target: tokID, Type: "controls_authority",
			Role: roleByEdge[r.Authority+"|"+r.Mint]}
	}
	return store.WalletGraphResult{Nodes: mapNodes(nodes), Edges: mapEdges(edges)}
}

// mergeRole, aynı (authority,token) için mint+freeze rollerini "both"a birleştirir.
func mergeRole(existing, incoming string) string {
	if existing == "" || existing == incoming {
		return incoming
	}
	return "both"
}
