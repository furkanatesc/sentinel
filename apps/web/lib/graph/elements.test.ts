import { toCytoscapeElements, neighborsOf } from "./elements";
import { EMPTY_GRAPH_FILTERS } from "@/lib/api/types";
import type { WalletGraph } from "@/lib/api/types";

const g: WalletGraph = {
  nodes: [
    { id: "F1", type: "funding_wallet", label: "Funder", riskLevel: "high", firstSeen: "x", lastSeen: "y" },
    { id: "C1", type: "creator_wallet", label: "Creator", riskLevel: "medium", firstSeen: "x", lastSeen: "y" },
    { id: "T1", type: "token", label: "PULSE", riskLevel: "good", firstSeen: "x", lastSeen: "y" },
  ],
  edges: [
    { id: "e1", source: "F1", target: "C1", type: "funded" },
    { id: "e2", source: "C1", target: "T1", type: "created" },
  ],
};

test("empty filters => all nodes+edges", () => {
  const els = toCytoscapeElements(g, EMPTY_GRAPH_FILTERS, null);
  expect(els.filter((e) => !e.data.source)).toHaveLength(3); // nodes
  expect(els.filter((e) => e.data.source)).toHaveLength(2);  // edges
});
test("relationship filter drops non-matching edges", () => {
  const els = toCytoscapeElements(g, { ...EMPTY_GRAPH_FILTERS, relationships: ["created"] }, null);
  expect(els.filter((e) => e.data.source).map((e) => e.data.id)).toEqual(["e2"]);
});
test("risk filter drops nodes and their edges", () => {
  const els = toCytoscapeElements(g, { ...EMPTY_GRAPH_FILTERS, risks: ["good"] }, null);
  const nodes = els.filter((e) => !e.data.source);
  expect(nodes.map((e) => e.data.id)).toEqual(["T1"]);
  expect(els.filter((e) => e.data.source)).toHaveLength(0); // both edges lose an endpoint
});
test("focus fades non-neighbors", () => {
  const els = toCytoscapeElements(g, EMPTY_GRAPH_FILTERS, "F1");
  const t1 = els.find((e) => e.data.id === "T1")!;
  const c1 = els.find((e) => e.data.id === "C1")!;
  expect(t1.classes).toContain("faded");     // not a neighbor of F1
  expect(c1.classes).not.toContain("faded"); // direct neighbor
});
test("neighborsOf returns node + direct neighbors", () => {
  expect([...neighborsOf(g, "C1")].sort()).toEqual(["C1", "F1", "T1"]);
});
