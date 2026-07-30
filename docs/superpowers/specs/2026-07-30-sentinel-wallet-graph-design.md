# SENTINEL Frontend — Increment 4 Design Spec
### Wallet Graph (İnteraktif On-chain Entity Graph)

- **Tarih:** 2026-07-30
- **Durum:** Onay bekliyor
- **Önkoşul:** Increment 1–3 master'da.
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 5), `ROADMAP.md` (§4 knowledge graph), `docs/progress.md`.

---

## 1. Amaç ve kapsam

`/wallet-graph` altında interaktif on-chain entity graph (Cytoscape.js): cüzdan/token/havuz düğümleri ve aralarındaki ilişkiler; creator↔funder↔token kümelerini görselleştirir (ROADMAP §4).

Bu artımda: **zoom/pan + node tıkla → sağ detay paneli + ilişki (edge) filtresi + risk filtresi + node expand (komşu odağı)** ve **tam veri modeli** (8 node + 9 edge tipi, config-driven). Graph mock'tan gelir; backend gelince aynı seam gerçek graph API'sine geçer.

Kapsam dışı (sonraki artım): path finder, wallet compare, export (PNG/veri), cluster highlight, time-range filter.

---

## 2. Clean Code & SOLID ilkeleri (ölçüt)

- **SRP:** Graph **mantığı bileşende değil** `lib/graph/`'te saf fonksiyonlarda (`toCytoscapeElements`, `neighborsOf`, `buildStylesheet`); `WalletGraphCanvas` yalnız Cytoscape render'ı; `GraphFilters`, `NodeDetailPanel`, `GraphLegend` ayrı.
- **OCP:** Node/edge tipleri `NODE_TYPE_DEFS` / `EDGE_TYPE_DEFS` registry'lerinden (renk/şekil/etiket + Cytoscape stylesheet + legend + filtreler bunlardan türer). Yeni tip = registry entry.
- **DIP:** Bileşenler `useWalletGraph()`/`getApi()` soyutlamasına bağımlı; mock import etmez.
- **ISP:** `NodeDetailPanel` yalnız seçili `GraphNode`; `GraphFilters` `{ value, onChange }`; `WalletGraphCanvas` `{ elements, stylesheet, onNodeSelect }`.
- **Test edilebilirlik:** Cytoscape canvas tabanlı (jsdom render edemez) → tüm çıkarım/filtre/stil mantığı saf lib fonksiyonlarında TDD ile test edilir; `WalletGraphCanvas` yalnız smoke (cytoscape mock'lanır).

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
```ts
export type GraphNodeType =
  | "creator_wallet" | "funding_wallet" | "token" | "liquidity_pool"
  | "trader_wallet" | "smart_wallet" | "suspicious_wallet" | "exchange_wallet";
export type GraphEdgeType =
  | "funded" | "created" | "bought" | "sold" | "transferred"
  | "provided_liquidity" | "removed_liquidity" | "shares_funder" | "controls_authority";

export interface GraphNode {
  id: string; type: GraphNodeType; label: string;
  address?: string; riskLevel: RiskLevel;   // @/lib/format
  balanceSol?: number; firstSeen: string; lastSeen: string;
}
export interface GraphEdge { id: string; source: string; target: string; type: GraphEdgeType; }
export interface WalletGraph { nodes: GraphNode[]; edges: GraphEdge[]; }

export interface GraphFilters { relationships: GraphEdgeType[]; risks: RiskLevel[]; } // boş = tümü
export const EMPTY_GRAPH_FILTERS: GraphFilters; // { relationships: [], risks: [] }
```
- `SentinelApi.getWalletGraph(): Promise<WalletGraph>` (mock: ~18-24 node; funder→creator→token→pool kümeleri + shares_funder bağları; ROADMAP §4 örneği).
- `httpApi.getWalletGraph` → `notReady`.
- `qk.walletGraph`; `useWalletGraph()` (queries.ts).

### 3.2 Saf graph mantığı (`lib/graph/` — SRP/OCP)
- `graph-defs.ts`:
  ```ts
  export interface NodeTypeDef { key: GraphNodeType; label: string; color: string; shape: string; }
  export const NODE_TYPE_DEFS: NodeTypeDef[];   // 8 tip
  export interface EdgeTypeDef { key: GraphEdgeType; label: string; color: string; }
  export const EDGE_TYPE_DEFS: EdgeTypeDef[];   // 9 tip
  ```
- `elements.ts`:
  ```ts
  export function neighborsOf(graph: WalletGraph, nodeId: string): Set<string>; // node + direct komşular
  export function toCytoscapeElements(graph: WalletGraph, filters: GraphFilters, focusNodeId: string | null): CyElement[];
  ```
  `toCytoscapeElements`: risk filtresi (boş değilse node.riskLevel ∈ risks), ilişki filtresi (boş değilse edge.type ∈ relationships), ucu elenen edge'leri düşür, `focusNodeId` varsa komşu olmayan node/edge'lere `faded` class ekle. Çıktı Cytoscape formatı `{ data, classes }[]`.
- `stylesheet.ts`:
  ```ts
  export function buildStylesheet(): CyStyle[]; // NODE_TYPE_DEFS→node stili, EDGE_TYPE_DEFS→edge stili, risk→border, .faded→opacity, :selected
  ```

### 3.3 Bileşenler (`components/graph/`)
```
components/graph/
├─ WalletGraphCanvas.tsx    # "use client"; Cytoscape render (ref+useEffect); {elements,stylesheet,onNodeSelect}
├─ GraphFilters.tsx         # ilişki (edge) çipleri + risk çipleri + Temizle; {value,onChange}
├─ GraphLegend.tsx          # NODE_TYPE_DEFS/EDGE_TYPE_DEFS'ten legend
├─ NodeDetailPanel.tsx      # seçili node detayı (adres/WalletAddress, tip, risk, balance, ilk/son, komşular, Token linki)
└─ WalletGraphContent.tsx   # kompozisyon: filtreler + canvas + legend + detay panel; useWalletGraph
lib/graph/{graph-defs,elements,stylesheet}.ts
app/(app)/wallet-graph/page.tsx  # server: getWalletGraph prefetch + hydration
```
- **WalletGraphCanvas:** `cytoscape` (yeni dep) init; `elements`/`stylesheet` prop; layout `cose` (force) veya `concentric`; `cy.on("tap","node",...)` → `onNodeSelect(id)`; zoom/pan yerleşik; `node expand` = seçili node komşularına fit/odak (`focusNodeId`). Dinamik import (`next/dynamic`, `ssr:false`) ile SSR'dan kaçınılır.
- **GraphFilters:** ilişki çipleri (`EDGE_TYPE_DEFS`), risk çipleri, "Temizle" → `EMPTY_GRAPH_FILTERS`.
- **NodeDetailPanel:** seçili `GraphNode`: entity tipi (NODE_TYPE_DEFS label), kısaltılmış adres (`WalletAddress`), risk, balance, ilk/son aktivite, en önemli komşular (`neighborsOf`), token node ise "Token Detayına Git".
- **WalletGraphContent:** `"use client"`; `useWalletGraph()`; filter state + `focusNodeId` + seçili node; `toCytoscapeElements` → canvas.

### 3.4 Sayfa & bağımlılık
- `app/(app)/wallet-graph/page.tsx` — server, `getWalletGraph` prefetch + hydration + `<WalletGraphContent/>`. Placeholder route değişir.
- Yeni dep: **`cytoscape`** (+ tip `@types/cytoscape`). Client-only (dynamic import ssr:false).

---

## 4. Kapsam dışı (bilinçli)
Path finder, wallet compare, export image/data, cluster highlight, time-range filter, smart-money analizi. Token Detail içindeki "Cüzdan Grafiği" sekmesi hâlâ placeholder (token-scoped graph sonraki artım). Gerçek backend graph API'si (mock; seam hazır).

## 5. Test stratejisi (TDD)
- `graph-defs`: 8 node + 9 edge tipinin hepsi tanımlı (label+color).
- `elements.ts` (saf, ana test yükü):
  - `toCytoscapeElements` boş filtre = tüm node+edge; ilişki filtresi edge tipini süzer; risk filtresi node'u süzer + ucu elenen edge düşer; `focusNodeId` komşu olmayanlara `faded` ekler.
  - `neighborsOf` node + direct komşuları döndürür.
- `stylesheet.ts`: her node/edge tipi için selector üretir (`buildStylesheet` yapısı).
- `getWalletGraph` adapter + `useWalletGraph` hook.
- `GraphFilters` çip toggle + Temizle → `EMPTY_GRAPH_FILTERS`.
- `NodeDetailPanel` seçili node alanlarını + (token ise) linki render eder.
- `WalletGraphCanvas`: **smoke** — cytoscape mock'lanır; container div render olur, crash yok.
- `WalletGraphContent` smoke.

## 6. Kabul kriterleri
1. Sidebar "Cüzdan Grafiği" → `/wallet-graph`: graph canvas + filtreler + legend + (seçim yokken) boş detay ipucu.
2. Zoom/pan çalışır; node'a tıklayınca sağ panel node detayını gösterir + komşuları odaklanır (faded).
3. İlişki filtresi edge tiplerini, risk filtresi node'ları süzer; "Temizle" sıfırlar.
4. 8 node + 9 edge tipi renk/şekil/etiketiyle registry'den render edilir; legend gösterir.
5. Token node detayında "Token Detayına Git" → `/tokens/[symbol]`.
6. Tüm veri `component → useWalletGraph → getApi() → mock`; hiçbir bileşen mock import etmez.
7. Saf mantık (elements/neighbors/stylesheet/filters/defs) testli; canvas smoke; build başarılı; SOLID/clean ölçütü.
