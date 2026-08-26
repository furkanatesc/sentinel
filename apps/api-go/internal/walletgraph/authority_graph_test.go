package walletgraph

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestBuildAuthorityGraph_HubRolesAndExclusion(t *testing.T) {
	rows := []store.AuthorityRow{
		{Authority: "F1", Mint: "mA", Symbol: "A", Role: "mint", SafetyScore: 40, FirstSeenTs: 1},
		{Authority: "F1", Mint: "mB", Symbol: "B", Role: "mint", SafetyScore: 40, FirstSeenTs: 2},
		{Authority: "F1", Mint: "mB", Symbol: "B", Role: "freeze", SafetyScore: 40, FirstSeenTs: 2},            // mB: mint+freeze → both
		{Authority: "11111111111111111111111111111111", Mint: "mC", Symbol: "C", Role: "mint", FirstSeenTs: 3}, // program → dışlanır
	}
	g := BuildAuthorityGraph(rows)
	// authority_wallet hub + token node'ları (program hariç).
	var hub, both bool
	for _, n := range g.Nodes {
		if n.ID == "auth:F1" && n.Type == "authority_wallet" {
			hub = true
		}
		if n.Address == "11111111111111111111111111111111" {
			t.Fatalf("program authority node üretilmemeli")
		}
	}
	for _, e := range g.Edges {
		if e.Type == "controls_authority" && e.Source == "auth:F1" && e.Target == "tok:mB" && e.Role == "both" {
			both = true
		}
	}
	if !hub {
		t.Fatalf("authority_wallet hub node beklenir")
	}
	if !both {
		t.Fatalf("mB edge rolü 'both' beklenir (mint+freeze)")
	}
}

func TestBuildAuthorityGraph_EmptyIsEmpty(t *testing.T) {
	g := BuildAuthorityGraph(nil)
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Fatalf("boş girdi → boş graph")
	}
}
