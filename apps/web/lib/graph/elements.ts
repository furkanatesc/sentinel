import type { WalletGraph, GraphFilters } from "@/lib/api/types";

export interface CyElement { data: Record<string, unknown>; classes?: string; }

export function neighborsOf(graph: WalletGraph, nodeId: string): Set<string> {
  const set = new Set<string>([nodeId]);
  for (const e of graph.edges) {
    if (e.source === nodeId) set.add(e.target);
    if (e.target === nodeId) set.add(e.source);
  }
  return set;
}

export function toCytoscapeElements(graph: WalletGraph, filters: GraphFilters): CyElement[] {
  const riskOk = (r: WalletGraph["nodes"][number]) => filters.risks.length === 0 || filters.risks.includes(r.riskLevel);
  const relOk = (e: WalletGraph["edges"][number]) => filters.relationships.length === 0 || filters.relationships.includes(e.type);
  const nodes = graph.nodes.filter(riskOk);
  const nodeIds = new Set(nodes.map((n) => n.id));
  const edges = graph.edges.filter((e) => relOk(e) && nodeIds.has(e.source) && nodeIds.has(e.target));
  const nodeEls: CyElement[] = nodes.map((n) => ({
    data: { id: n.id, label: n.label, type: n.type, risk: n.riskLevel },
    classes: `${n.type} risk-${n.riskLevel}`,
  }));
  const edgeEls: CyElement[] = edges.map((e) => ({
    data: { id: e.id, source: e.source, target: e.target, type: e.type },
    classes: e.type,
  }));
  return [...nodeEls, ...edgeEls];
}
