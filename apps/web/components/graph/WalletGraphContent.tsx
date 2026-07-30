"use client";
import { useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { useWalletGraph } from "@/lib/hooks/queries";
import { toCytoscapeElements } from "@/lib/graph/elements";
import { buildStylesheet } from "@/lib/graph/stylesheet";
import { EMPTY_GRAPH_FILTERS } from "@/lib/api/types";
import type { GraphFilters as GF, WalletGraph } from "@/lib/api/types";
import { GraphFilters } from "./GraphFilters";
import { GraphLegend } from "./GraphLegend";
import { NodeDetailPanel } from "./NodeDetailPanel";
import { Skeleton } from "@/components/ui/skeleton";

const WalletGraphCanvas = dynamic(() => import("./WalletGraphCanvas").then((m) => m.WalletGraphCanvas), {
  ssr: false, loading: () => <Skeleton className="h-[600px] w-full" />,
});

const EMPTY_GRAPH: WalletGraph = { nodes: [], edges: [] };

export function WalletGraphContent() {
  const { data } = useWalletGraph();
  const graph = data ?? EMPTY_GRAPH;
  const [filters, setFilters] = useState<GF>(EMPTY_GRAPH_FILTERS);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const stylesheet = useMemo(() => buildStylesheet(), []);
  const elements = useMemo(() => toCytoscapeElements(graph, filters), [graph, filters]);
  const visibleIds = useMemo(() => new Set(elements.filter((e) => !e.data.source).map((e) => e.data.id as string)), [elements]);
  const effectiveSelectedId = selectedId && visibleIds.has(selectedId) ? selectedId : null;
  const selectedNode = graph.nodes.find((n) => n.id === effectiveSelectedId) ?? null;
  return (
    <div className="space-y-4">
      <h1>Cüzdan Grafiği</h1>
      <GraphFilters value={filters} onChange={setFilters} />
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
        <div className="xl:col-span-3 space-y-3">
          <WalletGraphCanvas elements={elements} stylesheet={stylesheet} onNodeSelect={setSelectedId} focusNodeId={effectiveSelectedId} />
          <GraphLegend />
        </div>
        <div><NodeDetailPanel node={selectedNode} graph={graph} /></div>
      </div>
    </div>
  );
}
