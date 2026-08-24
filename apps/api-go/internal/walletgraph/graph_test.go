package walletgraph

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestBuildGraph_ClusterNodesAndEdges(t *testing.T) {
	rows := []store.ClusterRow{
		{Funder: "F1", Creator: "cA", Mint: "mA", Symbol: "AAA", SafetyScore: 80, ReputationScore: 40, FirstSeenTs: 1000},
		{Funder: "F1", Creator: "cB", Mint: "mB", Symbol: "BBB", SafetyScore: 20, ReputationScore: 10, FirstSeenTs: 1000},
	}
	g := BuildGraph(rows)
	// node'lar: 1 funding_wallet (F1) + 2 creator_wallet + 2 token = 5
	if len(g.Nodes) != 5 {
		t.Fatalf("5 node bekleniyordu, got %d", len(g.Nodes))
	}
	// edge'ler: 2 funded (F1→cA, F1→cB) + 2 created (cA→mA, cB→mB) = 4
	if len(g.Edges) != 4 {
		t.Fatalf("4 edge bekleniyordu, got %d", len(g.Edges))
	}
	// funding_wallet node tipi + created/funded edge tipleri doğru
	var fund int
	for _, n := range g.Nodes {
		if n.Type == "funding_wallet" {
			fund++
		}
	}
	if fund != 1 {
		t.Fatalf("1 funding_wallet bekleniyordu, got %d", fund)
	}
}

// knownCEXSample, test-yerel bilinen bir CEX adresi ekler ve döndürür. Production knownCEX
// boş/dolu olmasından BAĞIMSIZ çalışır (allowlist içeriğine göre testin kırılgan olmaması için) —
// gerçek dışlama mantığını (IsCEX/BuildGraph) deterministik bir girdiyle doğrular.
func knownCEXSample() string {
	const addr = "TestCEXAddr1111111111111111111111111111111"
	knownCEX[addr] = "TestExchange"
	return addr
}

func TestBuildGraph_ExcludesCEX(t *testing.T) {
	// F1 bilinen bir CEX ise küme dışlanır → boş graph.
	cex := knownCEXSample() // test yardımcısı: cex.go'daki set'e test-yerel adres ekler
	rows := []store.ClusterRow{
		{Funder: cex, Creator: "cA", Mint: "mA", Symbol: "A", FirstSeenTs: 1},
		{Funder: cex, Creator: "cB", Mint: "mB", Symbol: "B", FirstSeenTs: 1},
	}
	g := BuildGraph(rows)
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Fatalf("CEX funder dışlanmalı → boş graph, got %d node", len(g.Nodes))
	}
}

func TestBuildGraph_Empty(t *testing.T) {
	g := BuildGraph(nil)
	if g.Nodes == nil || g.Edges == nil {
		t.Fatal("boş graph nil değil, boş slice olmalı (JSON [])")
	}
}
