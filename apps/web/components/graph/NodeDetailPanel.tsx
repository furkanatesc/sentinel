"use client";
import Link from "next/link";
import type { GraphNode, WalletGraph } from "@/lib/api/types";
import { NODE_TYPE_DEFS } from "@/lib/graph/graph-defs";
import { neighborsOf } from "@/lib/graph/elements";
import { riskMeta } from "@/lib/format";
import { WalletAddress } from "@/components/sentinel/WalletAddress";

const LABEL = Object.fromEntries(NODE_TYPE_DEFS.map((d) => [d.key, d.label]));

export function NodeDetailPanel({ node, graph }: { node: GraphNode | null; graph: WalletGraph }) {
  if (!node) return <div className="rounded-lg border border-dashed border-border bg-card p-6 text-center text-muted-foreground" style={{ fontSize: 12 }}>Detay için bir düğüm seç</div>;
  const rm = riskMeta[node.riskLevel];
  const neighbors = [...neighborsOf(graph, node.id)].filter((id) => id !== node.id);
  return (
    <div className="space-y-2 rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between">
        <h3>{node.label}</h3>
        <span style={{ color: rm.color, fontSize: 12, fontWeight: 600 }}>{rm.label}</span>
      </div>
      <div className="text-muted-foreground" style={{ fontSize: 12 }}>{LABEL[node.type]}</div>
      {node.address && <WalletAddress address={node.address} />}
      {node.balanceSol !== undefined && <div style={{ fontSize: 12 }}>Bakiye: <span className="font-mono">{node.balanceSol} SOL</span></div>}
      <div className="text-muted-foreground" style={{ fontSize: 11 }}>İlk: {node.firstSeen} · Son: {node.lastSeen}</div>
      <div style={{ fontSize: 12 }}>Bağlantılar: <span className="font-mono">{neighbors.length}</span></div>
      {node.type === "token" && (
        <Link href={`/tokens/${node.label}`} className="mt-2 inline-block rounded-md bg-primary px-3 py-1.5 text-primary-foreground" style={{ fontSize: 12, fontWeight: 500 }}>Token Detayına Git</Link>
      )}
      {node.type === "creator_wallet" && (
        <Link href={`/creators/${node.address ?? node.id}`} className="mt-2 inline-block rounded-md bg-primary px-3 py-1.5 text-primary-foreground" style={{ fontSize: 12, fontWeight: 500 }}>Creator Detayına Git</Link>
      )}
    </div>
  );
}
