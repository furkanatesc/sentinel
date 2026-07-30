# SENTINEL Frontend — Increment 4 Implementation Plan
### Wallet Graph (İnteraktif On-chain Entity Graph)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`).

**Goal:** `/wallet-graph`'te Cytoscape.js interaktif entity graph: zoom/pan + node→detay paneli + ilişki/risk filtresi + node expand (komşu odağı), 8 node + 9 edge tipi config-driven — mock seam üzerinden.

**Architecture:** Seam genişler (`getWalletGraph`). Graph mantığı saf `lib/graph/` fonksiyonlarında (TDD), Cytoscape render'ı ince client bileşeni (dynamic import, ssr:false). Node/edge tipleri registry (OCP), bileşenler `useWalletGraph`/`getApi()` (DIP), dar prop'lar (ISP).

**Tech Stack:** Next 16 App Router, TS strict, Tailwind v4, shadcn/ui (Base UI), TanStack Query, **cytoscape** (+ @types/cytoscape), Vitest + RTL.

## Global Constraints
- `SENTINEL/apps/web/`; npm; branch `feat/wallet-graph`.
- **Dark-only**, **UI Türkçe** (teknik token/simge/adres hariç). Mono yalnız sayı/adres.
- **Veri kuralı:** hiçbir bileşen `lib/api/mock`'u import etmez → `useWalletGraph`/`getApi()`.
- **Clean Code & SOLID ölçütü:** graph mantığı lib'de (SRP), `NODE_TYPE_DEFS`/`EDGE_TYPE_DEFS` registry (OCP), DIP, ISP; küçük dosyalar; TDD; pristine test.
- `RiskLevel`/`riskMeta` `@/lib/format`'ten.
- Cytoscape client-only: `WalletGraphContent` `WalletGraphCanvas`'ı `next/dynamic` (`ssr:false`) ile yükler; canvas jsdom'da render edilemez → smoke testte cytoscape mock'lanır.
- Commit: her task sonunda; gövde şununla biter: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

---

### Task 1: Graph modeli + registry'ler + saf mantık (TDD)

**Files:**
- Modify: `apps/web/lib/api/types.ts` (Graph tipleri + EMPTY_GRAPH_FILTERS)
- Create: `apps/web/lib/graph/graph-defs.ts`, `elements.ts`, `stylesheet.ts`
- Test: `apps/web/lib/graph/elements.test.ts`, `graph-defs.test.ts`, `stylesheet.test.ts`

**Interfaces:**
- Produces: `GraphNodeType`, `GraphEdgeType`, `GraphNode`, `GraphEdge`, `WalletGraph`, `GraphFilters`, `EMPTY_GRAPH_FILTERS`, `NODE_TYPE_DEFS`, `EDGE_TYPE_DEFS`, `neighborsOf`, `toCytoscapeElements`, `buildStylesheet`, `CyElement`.

- [ ] **Step 1: `lib/api/types.ts`'e ekle** (RiskLevel import'u zaten var; ekleme yapma)

```ts
export type GraphNodeType =
  | "creator_wallet" | "funding_wallet" | "token" | "liquidity_pool"
  | "trader_wallet" | "smart_wallet" | "suspicious_wallet" | "exchange_wallet";
export type GraphEdgeType =
  | "funded" | "created" | "bought" | "sold" | "transferred"
  | "provided_liquidity" | "removed_liquidity" | "shares_funder" | "controls_authority";

export interface GraphNode {
  id: string; type: GraphNodeType; label: string;
  address?: string; riskLevel: RiskLevel; balanceSol?: number; firstSeen: string; lastSeen: string;
}
export interface GraphEdge { id: string; source: string; target: string; type: GraphEdgeType; }
export interface WalletGraph { nodes: GraphNode[]; edges: GraphEdge[]; }
export interface GraphFilters { relationships: GraphEdgeType[]; risks: RiskLevel[]; }
export const EMPTY_GRAPH_FILTERS: GraphFilters = { relationships: [], risks: [] };
```

- [ ] **Step 2: `lib/graph/graph-defs.ts` yaz**

```ts
import type { GraphNodeType, GraphEdgeType } from "@/lib/api/types";

export interface NodeTypeDef { key: GraphNodeType; label: string; color: string; shape: string; }
export const NODE_TYPE_DEFS: NodeTypeDef[] = [
  { key: "creator_wallet", label: "Creator Cüzdanı", color: "#7C5CFF", shape: "ellipse" },
  { key: "funding_wallet", label: "Fon Cüzdanı", color: "#3E9BFF", shape: "ellipse" },
  { key: "token", label: "Token", color: "#2FD98B", shape: "round-rectangle" },
  { key: "liquidity_pool", label: "Likidite Havuzu", color: "#FFB020", shape: "diamond" },
  { key: "trader_wallet", label: "Trader Cüzdanı", color: "#8A94A6", shape: "ellipse" },
  { key: "smart_wallet", label: "Akıllı Cüzdan", color: "#2FD98B", shape: "star" },
  { key: "suspicious_wallet", label: "Şüpheli Cüzdan", color: "#F0476B", shape: "ellipse" },
  { key: "exchange_wallet", label: "Borsa Cüzdanı", color: "#C4CBD8", shape: "hexagon" },
];

export interface EdgeTypeDef { key: GraphEdgeType; label: string; color: string; }
export const EDGE_TYPE_DEFS: EdgeTypeDef[] = [
  { key: "funded", label: "Fonladı", color: "#3E9BFF" },
  { key: "created", label: "Oluşturdu", color: "#7C5CFF" },
  { key: "bought", label: "Aldı", color: "#2FD98B" },
  { key: "sold", label: "Sattı", color: "#F0476B" },
  { key: "transferred", label: "Transfer", color: "#8A94A6" },
  { key: "provided_liquidity", label: "Likidite Sağladı", color: "#FFB020" },
  { key: "removed_liquidity", label: "Likidite Çekti", color: "#F0476B" },
  { key: "shares_funder", label: "Ortak Fonlayıcı", color: "#9B6BFF" },
  { key: "controls_authority", label: "Yetki Kontrolü", color: "#3E9BFF" },
];
```

- [ ] **Step 3: `lib/graph/elements.ts` yaz**

```ts
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

export function toCytoscapeElements(graph: WalletGraph, filters: GraphFilters, focusNodeId: string | null): CyElement[] {
  const riskOk = (r: WalletGraph["nodes"][number]) => filters.risks.length === 0 || filters.risks.includes(r.riskLevel);
  const relOk = (e: WalletGraph["edges"][number]) => filters.relationships.length === 0 || filters.relationships.includes(e.type);
  const nodes = graph.nodes.filter(riskOk);
  const nodeIds = new Set(nodes.map((n) => n.id));
  const edges = graph.edges.filter((e) => relOk(e) && nodeIds.has(e.source) && nodeIds.has(e.target));
  const focus = focusNodeId ? neighborsOf({ nodes, edges }, focusNodeId) : null;
  const cls = (parts: (string | false | null)[]) => parts.filter(Boolean).join(" ");
  const nodeEls: CyElement[] = nodes.map((n) => ({
    data: { id: n.id, label: n.label, type: n.type, risk: n.riskLevel },
    classes: cls([n.type, `risk-${n.riskLevel}`, focus && !focus.has(n.id) && "faded"]),
  }));
  const edgeEls: CyElement[] = edges.map((e) => ({
    data: { id: e.id, source: e.source, target: e.target, type: e.type },
    classes: cls([e.type, focus && !(focus.has(e.source) && focus.has(e.target)) && "faded"]),
  }));
  return [...nodeEls, ...edgeEls];
}
```

- [ ] **Step 4: `lib/graph/stylesheet.ts` yaz**

```ts
import { NODE_TYPE_DEFS, EDGE_TYPE_DEFS } from "./graph-defs";

export interface CyStyle { selector: string; style: Record<string, unknown>; }

export function buildStylesheet(): CyStyle[] {
  const styles: CyStyle[] = [
    { selector: "node", style: { label: "data(label)", color: "#C4CBD8", "font-size": 9, "text-valign": "bottom", "text-margin-y": 4, width: 26, height: 26, "border-width": 2, "border-color": "rgba(255,255,255,0.15)" } },
    { selector: "edge", style: { width: 1.5, "curve-style": "bezier", "target-arrow-shape": "triangle", "arrow-scale": 0.7, opacity: 0.7 } },
    { selector: ".faded", style: { opacity: 0.12 } },
    { selector: "node:selected", style: { "border-width": 3, "border-color": "#FFFFFF" } },
  ];
  for (const d of NODE_TYPE_DEFS) styles.push({ selector: `node.${d.key}`, style: { "background-color": d.color, shape: d.shape } });
  for (const d of EDGE_TYPE_DEFS) styles.push({ selector: `edge.${d.key}`, style: { "line-color": d.color, "target-arrow-color": d.color } });
  return styles;
}
```

- [ ] **Step 5: Testleri yaz** (`elements.test.ts`, `graph-defs.test.ts`, `stylesheet.test.ts`)

```ts
// elements.test.ts
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
```

```ts
// graph-defs.test.ts
import { NODE_TYPE_DEFS, EDGE_TYPE_DEFS } from "./graph-defs";
test("registries cover all 8 node + 9 edge types with label/color", () => {
  expect(NODE_TYPE_DEFS).toHaveLength(8);
  expect(EDGE_TYPE_DEFS).toHaveLength(9);
  for (const d of [...NODE_TYPE_DEFS, ...EDGE_TYPE_DEFS]) { expect(d.label).toBeTruthy(); expect(d.color).toMatch(/^#/); }
});
```

```ts
// stylesheet.test.ts
import { buildStylesheet } from "./stylesheet";
test("stylesheet has a selector per node and edge type", () => {
  const s = buildStylesheet();
  expect(s.some((x) => x.selector === "node.creator_wallet")).toBe(true);
  expect(s.some((x) => x.selector === "edge.shares_funder")).toBe(true);
  expect(s.some((x) => x.selector === ".faded")).toBe(true);
});
```

- [ ] **Step 6: RED→GREEN** — Run: `npm run test -- graph`; RED first, then implement, GREEN. Run `npm run build`; OK.
- [ ] **Step 7: Commit**

```bash
git add apps/web/lib/api/types.ts apps/web/lib/graph
git commit -m "feat(web): add wallet-graph model, type registries, and pure graph logic"
```

---

### Task 2: Veri katmanı — getWalletGraph + useWalletGraph (TDD)

**Files:**
- Modify: `apps/web/lib/api/contract.ts`, `mock.ts`, `http.ts`, `lib/get-query-client.ts`, `lib/hooks/queries.ts`
- Test: `apps/web/lib/api/wallet-graph.test.ts`, `apps/web/lib/hooks/wallet-graph.test.tsx`

**Interfaces:**
- Produces: `SentinelApi.getWalletGraph`; `qk.walletGraph`; `useWalletGraph()`.

- [ ] **Step 1: `contract.ts`** — `SentinelApi`'ye `getWalletGraph(): Promise<WalletGraph>;` + `import type { ..., WalletGraph } from "./types";`.

- [ ] **Step 2: Testi yaz** (`lib/api/wallet-graph.test.ts`)

```ts
import { mockApi } from "./mock";
test("getWalletGraph returns connected nodes and edges", async () => {
  const g = await mockApi.getWalletGraph();
  expect(g.nodes.length).toBeGreaterThan(10);
  expect(g.edges.length).toBeGreaterThan(10);
  const ids = new Set(g.nodes.map((n) => n.id));
  for (const e of g.edges) { expect(ids.has(e.source)).toBe(true); expect(ids.has(e.target)).toBe(true); }
  expect(g.nodes.some((n) => n.type === "token")).toBe(true);
});
```

- [ ] **Step 3: RED** — Run: `npm run test -- api/wallet-graph`; FAIL.

- [ ] **Step 4: `mock.ts`'e ekle** (deterministik graph; `tokens` seed sembolleri token node label'ı olur)

```ts
import type { GraphNode, GraphEdge, WalletGraph } from "./types";

function buildWalletGraph(): WalletGraph {
  const short = (s: string) => `${s}...${s.slice(-2)}`;
  const nodes: GraphNode[] = [
    { id: "F1", type: "funding_wallet", label: "Funder-1", address: short("Fnd1Qk"), riskLevel: "high", balanceSol: 420, firstSeen: "3g önce", lastSeen: "az önce" },
    { id: "F2", type: "funding_wallet", label: "Funder-2", address: short("Fnd2Rp"), riskLevel: "medium", balanceSol: 88, firstSeen: "5g önce", lastSeen: "1g önce" },
    { id: "C1", type: "creator_wallet", label: "Creator-A", address: short("CreAxz"), riskLevel: "high", balanceSol: 12, firstSeen: "2g önce", lastSeen: "az önce" },
    { id: "C2", type: "creator_wallet", label: "Creator-B", address: short("CreBmn"), riskLevel: "medium", balanceSol: 5, firstSeen: "2g önce", lastSeen: "3s önce" },
    { id: "C3", type: "creator_wallet", label: "Creator-C", address: short("CreCqw"), riskLevel: "critical", balanceSol: 1, firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "T1", type: "token", label: "PULSE", address: short("9xQeWv"), riskLevel: "good", firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "T2", type: "token", label: "GFROG", address: short("7mLp2c"), riskLevel: "critical", firstSeen: "12s önce", lastSeen: "az önce" },
    { id: "T3", type: "token", label: "LMN", address: short("Cd93Kf"), riskLevel: "strong", firstSeen: "5s önce", lastSeen: "az önce" },
    { id: "P1", type: "liquidity_pool", label: "Havuz-1", riskLevel: "good", firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "P2", type: "liquidity_pool", label: "Havuz-2", riskLevel: "critical", firstSeen: "12s önce", lastSeen: "az önce" },
    { id: "W1", type: "trader_wallet", label: "Trader-1", address: short("Trd1aa"), riskLevel: "medium", balanceSol: 34, firstSeen: "6g önce", lastSeen: "az önce" },
    { id: "W2", type: "trader_wallet", label: "Trader-2", address: short("Trd2bb"), riskLevel: "good", balanceSol: 51, firstSeen: "8g önce", lastSeen: "2s önce" },
    { id: "S1", type: "smart_wallet", label: "Smart-1", address: short("Smt1cc"), riskLevel: "strong", balanceSol: 210, firstSeen: "40g önce", lastSeen: "az önce" },
    { id: "X1", type: "suspicious_wallet", label: "Şüpheli-1", address: short("Sus1dd"), riskLevel: "critical", balanceSol: 3, firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "X2", type: "suspicious_wallet", label: "Şüpheli-2", address: short("Sus2ee"), riskLevel: "critical", balanceSol: 2, firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "E1", type: "exchange_wallet", label: "Borsa", address: short("Cex1ff"), riskLevel: "good", firstSeen: "200g önce", lastSeen: "az önce" },
  ];
  const edges: GraphEdge[] = [
    { id: "g1", source: "E1", target: "F1", type: "transferred" },
    { id: "g2", source: "F1", target: "C1", type: "funded" },
    { id: "g3", source: "F1", target: "C2", type: "funded" },
    { id: "g4", source: "F2", target: "C3", type: "funded" },
    { id: "g5", source: "F1", target: "F2", type: "shares_funder" },
    { id: "g6", source: "C1", target: "T1", type: "created" },
    { id: "g7", source: "C2", target: "T2", type: "created" },
    { id: "g8", source: "C3", target: "T3", type: "created" },
    { id: "g9", source: "C1", target: "T1", type: "controls_authority" },
    { id: "g10", source: "C1", target: "P1", type: "provided_liquidity" },
    { id: "g11", source: "C2", target: "P2", type: "provided_liquidity" },
    { id: "g12", source: "C2", target: "P2", type: "removed_liquidity" },
    { id: "g13", source: "W1", target: "T1", type: "bought" },
    { id: "g14", source: "W2", target: "T1", type: "bought" },
    { id: "g15", source: "S1", target: "T3", type: "bought" },
    { id: "g16", source: "W1", target: "T2", type: "sold" },
    { id: "g17", source: "X1", target: "T2", type: "bought" },
    { id: "g18", source: "X2", target: "T2", type: "bought" },
    { id: "g19", source: "F1", target: "X1", type: "funded" },
    { id: "g20", source: "F1", target: "X2", type: "funded" },
    { id: "g21", source: "X1", target: "X2", type: "transferred" },
    { id: "g22", source: "C1", target: "T2", type: "sold" },
  ];
  return { nodes, edges };
}
const walletGraph = buildWalletGraph();
```
Ve `mockApi`'ye: `getWalletGraph: () => delay(walletGraph),`.

- [ ] **Step 5: `http.ts`** — `getWalletGraph: notReady,`.
- [ ] **Step 6: `get-query-client.ts`** — `qk`'ye `walletGraph: ["wallet-graph"] as const,`.
- [ ] **Step 7: `queries.ts`** —

```ts
export function useWalletGraph() {
  return useQuery({ queryKey: qk.walletGraph, queryFn: () => getApi().getWalletGraph() });
}
```

- [ ] **Step 8: Hook testi** (`lib/hooks/wallet-graph.test.tsx`)

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useWalletGraph } from "./queries";
function Probe() { const { data } = useWalletGraph(); return <div>n:{data?.nodes.length ?? 0}</div>; }
test("useWalletGraph loads the graph", async () => {
  render(<QueryClientProvider client={getQueryClient()}><Probe /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText(/n:1[0-9]/)).toBeInTheDocument());
});
```

- [ ] **Step 9: GREEN + build** — Run: `npm run test -- wallet-graph`; PASS. Build OK.
- [ ] **Step 10: Commit**

```bash
git add apps/web/lib/api apps/web/lib/get-query-client.ts apps/web/lib/hooks
git commit -m "feat(web): add getWalletGraph seam and useWalletGraph hook"
```

---

### Task 3: cytoscape dep + WalletGraphCanvas + GraphLegend (TDD/smoke)

**Files:**
- Add dep: `cytoscape` + `@types/cytoscape` (dev)
- Create: `apps/web/components/graph/WalletGraphCanvas.tsx`, `GraphLegend.tsx`
- Test: `apps/web/components/graph/WalletGraphCanvas.test.tsx`, `GraphLegend.test.tsx`

**Interfaces:**
- Consumes: `cytoscape`, `CyElement`, `CyStyle`, `NODE_TYPE_DEFS`/`EDGE_TYPE_DEFS`.
- Produces: `<WalletGraphCanvas elements stylesheet onNodeSelect/>`, `<GraphLegend/>`.

- [ ] **Step 1: dep ekle** — `npm i cytoscape && npm i -D @types/cytoscape`.

- [ ] **Step 2: `WalletGraphCanvas.tsx`** (`"use client"`; SRP: yalnız render)

```tsx
"use client";
import { useEffect, useRef } from "react";
import cytoscape, { type Core } from "cytoscape";
import type { CyElement } from "@/lib/graph/elements";
import type { CyStyle } from "@/lib/graph/stylesheet";

export function WalletGraphCanvas({ elements, stylesheet, onNodeSelect }: {
  elements: CyElement[]; stylesheet: CyStyle[]; onNodeSelect: (id: string | null) => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const cbRef = useRef(onNodeSelect);
  cbRef.current = onNodeSelect;
  useEffect(() => {
    if (!ref.current) return;
    const cy: Core = cytoscape({
      container: ref.current,
      elements: elements as cytoscape.ElementDefinition[],
      style: stylesheet as cytoscape.Stylesheet[],
      layout: { name: "cose", animate: false, padding: 30 },
      wheelSensitivity: 0.2,
    });
    cy.on("tap", "node", (evt) => cbRef.current(evt.target.id()));
    cy.on("tap", (evt) => { if (evt.target === cy) cbRef.current(null); });
    return () => cy.destroy();
  }, [elements, stylesheet]);
  return <div ref={ref} data-testid="graph-canvas" className="h-[600px] w-full rounded-lg border border-border bg-card" />;
}
```

- [ ] **Step 3: `GraphLegend.tsx`** (registry-driven)

```tsx
import { NODE_TYPE_DEFS, EDGE_TYPE_DEFS } from "@/lib/graph/graph-defs";

export function GraphLegend() {
  return (
    <div className="rounded-lg border border-border bg-card p-3" style={{ fontSize: 11 }}>
      <div className="mb-2 flex flex-wrap gap-3">
        <span className="text-muted-foreground">Düğümler:</span>
        {NODE_TYPE_DEFS.map((d) => (
          <span key={d.key} className="flex items-center gap-1.5"><span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: d.color }} />{d.label}</span>
        ))}
      </div>
      <div className="flex flex-wrap gap-3">
        <span className="text-muted-foreground">İlişkiler:</span>
        {EDGE_TYPE_DEFS.map((d) => (
          <span key={d.key} className="flex items-center gap-1.5"><span className="h-0.5 w-4" style={{ backgroundColor: d.color }} />{d.label}</span>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Testleri yaz** (canvas smoke = cytoscape mock; legend registry)

```tsx
// WalletGraphCanvas.test.tsx
import { render, screen } from "@testing-library/react";
import { WalletGraphCanvas } from "./WalletGraphCanvas";
const on = vi.fn();
vi.mock("cytoscape", () => ({ default: vi.fn(() => ({ on, destroy: vi.fn() })) }));
test("mounts the canvas container and inits cytoscape", async () => {
  const cytoscape = (await import("cytoscape")).default as unknown as ReturnType<typeof vi.fn>;
  render(<WalletGraphCanvas elements={[{ data: { id: "a" } }]} stylesheet={[]} onNodeSelect={() => {}} />);
  expect(screen.getByTestId("graph-canvas")).toBeInTheDocument();
  expect(cytoscape).toHaveBeenCalled();
});
```

```tsx
// GraphLegend.test.tsx
import { render, screen } from "@testing-library/react";
import { GraphLegend } from "./GraphLegend";
test("legend lists node and edge types", () => {
  render(<GraphLegend />);
  expect(screen.getByText("Creator Cüzdanı")).toBeInTheDocument();
  expect(screen.getByText("Ortak Fonlayıcı")).toBeInTheDocument();
});
```

- [ ] **Step 5: GREEN + build** — Run: `npm run test -- graph/WalletGraphCanvas graph/GraphLegend`; PASS. Run `npm run build`; OK (cytoscape client-only; canvas imported by content via next/dynamic in Task 5, so a stray SSR import must not exist — this component is only referenced through dynamic later).
- [ ] **Step 6: Commit**

```bash
git add apps/web/components/graph/WalletGraphCanvas.tsx apps/web/components/graph/GraphLegend.tsx apps/web/components/graph/WalletGraphCanvas.test.tsx apps/web/components/graph/GraphLegend.test.tsx apps/web/package.json apps/web/package-lock.json
git commit -m "feat(web): add cytoscape canvas and graph legend"
```

---

### Task 4: GraphFilters + NodeDetailPanel (TDD)

**Files:**
- Create: `apps/web/components/graph/GraphFilters.tsx`, `NodeDetailPanel.tsx`
- Test: `apps/web/components/graph/GraphFilters.test.tsx`, `NodeDetailPanel.test.tsx`

**Interfaces:**
- Consumes: `GraphFilters`, `EMPTY_GRAPH_FILTERS`, `EDGE_TYPE_DEFS`, `riskMeta`/`RiskLevel`, `GraphNode`, `WalletGraph`, `neighborsOf`, `NODE_TYPE_DEFS`, `WalletAddress`.
- Produces: `<GraphFilters value onChange/>`, `<NodeDetailPanel node graph onClose/>`.

- [ ] **Step 1: `GraphFilters.tsx`** (`"use client"`; ilişki + risk çipleri; controlled)

```tsx
"use client";
import { X } from "lucide-react";
import type { GraphFilters as GF, GraphEdgeType } from "@/lib/api/types";
import { EMPTY_GRAPH_FILTERS } from "@/lib/api/types";
import { EDGE_TYPE_DEFS } from "@/lib/graph/graph-defs";
import { riskMeta, type RiskLevel } from "@/lib/format";

const RISKS = Object.keys(riskMeta) as RiskLevel[];

export function GraphFilters({ value, onChange }: { value: GF; onChange: (f: GF) => void }) {
  const toggle = <T,>(arr: T[], v: T): T[] => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
  return (
    <div className="space-y-2 rounded-lg border border-border bg-card p-3" style={{ fontSize: 12 }}>
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">İlişki:</span>
        {EDGE_TYPE_DEFS.map((d) => {
          const on = value.relationships.includes(d.key);
          return <button key={d.key} onClick={() => onChange({ ...value, relationships: toggle(value.relationships, d.key) })}
            className={`rounded px-2 py-1 ${on ? "bg-primary text-primary-foreground" : "bg-surface-2 text-muted-foreground hover:text-foreground"}`}>{d.label}</button>;
        })}
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">Risk:</span>
        {RISKS.map((r) => {
          const on = value.risks.includes(r); const m = riskMeta[r];
          return <button key={r} onClick={() => onChange({ ...value, risks: toggle(value.risks, r) })}
            className="rounded px-2 py-1" style={{ color: on ? m.color : undefined, backgroundColor: on ? m.bg : "var(--sentinel-surface-2)", border: on ? `1px solid ${m.border}` : "1px solid transparent" }}>{m.label}</button>;
        })}
        <button onClick={() => onChange(EMPTY_GRAPH_FILTERS)} className="ml-auto flex items-center gap-1 rounded-md border border-border px-2 py-1 text-muted-foreground hover:text-foreground"><X size={12} /> Temizle</button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: `NodeDetailPanel.tsx`** (`"use client"`; seçili node detayı)

```tsx
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
    </div>
  );
}
```

- [ ] **Step 3: Testleri yaz**

```tsx
// GraphFilters.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphFilters } from "./GraphFilters";
import { EMPTY_GRAPH_FILTERS } from "@/lib/api/types";
test("toggling a relationship chip emits it", async () => {
  const onChange = vi.fn();
  render(<GraphFilters value={EMPTY_GRAPH_FILTERS} onChange={onChange} />);
  await userEvent.click(screen.getByText("Oluşturdu"));
  expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ relationships: ["created"] }));
});
test("Temizle resets", async () => {
  const onChange = vi.fn();
  render(<GraphFilters value={{ relationships: ["created"], risks: [] }} onChange={onChange} />);
  await userEvent.click(screen.getByText("Temizle"));
  expect(onChange).toHaveBeenCalledWith(EMPTY_GRAPH_FILTERS);
});
```

```tsx
// NodeDetailPanel.test.tsx
import { render, screen } from "@testing-library/react";
import { NodeDetailPanel } from "./NodeDetailPanel";
import type { WalletGraph, GraphNode } from "@/lib/api/types";
const graph: WalletGraph = { nodes: [], edges: [{ id: "e", source: "T1", target: "C1", type: "created" }] };
const token: GraphNode = { id: "T1", type: "token", label: "PULSE", riskLevel: "good", firstSeen: "x", lastSeen: "y" };
test("empty state when no node", () => {
  render(<NodeDetailPanel node={null} graph={graph} />);
  expect(screen.getByText("Detay için bir düğüm seç")).toBeInTheDocument();
});
test("token node shows type and token link", () => {
  render(<NodeDetailPanel node={token} graph={graph} />);
  expect(screen.getByText("Token")).toBeInTheDocument();
  expect(screen.getByText("Token Detayına Git").closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
});
```

- [ ] **Step 4: GREEN + build** — Run: `npm run test -- graph/GraphFilters graph/NodeDetailPanel`; PASS. Build OK.
- [ ] **Step 5: Commit**

```bash
git add apps/web/components/graph/GraphFilters.tsx apps/web/components/graph/NodeDetailPanel.tsx apps/web/components/graph/GraphFilters.test.tsx apps/web/components/graph/NodeDetailPanel.test.tsx
git commit -m "feat(web): add graph filters and node detail panel"
```

---

### Task 5: WalletGraphContent + sayfa (entegrasyon, TDD)

**Files:**
- Create: `apps/web/components/graph/WalletGraphContent.tsx`, `apps/web/app/(app)/wallet-graph/page.tsx`
- Test: `apps/web/components/graph/WalletGraphContent.test.tsx`

**Interfaces:**
- Consumes: `useWalletGraph`, `toCytoscapeElements`, `buildStylesheet`, `EMPTY_GRAPH_FILTERS`, `GraphFilters`, `GraphLegend`, `NodeDetailPanel`, `WalletGraphCanvas` (dynamic), `getQueryClient/qk/getApi`.
- Produces: `<WalletGraphContent/>`, `/wallet-graph` sayfası.

- [ ] **Step 1: `WalletGraphContent.tsx`** (`"use client"`; canvas dynamic import ssr:false)

```tsx
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
  const elements = useMemo(() => toCytoscapeElements(graph, filters, selectedId), [graph, filters, selectedId]);
  const selectedNode = graph.nodes.find((n) => n.id === selectedId) ?? null;
  return (
    <div className="space-y-4">
      <h1>Cüzdan Grafiği</h1>
      <GraphFilters value={filters} onChange={setFilters} />
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
        <div className="xl:col-span-3 space-y-3">
          <WalletGraphCanvas elements={elements} stylesheet={stylesheet} onNodeSelect={setSelectedId} />
          <GraphLegend />
        </div>
        <div><NodeDetailPanel node={selectedNode} graph={graph} /></div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: `app/(app)/wallet-graph/page.tsx`** (server; prefetch; placeholder yerine)

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { WalletGraphContent } from "@/components/graph/WalletGraphContent";

export default async function WalletGraphPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.walletGraph, queryFn: () => getApi().getWalletGraph() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <WalletGraphContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 3: Testi yaz** (`WalletGraphContent.test.tsx` — canvas dynamic+cytoscape jsdom'da sorun çıkarmasın diye cytoscape mock)

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { WalletGraphContent } from "./WalletGraphContent";
vi.mock("cytoscape", () => ({ default: vi.fn(() => ({ on: vi.fn(), destroy: vi.fn() })) }));
test("renders title, filters, legend and empty detail hint", async () => {
  render(<QueryClientProvider client={getQueryClient()}><WalletGraphContent /></QueryClientProvider>);
  expect(screen.getByRole("heading", { name: "Cüzdan Grafiği" })).toBeInTheDocument();
  expect(screen.getByText("Detay için bir düğüm seç")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByText("Creator Cüzdanı")).toBeInTheDocument()); // legend
});
```
(Not: `next/dynamic` `ssr:false` bileşeni jsdom'da async yüklenir; cytoscape mock'landığı için canvas mount olur. Test dynamic yüklemeyi beklemek zorunda değil — legend/panel senkron render olur.)

- [ ] **Step 4: GREEN + build + manuel** — Run: `npm run test`; tüm suite PASS. Run `npm run build`; `/wallet-graph` derlenir. Manuel (dev): sidebar "Cüzdan Grafiği" → graph çizilir; node'a tıkla → sağ panel + komşu odağı; ilişki/risk filtresi süzer; Temizle sıfırlar; token node'da "Token Detayına Git".
- [ ] **Step 5: Commit**

```bash
git add apps/web/components/graph/WalletGraphContent.tsx "apps/web/app/(app)/wallet-graph/page.tsx" apps/web/components/graph/WalletGraphContent.test.tsx
git commit -m "feat(web): wire wallet-graph page with cytoscape canvas, filters and detail panel"
```

---

## Self-Review (yazar kontrolü)

**Spec coverage:** rota+prefetch (T5), seam+hook (T2), 8 node/9 edge registry (T1), saf toCytoscapeElements/neighbors/stylesheet (T1), zoom/pan+node tıkla+expand (Canvas T3 + Content T5), ilişki/risk filtresi (T1 logic + T4 UI), detay paneli+token linki (T4), legend (T3), mock import yasağı (kabul #6). ✅

**SOLID:** SRP (mantık lib'de; canvas yalnız render; filtre/panel/legend ayrı), OCP (NODE/EDGE_TYPE_DEFS → stylesheet+legend+filtre+render), DIP (useWalletGraph/getApi), ISP (dar prop'lar). ✅

**Placeholder taraması:** kod gerçek; wallet-graph artık gerçek sayfa. ✅

**Tip tutarlılığı:** `WalletGraph`/`GraphNode`/`GraphEdge`/`GraphFilters` tek kaynak (types.ts); `CyElement`/`CyStyle` elements/stylesheet'ten; `qk.walletGraph` prefetch (T5) + hook (T2) aynı; canvas cytoscape mock'u tüm testlerde tutarlı. ✅
