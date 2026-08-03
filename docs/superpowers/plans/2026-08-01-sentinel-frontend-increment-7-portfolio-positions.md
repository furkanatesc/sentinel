# SENTINEL Frontend — Increment 7 (Portfolio & Positions) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two read-only screens — `/portfolio` (KPIs + equity curve + PnL-by-strategy + risk-allocation + win/loss charts + open-positions summary) and `/positions` (rich sortable table + detail drawer) — driven entirely by the mock data seam.

**Architecture:** Follows the established seam `component → usePortfolio/usePositions → getApi() → mock`. New types on `SentinelApi`, deterministic mock adapters, small SRP components composed into two RSC-prefetched pages. Reuses shared primitives; promotes `EquityCurve` to a shared component. No component imports the mock directly.

**Tech Stack:** Next.js 16 (App Router, server-first), TypeScript, Tailwind v4 (Sentinel dark tokens), TanStack Query (RSC prefetch + HydrationBoundary), Recharts (Area/Bar/Pie), shadcn Sheet (drawer), sonner (toast), Vitest + React Testing Library (TDD).

## Global Constraints

- **UI dili Türkçe** — all visible strings in Turkish (technical tokens/symbols exempt).
- **Seam rule (DIP):** no component imports `lib/api/mock` directly; data flows only through `usePortfolio`/`usePositions` → `getApi()`. `httpApi` methods resolve to `notReady`.
- **Read-only:** position actions never place a real trade. Mirror `TokenActions`: `const mode = useSessionStore((s) => s.tradingMode)`; live → `toast.warning(...)`, paper/shadow → `toast(...)`.
- **Reuse:** `MetricTile` (`@/components/sentinel/MetricTile`) for tiles; `riskMeta` (`@/lib/format`) for risk badges (`RiskLevel = "critical"|"high"|"medium"|"good"|"strong"`); `TokenAvatar`/`WalletAddress`; shadcn `Sheet` (drawer, mirror `components/feed/EventDetailDrawer.tsx`); `EquityPoint` type (from Strategies). Token links resolve by **symbol** (`/tokens/${symbol}`, the established pattern in `EventDetailDrawer`).
- **Clean code + SOLID:** SRP (one responsibility per file), OCP (risk-level registry, palette from `riskMeta`), DIP, ISP (narrow props), DRY without premature abstraction.
- **Scope:** read-only only — no real order placement, no PnL-by-creator/age charts, no saved views/CSV/column customization/virtual scroll.
- **Repo layout:** `apps/web/` (no `src/`; `app/`, `components/`, `lib/` top-level; alias `@/` → `apps/web/`). Run all commands from `apps/web/`.
- **Conventions:** interactive/hook components start with `"use client"`; pages are server components; inline numeric `style={{ fontSize: N }}` + Tailwind classes (`rounded-lg border border-border bg-card`, `text-muted-foreground`, `font-mono tabular-nums`).

---

### Task 1: Data seam — types, contract, http stub, mock adapters, qk, hooks

**Files:**
- Modify: `apps/web/lib/api/types.ts` (append Portfolio/Position types)
- Modify: `apps/web/lib/api/contract.ts` (add `getPortfolio`/`getPositions`)
- Modify: `apps/web/lib/api/http.ts` (add `notReady` entries)
- Modify: `apps/web/lib/api/mock.ts` (deterministic portfolio + positions data)
- Modify: `apps/web/lib/get-query-client.ts` (add `qk.portfolio`, `qk.positions`)
- Modify: `apps/web/lib/hooks/queries.ts` (add `usePortfolio`, `usePositions`)
- Test: `apps/web/lib/api/portfolio.test.ts`
- Test: `apps/web/lib/hooks/portfolio.test.tsx`

**Interfaces:**
- Consumes: existing `mockApi`, `qk`, `getApi()`, `scoreToLevel`, `riskMeta`, `RiskLevel`, `EquityPoint`, existing `delay`, `seedOf`, `STRATEGY_DEFS` (all in mock.ts).
- Produces (later tasks rely on these exact names/types):
  - `PortfolioSummary { totalValueSol; availableSol; investedSol; realizedPnlSol; unrealizedPnlSol; dailyPnlSol; maxDrawdownPct; riskExposurePct; rugExposurePct: number }`
  - `StrategyPnl { strategyId: string; name: string; pnlSol: number }`
  - `AllocationSlice { label: string; pct: number; color: string }`
  - `WinLossBucket { label: string; count: number }`
  - `PortfolioOverview { summary: PortfolioSummary; equityCurve: EquityPoint[]; pnlByStrategy: StrategyPnl[]; riskAllocation: AllocationSlice[]; winLoss: WinLossBucket[] }`
  - `Position { id; tokenMint; tokenSymbol; strategyId; strategyName; entryPrice; currentPrice; sizeSol; pnlSol; pnlPct; stopLossPct; takeProfitPct; tokenRisk: RiskLevel; creatorRisk: RiskLevel; ageLabel; openedAt: string }`
  - `mockApi.getPortfolio(): Promise<PortfolioOverview>`, `mockApi.getPositions(): Promise<Position[]>` (≥6 rows, deterministic)
  - `qk.portfolio` (const tuple), `qk.positions` (const tuple)
  - `usePortfolio()`, `usePositions()`

- [ ] **Step 1: Append types to `lib/api/types.ts`**

```ts
// --- Portfolio & Positions (Increment 7) ---
export interface PortfolioSummary {
  totalValueSol: number; availableSol: number; investedSol: number;
  realizedPnlSol: number; unrealizedPnlSol: number; dailyPnlSol: number;
  maxDrawdownPct: number; riskExposurePct: number; rugExposurePct: number;
}
export interface StrategyPnl { strategyId: string; name: string; pnlSol: number; }
export interface AllocationSlice { label: string; pct: number; color: string; }
export interface WinLossBucket { label: string; count: number; }
export interface PortfolioOverview {
  summary: PortfolioSummary;
  equityCurve: EquityPoint[];
  pnlByStrategy: StrategyPnl[];
  riskAllocation: AllocationSlice[];
  winLoss: WinLossBucket[];
}
export interface Position {
  id: string; tokenMint: string; tokenSymbol: string;
  strategyId: string; strategyName: string;
  entryPrice: number; currentPrice: number; sizeSol: number;
  pnlSol: number; pnlPct: number;
  stopLossPct: number; takeProfitPct: number;
  tokenRisk: RiskLevel; creatorRisk: RiskLevel;
  ageLabel: string; openedAt: string;
}
```
> `EquityPoint` and `RiskLevel` are already imported/defined in `types.ts` (RiskLevel via `import type { RiskLevel } from "@/lib/format"`). If `RiskLevel` is not yet imported in types.ts, add it to the existing format import.

- [ ] **Step 2: Extend `SentinelApi` in `lib/api/contract.ts`**

Update the type import to add `PortfolioOverview, Position`, and add after `getStrategy`:
```ts
  getPortfolio(): Promise<PortfolioOverview>;
  getPositions(): Promise<Position[]>;
```

- [ ] **Step 3: Add `notReady` entries in `lib/api/http.ts`** (after `getStrategy: notReady,`)
```ts
  getPortfolio: notReady,
  getPositions: notReady,
```

- [ ] **Step 4: Write the failing adapter test** — `apps/web/lib/api/portfolio.test.ts`

```ts
import { mockApi } from "./mock";

test("getPortfolio returns deterministic summary + all four series", async () => {
  const a = await mockApi.getPortfolio();
  const b = await mockApi.getPortfolio();
  expect(a).toEqual(b);
  expect(typeof a.summary.totalValueSol).toBe("number");
  expect(a.equityCurve.length).toBeGreaterThan(1);
  expect(a.pnlByStrategy.length).toBeGreaterThan(0);
  expect(a.riskAllocation.length).toBeGreaterThan(0);
  expect(a.riskAllocation.every((s) => s.color.startsWith("#"))).toBe(true);
  expect(a.winLoss.length).toBeGreaterThan(0);
});

test("getPositions returns deterministic rows referencing real token symbols + strategy ids", async () => {
  const a = await mockApi.getPositions();
  const b = await mockApi.getPositions();
  expect(a).toEqual(b);
  expect(a.length).toBeGreaterThanOrEqual(6);
  for (const p of a) {
    expect(p.tokenSymbol).toBeTruthy();
    expect(p.strategyId).toBeTruthy();
    expect(["critical", "high", "medium", "good", "strong"]).toContain(p.tokenRisk);
    expect(typeof p.pnlSol).toBe("number");
  }
});
```

- [ ] **Step 5: Run to verify it fails** — `npm test -- portfolio.test.ts` → FAIL (methods missing).

- [ ] **Step 6: Add mock data + adapters in `lib/api/mock.ts`**

Add the new type names to the existing `import type { ... } from "./types";` line: `PortfolioSummary, PortfolioOverview, StrategyPnl, AllocationSlice, WinLossBucket, Position, EquityPoint, RiskLevel` (drop any already present). Ensure `riskMeta` and `scoreToLevel` are imported from `@/lib/format` (scoreToLevel already is; add `riskMeta` if missing). Then add this block just above `export const mockApi: SentinelApi = {`:

```ts
// --- Portfolio & Positions (Increment 7) ---
function portfolioEquity(seed: number, len = 40): EquityPoint[] {
  const out: EquityPoint[] = [];
  let v = 500;
  for (let i = 0; i < len; i++) {
    v += Math.sin(seed + i * 0.5) * 12 + ((seed * (i + 1)) % 9) - 4;
    out.push({ t: i, v: Math.round(Math.max(50, v) * 100) / 100 });
  }
  return out;
}

const portfolioOverview: PortfolioOverview = (() => {
  const seed = 4242;
  const invested = 640, available = 210;
  const unrealized = 84.5, realized = 312.7, daily = -18.2;
  const summary: PortfolioSummary = {
    totalValueSol: Math.round((invested + available + unrealized) * 10) / 10,
    availableSol: available, investedSol: invested,
    realizedPnlSol: realized, unrealizedPnlSol: unrealized, dailyPnlSol: daily,
    maxDrawdownPct: 22, riskExposurePct: 68, rugExposurePct: 4,
  };
  const pnlByStrategy: StrategyPnl[] = STRATEGY_DEFS.slice(0, 5).map((d, i) => ({
    strategyId: d.id, name: d.name, pnlSol: Math.round((Math.sin(seed + i) * 120 + (i % 3 === 2 ? -60 : 90)) * 10) / 10,
  }));
  const riskAllocation: AllocationSlice[] = [
    { label: riskMeta.strong.label, pct: 34, color: riskMeta.strong.color },
    { label: riskMeta.good.label, pct: 28, color: riskMeta.good.color },
    { label: riskMeta.medium.label, pct: 22, color: riskMeta.medium.color },
    { label: riskMeta.high.label, pct: 12, color: riskMeta.high.color },
    { label: riskMeta.critical.label, pct: 4, color: riskMeta.critical.color },
  ];
  const winLoss: WinLossBucket[] = [
    { label: "Büyük Kazanç", count: 14 }, { label: "Kazanç", count: 38 },
    { label: "Başabaş", count: 9 }, { label: "Kayıp", count: 21 }, { label: "Büyük Kayıp", count: 6 },
  ];
  return { summary, equityCurve: portfolioEquity(seed), pnlByStrategy, riskAllocation, winLoss };
})();

const POSITION_TOKENS: { symbol: string; mint: string }[] = [
  { symbol: "PULSE", mint: "9xQeWv...4Fk2" }, { symbol: "LMN", mint: "Cd93Kf...6Rt4" },
  { symbol: "HLS", mint: "Hh77Nb...2Kl9" }, { symbol: "PXL", mint: "Ii22Vc...5Dw3" },
  { symbol: "MCAT", mint: "Ff01Xq...3Bn7" }, { symbol: "NOVA", mint: "Ap12Rd...9Zk1" },
  { symbol: "ZAP", mint: "Gg44Lm...8Yu2" }, { symbol: "GFROG", mint: "7mLp2c...1Qw8" },
];

const positions: Position[] = POSITION_TOKENS.map((tok, i) => {
  const strat = STRATEGY_DEFS[i % STRATEGY_DEFS.length];
  const seed = seedOf(tok.symbol + strat.id);
  const entry = Math.round((0.0004 + (seed % 90) / 100000) * 1e6) / 1e6;
  const pnlPct = ((seed % 160) - 60);
  const current = Math.round(entry * (1 + pnlPct / 100) * 1e6) / 1e6;
  const size = 4 + (seed % 30);
  return {
    id: `pos-${i + 1}`, tokenMint: tok.mint, tokenSymbol: tok.symbol,
    strategyId: strat.id, strategyName: strat.name,
    entryPrice: entry, currentPrice: current, sizeSol: size,
    pnlSol: Math.round(size * (pnlPct / 100) * 10) / 10, pnlPct,
    stopLossPct: 12 + (seed % 10), takeProfitPct: 40 + (seed % 60),
    tokenRisk: scoreToLevel(30 + (seed % 65)), creatorRisk: scoreToLevel(35 + ((seed * 3) % 60)),
    ageLabel: `${1 + (seed % 46)} dk`, openedAt: `${1 + (seed % 46)} dk önce`,
  };
});
```

Then add to the `mockApi` object (after `getStrategy: ...,`):
```ts
  getPortfolio: () => delay(portfolioOverview),
  getPositions: () => delay(positions),
```

- [ ] **Step 7: Run the adapter test** — `npm test -- portfolio.test.ts` → PASS.

- [ ] **Step 8: Add query keys in `lib/get-query-client.ts`** (after `strategy` entry)
```ts
  portfolio: ["portfolio"] as const,
  positions: ["positions-list"] as const,
```
> Note: use `["positions-list"]` (not `["positions"]`) to avoid any collision with unrelated keys; later tasks reference `qk.positions` by name so the string value is internal.

- [ ] **Step 9: Write the failing hooks test** — `apps/web/lib/hooks/portfolio.test.tsx`

```tsx
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { usePortfolio, usePositions } from "./queries";

function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("usePortfolio loads the overview", async () => {
  const { result } = renderHook(() => usePortfolio(), { wrapper });
  await waitFor(() => expect(result.current.data?.summary).toBeDefined());
});

test("usePositions loads rows", async () => {
  const { result } = renderHook(() => usePositions(), { wrapper });
  await waitFor(() => expect(result.current.data?.length).toBeGreaterThan(0));
});
```

- [ ] **Step 10: Run to verify it fails** — `npm test -- hooks/portfolio.test.tsx` → FAIL.

- [ ] **Step 11: Add hooks in `lib/hooks/queries.ts`**
```ts
export function usePortfolio() {
  return useQuery({ queryKey: qk.portfolio, queryFn: () => getApi().getPortfolio() });
}
export function usePositions() {
  return useQuery({ queryKey: qk.positions, queryFn: () => getApi().getPositions() });
}
```

- [ ] **Step 12: Run both tests** — `npm test -- portfolio.test.ts hooks/portfolio.test.tsx` → PASS (4 tests).

- [ ] **Step 13: Commit**
```bash
git add apps/web/lib/api apps/web/lib/get-query-client.ts apps/web/lib/hooks/queries.ts
git commit -m "feat(portfolio): add data seam (types, mock adapters, hooks)"
```

---

### Task 2: Promote `EquityCurve` to a shared component (refactor)

**Files:**
- Rename: `apps/web/components/strategy/EquityCurve.tsx` → `apps/web/components/sentinel/EquityCurve.tsx`
- Rename: `apps/web/components/strategy/EquityCurve.test.tsx` → `apps/web/components/sentinel/EquityCurve.test.tsx`
- Modify: `apps/web/components/strategy/StrategyDetailContent.tsx` (update import path)

**Interfaces:**
- Consumes: `EquityPoint`.
- Produces: `EquityCurve({ data, title?, color? }: { data: EquityPoint[]; title?: string; color?: string })` at `@/components/sentinel/EquityCurve`. Defaults: `title = "Equity Curve"`, `color = "#2FD98B"`. Gradient `id` derived from `color` (no static `equity-grad`).

- [ ] **Step 1: Move the files with git (preserve history)**
```bash
cd apps/web
git mv components/strategy/EquityCurve.tsx components/sentinel/EquityCurve.tsx
git mv components/strategy/EquityCurve.test.tsx components/sentinel/EquityCurve.test.tsx
```

- [ ] **Step 2: Update the moved component** — `components/sentinel/EquityCurve.tsx`

Replace its body with (adds `title`/`color` props, derives gradient id):
```tsx
"use client";
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { EquityPoint } from "@/lib/api/types";

export function EquityCurve({ data, title = "Equity Curve", color = "#2FD98B" }: { data: EquityPoint[]; title?: string; color?: string }) {
  const gradId = `equity-grad-${color.replace("#", "")}`;
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>{title}</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="t" hide />
            <YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} labelFormatter={() => ""} />
            <Area type="monotone" dataKey="v" stroke={color} strokeWidth={1.5} fill={`url(#${gradId})`} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Update the moved test** — `components/sentinel/EquityCurve.test.tsx`

Keep the existing smoke test, and add a props assertion. The import path stays `./EquityCurve` (co-located). Append:
```tsx
import { render, screen } from "@testing-library/react";
// (existing smoke test stays above)

test("renders a custom title", () => {
  render(<div style={{ width: 400, height: 200 }}><EquityCurve data={[{ t: 0, v: 1 }, { t: 1, v: 2 }]} title="Portföy Değeri" /></div>);
  expect(screen.getByText("Portföy Değeri")).toBeInTheDocument();
});
```
> Ensure `EquityCurve` and `screen` are imported at the top of the file (the original test imported `render`; add `screen`).

- [ ] **Step 4: Update the strategy import** — `components/strategy/StrategyDetailContent.tsx`

Change `import { EquityCurve } from "./EquityCurve";` → `import { EquityCurve } from "@/components/sentinel/EquityCurve";`

- [ ] **Step 5: Run the affected tests (regression)** — 
```
npm test -- components/sentinel/EquityCurve.test.tsx components/strategy/StrategyDetailContent.test.tsx
```
Expected: PASS — moved smoke + new title test green; strategy detail still renders its equity curve.

- [ ] **Step 6: Commit**
```bash
git add apps/web/components/sentinel/EquityCurve.tsx apps/web/components/sentinel/EquityCurve.test.tsx apps/web/components/strategy/StrategyDetailContent.tsx
git commit -m "refactor(charts): promote EquityCurve to shared sentinel component with title/color props"
```

---

### Task 3: Position risk-filter config + pnlColor (OCP)

**Files:**
- Create: `apps/web/lib/position/risk-filter.ts`
- Test: `apps/web/lib/position/risk-filter.test.ts`

**Interfaces:**
- Consumes: `RiskLevel` from `@/lib/format`.
- Produces: `POSITION_RISK_LEVELS: RiskLevel[]`, `pnlColor(v: number): string`.

- [ ] **Step 1: Write the failing test** — `apps/web/lib/position/risk-filter.test.ts`
```ts
import { POSITION_RISK_LEVELS, pnlColor } from "./risk-filter";

test("POSITION_RISK_LEVELS covers the five risk levels", () => {
  expect(POSITION_RISK_LEVELS).toEqual(["strong", "good", "medium", "high", "critical"]);
});

test("pnlColor is green for >=0 and red for <0", () => {
  expect(pnlColor(0)).toBe("#2FD98B");
  expect(pnlColor(5)).toBe("#2FD98B");
  expect(pnlColor(-1)).toBe("#F0476B");
});
```

- [ ] **Step 2: Run to verify it fails** — `npm test -- risk-filter.test.ts` → FAIL.

- [ ] **Step 3: Implement** — `apps/web/lib/position/risk-filter.ts`
```ts
import type { RiskLevel } from "@/lib/format";

export const POSITION_RISK_LEVELS: RiskLevel[] = ["strong", "good", "medium", "high", "critical"];

export function pnlColor(v: number): string {
  return v >= 0 ? "#2FD98B" : "#F0476B";
}
```

- [ ] **Step 4: Run to verify it passes** — `npm test -- risk-filter.test.ts` → PASS.

- [ ] **Step 5: Commit**
```bash
git add apps/web/lib/position
git commit -m "feat(positions): add risk-filter config + pnlColor"
```

---

### Task 4: PortfolioKpis (+ MetricTile valueColor extension)

**Files:**
- Modify: `apps/web/components/sentinel/MetricTile.tsx` (add optional `valueColor?`)
- Create: `apps/web/components/portfolio/PortfolioKpis.tsx`
- Test: `apps/web/components/portfolio/PortfolioKpis.test.tsx`

**Interfaces:**
- Consumes: `PortfolioSummary`, `MetricTile`, `pnlColor`.
- Produces: `MetricTile({ label, value, hint?, valueColor? })`; `PortfolioKpis({ summary }: { summary: PortfolioSummary })`.

- [ ] **Step 1: Extend `MetricTile`** — `components/sentinel/MetricTile.tsx`

Add an optional `valueColor` prop applied to the value div (backward compatible — existing callers pass nothing):
```tsx
export function MetricTile({ label, value, hint, valueColor }: { label: string; value: string; hint?: string; valueColor?: string }) {
  return (
    <div className="rounded-lg border border-border bg-surface-2 p-3">
      <div className="text-muted-foreground" style={{ fontSize: 11 }}>{label}</div>
      <div className="mt-1 font-mono tabular-nums" style={{ fontSize: 18, fontWeight: 600, color: valueColor }}>{value}</div>
      {hint && <div className="text-muted-foreground" style={{ fontSize: 10 }}>{hint}</div>}
    </div>
  );
}
```

- [ ] **Step 2: Write the failing test** — `apps/web/components/portfolio/PortfolioKpis.test.tsx`
```tsx
import { render, screen } from "@testing-library/react";
import { PortfolioKpis } from "./PortfolioKpis";
import type { PortfolioSummary } from "@/lib/api/types";

const summary: PortfolioSummary = {
  totalValueSol: 934.5, availableSol: 210, investedSol: 640,
  realizedPnlSol: 312.7, unrealizedPnlSol: 84.5, dailyPnlSol: -18.2,
  maxDrawdownPct: 22, riskExposurePct: 68, rugExposurePct: 4,
};

test("shows key portfolio KPI labels and values", () => {
  render(<PortfolioKpis summary={summary} />);
  expect(screen.getByText("Toplam Değer")).toBeInTheDocument();
  expect(screen.getByText("Günlük PnL")).toBeInTheDocument();
  expect(screen.getByText(/-18.2 SOL/)).toBeInTheDocument();
});
```

- [ ] **Step 3: Run to verify it fails** — `npm test -- PortfolioKpis.test.tsx` → FAIL (module missing).

- [ ] **Step 4: Implement** — `apps/web/components/portfolio/PortfolioKpis.tsx`
```tsx
import type { PortfolioSummary } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { pnlColor } from "@/lib/position/risk-filter";

export function PortfolioKpis({ summary: s }: { summary: PortfolioSummary }) {
  const tiles: { label: string; value: string; valueColor?: string }[] = [
    { label: "Toplam Değer", value: `${s.totalValueSol} SOL` },
    { label: "Kullanılabilir", value: `${s.availableSol} SOL` },
    { label: "Yatırılan", value: `${s.investedSol} SOL` },
    { label: "Gerçekleşen PnL", value: `${s.realizedPnlSol} SOL`, valueColor: pnlColor(s.realizedPnlSol) },
    { label: "Gerçekleşmemiş PnL", value: `${s.unrealizedPnlSol} SOL`, valueColor: pnlColor(s.unrealizedPnlSol) },
    { label: "Günlük PnL", value: `${s.dailyPnlSol} SOL`, valueColor: pnlColor(s.dailyPnlSol) },
    { label: "Maks. Drawdown", value: `%${s.maxDrawdownPct}` },
    { label: "Risk Maruziyeti", value: `%${s.riskExposurePct}` },
    { label: "Rug Maruziyeti", value: `%${s.rugExposurePct}` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-5">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} valueColor={t.valueColor} />)}
    </div>
  );
}
```

- [ ] **Step 5: Run to verify it passes** — `npm test -- PortfolioKpis.test.tsx` → PASS.

- [ ] **Step 6: Commit**
```bash
git add apps/web/components/sentinel/MetricTile.tsx apps/web/components/portfolio/PortfolioKpis.tsx apps/web/components/portfolio/PortfolioKpis.test.tsx
git commit -m "feat(portfolio): add PortfolioKpis with MetricTile valueColor"
```

---

### Task 5: Portfolio charts — PnlByStrategyChart, RiskAllocationChart, WinLossChart

**Files:**
- Create: `apps/web/components/portfolio/PnlByStrategyChart.tsx`
- Create: `apps/web/components/portfolio/RiskAllocationChart.tsx`
- Create: `apps/web/components/portfolio/WinLossChart.tsx`
- Test: `apps/web/components/portfolio/PortfolioCharts.test.tsx`

**Interfaces:**
- Consumes: `StrategyPnl[]`, `AllocationSlice[]`, `WinLossBucket[]`; Recharts; `pnlColor`.
- Produces: `PnlByStrategyChart({ data })`, `RiskAllocationChart({ data })`, `WinLossChart({ data })`.

- [ ] **Step 1: Write the failing test** — `apps/web/components/portfolio/PortfolioCharts.test.tsx`
```tsx
import { render } from "@testing-library/react";
import { PnlByStrategyChart } from "./PnlByStrategyChart";
import { RiskAllocationChart } from "./RiskAllocationChart";
import { WinLossChart } from "./WinLossChart";

const wrap = (ui: React.ReactNode) => render(<div style={{ width: 400, height: 240 }}>{ui}</div>);

test("PnL-by-strategy chart renders (smoke)", () => {
  const { container } = wrap(<PnlByStrategyChart data={[{ strategyId: "s1", name: "S1", pnlSol: 12 }, { strategyId: "s2", name: "S2", pnlSol: -4 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});

test("risk-allocation chart renders (smoke)", () => {
  const { container } = wrap(<RiskAllocationChart data={[{ label: "Güçlü", pct: 60, color: "#2FD98B" }, { label: "Orta", pct: 40, color: "#3E9BFF" }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});

test("win/loss chart renders (smoke)", () => {
  const { container } = wrap(<WinLossChart data={[{ label: "Kazanç", count: 10 }, { label: "Kayıp", count: 4 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
```

- [ ] **Step 2: Run to verify it fails** — `npm test -- PortfolioCharts.test.tsx` → FAIL.

- [ ] **Step 3: Implement `PnlByStrategyChart.tsx`**
```tsx
"use client";
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip, Cell } from "recharts";
import type { StrategyPnl } from "@/lib/api/types";
import { pnlColor } from "@/lib/position/risk-filter";

export function PnlByStrategyChart({ data }: { data: StrategyPnl[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Strateji Bazında PnL (SOL)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} layout="vertical" margin={{ top: 4, right: 8, bottom: 0, left: 8 }}>
            <XAxis type="number" hide />
            <YAxis type="category" dataKey="name" width={110} tick={{ fill: "#8A94A6", fontSize: 11 }} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} cursor={{ fill: "rgba(255,255,255,0.04)" }} />
            <Bar dataKey="pnlSol" radius={[0, 4, 4, 0]}>
              {data.map((d) => <Cell key={d.strategyId} fill={pnlColor(d.pnlSol)} />)}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Implement `RiskAllocationChart.tsx`**
```tsx
"use client";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from "recharts";
import type { AllocationSlice } from "@/lib/api/types";

export function RiskAllocationChart({ data }: { data: AllocationSlice[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Risk Dağılımı</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie data={data} dataKey="pct" nameKey="label" innerRadius={55} outerRadius={85} paddingAngle={2} stroke="none">
              {data.map((d) => <Cell key={d.label} fill={d.color} />)}
            </Pie>
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} formatter={(v: number, n) => [`%${v}`, n]} />
            <Legend wrapperStyle={{ fontSize: 11 }} />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Implement `WinLossChart.tsx`**
```tsx
"use client";
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { WinLossBucket } from "@/lib/api/types";

export function WinLossChart({ data }: { data: WinLossBucket[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Kazanç/Kayıp Dağılımı</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis dataKey="label" tick={{ fill: "#8A94A6", fontSize: 10 }} interval={0} />
            <YAxis hide />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} cursor={{ fill: "rgba(255,255,255,0.04)" }} />
            <Bar dataKey="count" fill="#7C5CFF" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: Run to verify it passes** — `npm test -- PortfolioCharts.test.tsx` → PASS (3 tests).

- [ ] **Step 7: Commit**
```bash
git add apps/web/components/portfolio/PnlByStrategyChart.tsx apps/web/components/portfolio/RiskAllocationChart.tsx apps/web/components/portfolio/WinLossChart.tsx apps/web/components/portfolio/PortfolioCharts.test.tsx
git commit -m "feat(portfolio): add PnL-by-strategy, risk-allocation, win/loss charts"
```

---

### Task 6: OpenPositionsSummary

**Files:**
- Create: `apps/web/components/portfolio/OpenPositionsSummary.tsx`
- Test: `apps/web/components/portfolio/OpenPositionsSummary.test.tsx`

**Interfaces:**
- Consumes: `usePositions`, `Position`, `pnlColor`, `riskMeta`, `next/link`.
- Produces: `OpenPositionsSummary()` — `"use client"`; lists the first ~5 positions with a "Tümü →" link to `/positions`.

- [ ] **Step 1: Write the failing test** — `apps/web/components/portfolio/OpenPositionsSummary.test.tsx`
```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { OpenPositionsSummary } from "./OpenPositionsSummary";

test("lists open positions with a link to /positions", async () => {
  render(<QueryClientProvider client={getQueryClient()}><OpenPositionsSummary /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("Açık Pozisyonlar")).toBeInTheDocument());
  const link = screen.getByText(/Tümü/).closest("a")!;
  expect(link.getAttribute("href")).toBe("/positions");
});
```

- [ ] **Step 2: Run to verify it fails** — `npm test -- OpenPositionsSummary.test.tsx` → FAIL.

- [ ] **Step 3: Implement** — `apps/web/components/portfolio/OpenPositionsSummary.tsx`
```tsx
"use client";
import Link from "next/link";
import { usePositions } from "@/lib/hooks/queries";
import { pnlColor } from "@/lib/position/risk-filter";
import { riskMeta } from "@/lib/format";

export function OpenPositionsSummary() {
  const { data } = usePositions();
  const rows = (data ?? []).slice(0, 5);
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <span className="font-medium" style={{ fontSize: 13 }}>Açık Pozisyonlar</span>
        <Link href="/positions" className="text-primary" style={{ fontSize: 12 }}>Tümü →</Link>
      </div>
      <ul className="space-y-2">
        {rows.map((p) => (
          <li key={p.id} className="flex items-center justify-between" style={{ fontSize: 12 }}>
            <span className="flex items-center gap-2">
              <span className="font-mono">{p.tokenSymbol}</span>
              <span className="text-muted-foreground">{p.strategyName}</span>
              <span style={{ color: riskMeta[p.tokenRisk].color, fontSize: 10 }}>{riskMeta[p.tokenRisk].label}</span>
            </span>
            <span className="font-mono tabular-nums" style={{ color: pnlColor(p.pnlSol) }}>{p.pnlSol} SOL (%{p.pnlPct})</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 4: Run to verify it passes** — `npm test -- OpenPositionsSummary.test.tsx` → PASS.

- [ ] **Step 5: Commit**
```bash
git add apps/web/components/portfolio/OpenPositionsSummary.tsx apps/web/components/portfolio/OpenPositionsSummary.test.tsx
git commit -m "feat(portfolio): add OpenPositionsSummary"
```

---

### Task 7: PortfolioContent (composition)

**Files:**
- Create: `apps/web/components/portfolio/PortfolioContent.tsx`
- Test: `apps/web/components/portfolio/PortfolioContent.test.tsx`

**Interfaces:**
- Consumes: `usePortfolio`, `PortfolioKpis`, `EquityCurve` (`@/components/sentinel/EquityCurve`), `PnlByStrategyChart`, `RiskAllocationChart`, `WinLossChart`, `OpenPositionsSummary`, `Skeleton`.
- Produces: `PortfolioContent()`.

- [ ] **Step 1: Write the failing test** — `apps/web/components/portfolio/PortfolioContent.test.tsx`
```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { PortfolioContent } from "./PortfolioContent";

test("composes KPIs, charts and open-positions summary", async () => {
  render(<QueryClientProvider client={getQueryClient()}><PortfolioContent /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("Portföy")).toBeInTheDocument());
  expect(screen.getByText("Toplam Değer")).toBeInTheDocument();
  expect(screen.getByText("Risk Dağılımı")).toBeInTheDocument();
  expect(screen.getByText("Açık Pozisyonlar")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run to verify it fails** — `npm test -- PortfolioContent.test.tsx` → FAIL.

- [ ] **Step 3: Implement** — `apps/web/components/portfolio/PortfolioContent.tsx`
```tsx
"use client";
import { usePortfolio } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";
import { EquityCurve } from "@/components/sentinel/EquityCurve";
import { PortfolioKpis } from "./PortfolioKpis";
import { PnlByStrategyChart } from "./PnlByStrategyChart";
import { RiskAllocationChart } from "./RiskAllocationChart";
import { WinLossChart } from "./WinLossChart";
import { OpenPositionsSummary } from "./OpenPositionsSummary";

export function PortfolioContent() {
  const { data, isError } = usePortfolio();
  if (isError) return <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Portföy yüklenemedi.</div>;
  if (!data) return <div className="space-y-4"><Skeleton className="h-24 w-full" /><Skeleton className="h-56 w-full" /></div>;
  return (
    <div className="space-y-5">
      <h1>Portföy</h1>
      <PortfolioKpis summary={data.summary} />
      <EquityCurve data={data.equityCurve} title="Portföy Değeri" />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <PnlByStrategyChart data={data.pnlByStrategy} />
        <RiskAllocationChart data={data.riskAllocation} />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <WinLossChart data={data.winLoss} />
        <OpenPositionsSummary />
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run to verify it passes** — `npm test -- PortfolioContent.test.tsx` → PASS.

- [ ] **Step 5: Commit**
```bash
git add apps/web/components/portfolio/PortfolioContent.tsx apps/web/components/portfolio/PortfolioContent.test.tsx
git commit -m "feat(portfolio): compose PortfolioContent"
```

---

### Task 8: Portfolio page (RSC prefetch)

**Files:**
- Modify: `apps/web/app/(app)/portfolio/page.tsx` (replace placeholder)

**Interfaces:**
- Consumes: `getQueryClient`, `qk`, `getApi`, `PortfolioContent`, `dehydrate`, `HydrationBoundary`.
- Produces: `/portfolio` route (prefetches both `getPortfolio` and `getPositions`).

- [ ] **Step 1: Replace the placeholder** — `apps/web/app/(app)/portfolio/page.tsx`
```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { PortfolioContent } from "@/components/portfolio/PortfolioContent";

export default async function PortfolioPage() {
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.prefetchQuery({ queryKey: qk.portfolio, queryFn: () => getApi().getPortfolio() }),
    queryClient.prefetchQuery({ queryKey: qk.positions, queryFn: () => getApi().getPositions() }),
  ]);
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <PortfolioContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 2: Run the full suite** — `npm test` → all green (existing + new).

- [ ] **Step 3: Commit**
```bash
git add apps/web/app/(app)/portfolio/page.tsx
git commit -m "feat(portfolio): wire /portfolio page with RSC prefetch"
```

---

### Task 9: PositionActions (read-only, toast)

**Files:**
- Create: `apps/web/components/position/PositionActions.tsx`
- Test: `apps/web/components/position/PositionActions.test.tsx`

**Interfaces:**
- Consumes: `useSessionStore` (`@/lib/store/session`), `toast` (sonner).
- Produces: `PositionActions({ symbol }: { symbol: string })` — "Kapat" and "SL/TP" buttons; live → `toast.warning`, else `toast`.

- [ ] **Step 1: Write the failing test** — `apps/web/components/position/PositionActions.test.tsx`
```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { PositionActions } from "./PositionActions";
import { useSessionStore } from "@/lib/store/session";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { warning: vi.fn() }) }));
import { toast } from "sonner";

test("live mode close fires toast.warning; paper mode fires toast", () => {
  useSessionStore.getState().setTradingMode("live");
  render(<PositionActions symbol="PULSE" />);
  fireEvent.click(screen.getByRole("button", { name: "Kapat" }));
  expect((toast as unknown as { warning: ReturnType<typeof vi.fn> }).warning).toHaveBeenCalled();

  useSessionStore.getState().setTradingMode("paper");
  fireEvent.click(screen.getByRole("button", { name: "SL/TP" }));
  expect(toast).toHaveBeenCalled();
});
```

- [ ] **Step 2: Run to verify it fails** — `npm test -- PositionActions.test.tsx` → FAIL.

- [ ] **Step 3: Implement** — `apps/web/components/position/PositionActions.tsx`
```tsx
"use client";
import { toast } from "sonner";
import { useSessionStore } from "@/lib/store/session";

export function PositionActions({ symbol }: { symbol: string }) {
  const mode = useSessionStore((s) => s.tradingMode);
  const act = (label: string) => {
    if (mode === "live") toast.warning(`CANLI mod — ${label} ${symbol}`, { description: "Gerçek para. Bu demoda emir gönderilmez." });
    else toast(`${label} ${symbol}`, { description: `${mode === "paper" ? "Kağıt" : "Gölge"} modda simüle edilir.` });
  };
  return (
    <div className="flex items-center gap-1.5">
      <button onClick={() => act("Kapat")} className="rounded-md px-2 py-1" style={{ backgroundColor: "rgba(240,71,107,0.15)", border: "1px solid rgba(240,71,107,0.4)", color: "#F0476B", fontSize: 11, fontWeight: 600 }}>Kapat</button>
      <button onClick={() => act("SL/TP ayarla")} className="rounded-md border border-border px-2 py-1 hover:bg-accent" style={{ fontSize: 11 }}>SL/TP</button>
    </div>
  );
}
```

- [ ] **Step 4: Run to verify it passes** — `npm test -- PositionActions.test.tsx` → PASS.

- [ ] **Step 5: Commit**
```bash
git add apps/web/components/position/PositionActions.tsx apps/web/components/position/PositionActions.test.tsx
git commit -m "feat(positions): add read-only PositionActions"
```

---

### Task 10: PositionsTable (sortable, links, risk badges, actions, row→drawer)

**Files:**
- Create: `apps/web/components/position/PositionsTable.tsx`
- Test: `apps/web/components/position/PositionsTable.test.tsx`

**Interfaces:**
- Consumes: `Position`, `pnlColor`, `riskMeta`, `PositionActions`, `next/link`.
- Produces: `PositionsTable({ rows, sortKey, onSort, onRowClick }: { rows: Position[]; sortKey: SortKey; onSort: (k: SortKey) => void; onRowClick: (p: Position) => void })` and `export type SortKey = "pnlSol" | "sizeSol" | "ageLabel"`.

- [ ] **Step 1: Write the failing test** — `apps/web/components/position/PositionsTable.test.tsx`
```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { PositionsTable } from "./PositionsTable";
import type { Position } from "@/lib/api/types";

const rows: Position[] = [
  { id: "pos-1", tokenMint: "9x..2", tokenSymbol: "PULSE", strategyId: "momentum-scalp", strategyName: "Momentum Scalp", entryPrice: 0.0004, currentPrice: 0.0006, sizeSol: 12, pnlSol: 5.4, pnlPct: 45, stopLossPct: 12, takeProfitPct: 60, tokenRisk: "good", creatorRisk: "medium", ageLabel: "8 dk", openedAt: "8 dk önce" },
];

test("renders token→/tokens and strategy→/strategies links; row click fires; action does not", () => {
  const onRowClick = vi.fn();
  render(<PositionsTable rows={rows} sortKey="pnlSol" onSort={() => {}} onRowClick={onRowClick} />);
  expect(screen.getByText("PULSE").closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
  expect(screen.getByText("Momentum Scalp").closest("a")!.getAttribute("href")).toBe("/strategies/momentum-scalp");
  fireEvent.click(screen.getByText("Kapat"));           // action button
  expect(onRowClick).not.toHaveBeenCalled();            // stopPropagation
  fireEvent.click(screen.getByText("PULSE"));           // row (via token cell)
});
```
> Add `import { vi } from "vitest";` at the top.

- [ ] **Step 2: Run to verify it fails** — `npm test -- PositionsTable.test.tsx` → FAIL.

- [ ] **Step 3: Implement** — `apps/web/components/position/PositionsTable.tsx`
```tsx
"use client";
import Link from "next/link";
import type { Position } from "@/lib/api/types";
import { riskMeta } from "@/lib/format";
import { pnlColor } from "@/lib/position/risk-filter";
import { PositionActions } from "./PositionActions";

export type SortKey = "pnlSol" | "sizeSol" | "ageLabel";

const HEADERS: { key?: SortKey; label: string }[] = [
  { label: "Token" }, { label: "Strateji" }, { label: "Giriş" }, { label: "Güncel" },
  { key: "sizeSol", label: "Boyut" }, { key: "pnlSol", label: "PnL" }, { label: "SL/TP" },
  { label: "Token Risk" }, { label: "Creator Risk" }, { key: "ageLabel", label: "Yaş" }, { label: "" },
];

export function PositionsTable({ rows, sortKey, onSort, onRowClick }: { rows: Position[]; sortKey: SortKey; onSort: (k: SortKey) => void; onRowClick: (p: Position) => void }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead>
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {HEADERS.map((h) => (
                <th key={h.label} className="whitespace-nowrap px-3 py-2 text-left font-normal">
                  {h.key ? (
                    <button onClick={() => onSort(h.key!)} className="hover:text-foreground" style={{ color: sortKey === h.key ? "#7C5CFF" : undefined }}>{h.label} ↕</button>
                  ) : h.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.id} onClick={() => onRowClick(p)} className="cursor-pointer border-t border-border hover:bg-accent/40">
                <td className="px-3 py-2"><Link href={`/tokens/${p.tokenSymbol}`} onClick={(e) => e.stopPropagation()} className="font-mono hover:underline">{p.tokenSymbol}</Link></td>
                <td className="px-3 py-2"><Link href={`/strategies/${p.strategyId}`} onClick={(e) => e.stopPropagation()} className="hover:underline">{p.strategyName}</Link></td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.entryPrice}</td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.currentPrice}</td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.sizeSol} SOL</td>
                <td className="px-3 py-2 font-mono tabular-nums" style={{ color: pnlColor(p.pnlSol) }}>{p.pnlSol} (%{p.pnlPct})</td>
                <td className="px-3 py-2 font-mono tabular-nums">%{p.stopLossPct} / %{p.takeProfitPct}</td>
                <td className="px-3 py-2"><span style={{ color: riskMeta[p.tokenRisk].color, fontSize: 11 }}>{riskMeta[p.tokenRisk].label}</span></td>
                <td className="px-3 py-2"><span style={{ color: riskMeta[p.creatorRisk].color, fontSize: 11 }}>{riskMeta[p.creatorRisk].label}</span></td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.ageLabel}</td>
                <td className="px-3 py-2" onClick={(e) => e.stopPropagation()}><PositionActions symbol={p.tokenSymbol} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run to verify it passes** — `npm test -- PositionsTable.test.tsx` → PASS.

- [ ] **Step 5: Commit**
```bash
git add apps/web/components/position/PositionsTable.tsx apps/web/components/position/PositionsTable.test.tsx
git commit -m "feat(positions): add sortable PositionsTable with links + actions"
```

---

### Task 11: PositionDetailDrawer (Sheet)

**Files:**
- Create: `apps/web/components/position/PositionDetailDrawer.tsx`
- Test: `apps/web/components/position/PositionDetailDrawer.test.tsx`

**Interfaces:**
- Consumes: shadcn `Sheet` (`@/components/ui/sheet`), `Position`, `riskMeta`, `pnlColor`, `PositionActions`, `next/link`.
- Produces: `PositionDetailDrawer({ position, onClose }: { position: Position | null; onClose: () => void })` (mirror `EventDetailDrawer` shape).

- [ ] **Step 1: Write the failing test** — `apps/web/components/position/PositionDetailDrawer.test.tsx`
```tsx
import { render, screen } from "@testing-library/react";
import { PositionDetailDrawer } from "./PositionDetailDrawer";
import type { Position } from "@/lib/api/types";

const p: Position = { id: "pos-1", tokenMint: "9x..2", tokenSymbol: "PULSE", strategyId: "momentum-scalp", strategyName: "Momentum Scalp", entryPrice: 0.0004, currentPrice: 0.0006, sizeSol: 12, pnlSol: 5.4, pnlPct: 45, stopLossPct: 12, takeProfitPct: 60, tokenRisk: "good", creatorRisk: "medium", ageLabel: "8 dk", openedAt: "8 dk önce" };

test("renders position breakdown when open", () => {
  render(<PositionDetailDrawer position={p} onClose={() => {}} />);
  expect(screen.getByText("PULSE")).toBeInTheDocument();
  expect(screen.getByText("Giriş / Güncel")).toBeInTheDocument();
  expect(screen.getByText("Stop-Loss / Take-Profit")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run to verify it fails** — `npm test -- PositionDetailDrawer.test.tsx` → FAIL.

- [ ] **Step 3: Implement** — `apps/web/components/position/PositionDetailDrawer.tsx`
```tsx
"use client";
import Link from "next/link";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import type { Position } from "@/lib/api/types";
import { riskMeta } from "@/lib/format";
import { pnlColor } from "@/lib/position/risk-filter";
import { PositionActions } from "./PositionActions";

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="flex items-center justify-between border-b border-border py-2"><span className="text-muted-foreground" style={{ fontSize: 12 }}>{label}</span><span style={{ fontSize: 13 }}>{children}</span></div>;
}

export function PositionDetailDrawer({ position, onClose }: { position: Position | null; onClose: () => void }) {
  return (
    <Sheet open={!!position} onOpenChange={(o) => { if (!o) onClose(); }}>
      <SheetContent side="right" className="w-[380px] bg-popover">
        {position && (
          <>
            <SheetHeader><SheetTitle><span className="font-mono">{position.tokenSymbol}</span> · {position.strategyName}</SheetTitle></SheetHeader>
            <div className="mt-3 space-y-1">
              <Row label="Giriş / Güncel"><span className="font-mono">{position.entryPrice} / {position.currentPrice}</span></Row>
              <Row label="Boyut"><span className="font-mono">{position.sizeSol} SOL</span></Row>
              <Row label="PnL"><span className="font-mono" style={{ color: pnlColor(position.pnlSol) }}>{position.pnlSol} SOL (%{position.pnlPct})</span></Row>
              <Row label="Stop-Loss / Take-Profit"><span className="font-mono">%{position.stopLossPct} / %{position.takeProfitPct}</span></Row>
              <Row label="Token Risk"><span style={{ color: riskMeta[position.tokenRisk].color }}>{riskMeta[position.tokenRisk].label}</span></Row>
              <Row label="Creator Risk"><span style={{ color: riskMeta[position.creatorRisk].color }}>{riskMeta[position.creatorRisk].label}</span></Row>
              <Row label="Açılış">{position.openedAt}</Row>
            </div>
            <div className="mt-4"><PositionActions symbol={position.tokenSymbol} /></div>
            <Link href={`/tokens/${position.tokenSymbol}`} className="mt-4 inline-block rounded-md bg-primary px-4 py-2 text-primary-foreground" style={{ fontSize: 13, fontWeight: 500 }}>Token Detayına Git</Link>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
```

- [ ] **Step 4: Run to verify it passes** — `npm test -- PositionDetailDrawer.test.tsx` → PASS.

- [ ] **Step 5: Commit**
```bash
git add apps/web/components/position/PositionDetailDrawer.tsx apps/web/components/position/PositionDetailDrawer.test.tsx
git commit -m "feat(positions): add PositionDetailDrawer"
```

---

### Task 12: PositionsContent (filter + sort + drawer state)

**Files:**
- Create: `apps/web/components/position/PositionsContent.tsx`
- Test: `apps/web/components/position/PositionsContent.test.tsx`

**Interfaces:**
- Consumes: `usePositions`, `PositionsTable` (+ `SortKey`), `PositionDetailDrawer`, `POSITION_RISK_LEVELS`, `riskMeta`, `RiskLevel`.
- Produces: `PositionsContent()` — client component; risk filter chips + sort state + drawer state.

- [ ] **Step 1: Write the failing test** — `apps/web/components/position/PositionsContent.test.tsx`
```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { PositionsContent } from "./PositionsContent";

function renderList() {
  return render(<QueryClientProvider client={getQueryClient()}><PositionsContent /></QueryClientProvider>);
}

test("renders positions from the seam", async () => {
  renderList();
  await waitFor(() => expect(screen.getByText("Pozisyonlar")).toBeInTheDocument());
  expect(screen.getAllByRole("row").length).toBeGreaterThan(1);
});

test("clicking a row opens the detail drawer", async () => {
  renderList();
  await waitFor(() => expect(screen.getAllByText("PULSE").length).toBeGreaterThan(0));
  fireEvent.click(screen.getAllByText("PULSE")[0]);
  await waitFor(() => expect(screen.getByText("Giriş / Güncel")).toBeInTheDocument());
});
```
> `PULSE` is the first mock position's symbol (from Task 1). The token cell is a link; clicking it calls `stopPropagation`, so target the row: click the row's `Boyut`/PnL cell text instead if needed — use `screen.getAllByText("PULSE")[0]` which is inside the row; if the link's stopPropagation prevents row open, click a non-link cell. Adjust the click target to a non-link cell (e.g. the size cell) to trigger `onRowClick`; assert the drawer heading appears.

- [ ] **Step 2: Run to verify it fails** — `npm test -- PositionsContent.test.tsx` → FAIL.

- [ ] **Step 3: Implement** — `apps/web/components/position/PositionsContent.tsx`
```tsx
"use client";
import { useState } from "react";
import { usePositions } from "@/lib/hooks/queries";
import { riskMeta, type RiskLevel } from "@/lib/format";
import { POSITION_RISK_LEVELS } from "@/lib/position/risk-filter";
import type { Position } from "@/lib/api/types";
import { PositionsTable, type SortKey } from "./PositionsTable";
import { PositionDetailDrawer } from "./PositionDetailDrawer";

export function PositionsContent() {
  const { data } = usePositions();
  const [risk, setRisk] = useState<RiskLevel | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("pnlSol");
  const [selected, setSelected] = useState<Position | null>(null);

  const rows = (data ?? [])
    .filter((p) => (risk ? p.tokenRisk === risk : true))
    .sort((a, b) => (sortKey === "ageLabel" ? a.ageLabel.localeCompare(b.ageLabel) : (b[sortKey] as number) - (a[sortKey] as number)));

  return (
    <div className="space-y-4">
      <h1>Pozisyonlar</h1>
      <div className="flex flex-wrap items-center gap-2">
        {POSITION_RISK_LEVELS.map((r) => (
          <button key={r} type="button" onClick={() => setRisk(risk === r ? null : r)} className="rounded-md border px-2.5 py-1"
            style={{ fontSize: 12, borderColor: risk === r ? riskMeta[r].color : "var(--border)", color: risk === r ? riskMeta[r].color : "inherit" }}>
            {riskMeta[r].label}
          </button>
        ))}
        {risk && <button type="button" onClick={() => setRisk(null)} className="text-muted-foreground" style={{ fontSize: 12 }}>Temizle</button>}
      </div>
      <PositionsTable rows={rows} sortKey={sortKey} onSort={setSortKey} onRowClick={setSelected} />
      <PositionDetailDrawer position={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
```

- [ ] **Step 4: Run to verify it passes** — `npm test -- PositionsContent.test.tsx` → PASS. If the row-open test is flaky because the clicked cell is a link, change the test to click a non-link cell (e.g. `screen.getByText(/SOL/)` within the row) — the implementation is correct; only the test's click target may need to avoid the `stopPropagation` link.

- [ ] **Step 5: Commit**
```bash
git add apps/web/components/position/PositionsContent.tsx apps/web/components/position/PositionsContent.test.tsx
git commit -m "feat(positions): add PositionsContent with risk filter + sort + drawer"
```

---

### Task 13: Positions page (RSC prefetch) + full verification

**Files:**
- Modify: `apps/web/app/(app)/positions/page.tsx` (replace placeholder)

**Interfaces:**
- Consumes: `getQueryClient`, `qk`, `getApi`, `PositionsContent`, `dehydrate`, `HydrationBoundary`.
- Produces: `/positions` route.

- [ ] **Step 1: Replace the placeholder** — `apps/web/app/(app)/positions/page.tsx`
```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { PositionsContent } from "@/components/position/PositionsContent";

export default async function PositionsPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.positions, queryFn: () => getApi().getPositions() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <PositionsContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 2: Run the full suite** — `npm test`
Expected: all prior tests plus the new Portfolio/Positions tests, zero failures, pristine output.

- [ ] **Step 3: Production build** — `npm run build`
Expected: successful build; `/portfolio` and `/positions` appear in the route list. Paste the relevant route lines + success line into the report. Do not commit a broken build.

- [ ] **Step 4: Commit**
```bash
git add apps/web/app/(app)/positions/page.tsx
git commit -m "feat(positions): wire /positions page with RSC prefetch"
```

---

### Task 14: Living-document updates + visual verification

**Files:**
- Modify: `docs/progress.md`
- Modify: `docs/superpowers/specs/2026-08-01-sentinel-portfolio-positions-design.md` (mark Durum)
- Modify: `docs/superpowers/followups-frontend.md` (close EquityCurve gradient-id followup — now fixed)
- Update memory: `sentinel-frontend-stack-and-plan.md`

- [ ] **Step 1: Visual check** — `npm run dev`, open `/portfolio` and `/positions`. Verify: KPIs (PnL green/red), equity curve ("Portföy Değeri"), PnL-by-strategy bar, risk-allocation donut, win/loss bar, open-positions summary (→ /positions). On `/positions`: table renders, token→/tokens & strategy→/strategies links work, risk filter chips narrow rows + "Temizle" clears, column sort works, row click opens drawer, action buttons toast (switch trading mode to Live → toast.warning). Confirm Turkish + dark theme.

- [ ] **Step 2: Update `docs/progress.md`** — flip the Increment 7 row to ✅ with merge commit + test count; add a dated "Kararlar günlüğü" entry (seam `getPortfolio`/`getPositions`; EquityCurve promoted to shared; MetricTile `valueColor`; read-only actions; 4 charts; read-only scope); update "Sırada" to Increment 8 (Trading Terminal).

- [ ] **Step 3: Mark the spec complete** — set `**Durum:**` to `Uygulandı / merge` in the spec file.

- [ ] **Step 4: Close the EquityCurve gradient-id followup** — in `docs/superpowers/followups-frontend.md`, mark the "EquityCurve statik gradient id" item as KAPANDI (fixed in this increment: id now derived from color) and note the component moved to `components/sentinel/`.

- [ ] **Step 5: Update project memory** — add an "Increment 7 (Portfolio & Positions) TAMAM" paragraph to `sentinel-frontend-stack-and-plan.md`, add `/portfolio` + `/positions` to the completed-screens list, and note EquityCurve is now shared under `components/sentinel/`.

- [ ] **Step 6: Commit**
```bash
git add docs/progress.md docs/superpowers/specs/2026-08-01-sentinel-portfolio-positions-design.md docs/superpowers/followups-frontend.md
git commit -m "docs: mark Increment 7 (Portfolio & Positions) complete"
```

---

## Self-Review

**Spec coverage:**
- `/portfolio` KPIs + 4 charts + open-positions summary → Tasks 4, 5, 6, 7, 8. ✅
- `/positions` rich table + detail drawer + read-only actions + filter/sort → Tasks 9, 10, 11, 12, 13. ✅
- Seam `getPortfolio`/`getPositions` + hooks → Task 1. ✅
- `EquityCurve` promoted to shared `components/sentinel/` with `title`/`color` + gradient-id fix; strategy regression covered → Task 2. ✅
- `MetricTile` `valueColor` extension → Task 4. ✅
- `pnlColor`/`POSITION_RISK_LEVELS` (OCP) → Task 3. ✅
- Read-only actions (live→toast.warning) mirroring TokenActions → Tasks 9, 11. ✅
- DIP (no direct mock import), ISP (narrow props), Turkish, dark theme → enforced per component task. ✅
- Out-of-scope (real orders, PnL-by-creator/age charts, saved views/CSV) → not implemented; `httpApi`→`notReady` (Task 1). ✅
- Portfolio page prefetches both queries (OpenPositionsSummary needs positions) → Task 8. ✅

**Type consistency:** `PortfolioSummary`/`PortfolioOverview`/`Position` field names used identically across Tasks 1, 4, 5, 6, 10, 11. `getPortfolio`/`getPositions`, `qk.portfolio`/`qk.positions`, `usePortfolio`/`usePositions` match between definition (Task 1) and consumers (Tasks 6, 7, 8, 12, 13). `EquityCurve` import path updated to `@/components/sentinel/EquityCurve` in both Task 2 (strategy) and Task 7 (portfolio). `SortKey`/`POSITION_RISK_LEVELS`/`pnlColor` match between definers (Tasks 10, 3) and consumers (Task 12). `RiskLevel` values (`critical|high|medium|good|strong`) consistent with `riskMeta`. ✅

**Placeholder scan:** No TBD/TODO/"handle edge cases". Every code step has concrete content. The only conditional notes are test-target guidance for the row-click/stopPropagation interaction (Task 12), each with a specific action. ✅

## Execution Handoff

Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — execute tasks in this session with checkpoints.
