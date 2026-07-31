# SENTINEL Frontend — Increment 6 (Strategies) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only Strategies screen — `/strategies` card list (status filter) and `/strategies/[id]` detail (conditions, risk/sizing, performance, equity curve, backtest, launchpads/min-scores, version history, audit log) — driven entirely by the mock data seam.

**Architecture:** Follows the established seam pattern `component → useStrategies/useStrategy → getApi() → mock`. New types on `SentinelApi`, a deterministic mock adapter, an OCP status/condition config registry, and small SRP components composed into two server-prefetched pages. No component imports the mock directly.

**Tech Stack:** Next.js 16 (App Router, server-first), TypeScript, Tailwind v4 (Sentinel dark tokens), TanStack Query (RSC prefetch + HydrationBoundary), Recharts (equity curve), Vitest + React Testing Library (TDD).

## Global Constraints

- **UI dili Türkçe** — all visible strings in Turkish (technical tokens/symbols exempt).
- **Seam rule:** no component imports `lib/api/mock` directly; data flows only through `useStrategies`/`useStrategy` → `getApi()`.
- **Clean code + SOLID** is a review criterion: SRP (one responsibility per file, no god-components), OCP (statuses/labels via registry/config — extend, don't modify), DIP (depend on `SentinelApi`, not mock), ISP (narrow props), DRY without premature abstraction.
- **Reuse:** performance/backtest tiles use the shared `MetricTile` (`@/components/sentinel/MetricTile`); equity curve follows the OverviewTab `MiniChart` Recharts pattern; status colors follow the `riskMeta`/`OUTCOME_DEFS` palette convention (hex strings).
- **Scope:** read-only only. No create/edit builder, no deploy/execution, no live toggle. `httpApi` methods resolve to `notReady`.
- **Repo layout:** `apps/web/` (no `src/`; `app/`, `components/`, `lib/` are top-level). Path alias `@/` → `apps/web/`. Run all commands from `apps/web/`.
- **Conventions:** interactive/hook components start with `"use client"`; pages are server components; inline numeric `style={{ fontSize: N }}` and Tailwind classes (`rounded-lg border border-border bg-card`, `text-muted-foreground`, `font-mono tabular-nums`) as in existing code.

---

### Task 1: Data seam — types, contract, http stub, mock adapter, qk, hooks

**Files:**
- Modify: `apps/web/lib/api/types.ts` (append Strategy types)
- Modify: `apps/web/lib/api/contract.ts` (add `getStrategies`/`getStrategy`)
- Modify: `apps/web/lib/api/http.ts` (add `notReady` entries)
- Modify: `apps/web/lib/api/mock.ts` (add deterministic strategy data + adapter methods)
- Modify: `apps/web/lib/get-query-client.ts` (add `qk.strategies`, `qk.strategy`)
- Modify: `apps/web/lib/hooks/queries.ts` (add `useStrategies`, `useStrategy`)
- Test: `apps/web/lib/api/strategies.test.ts` (adapter)
- Test: `apps/web/lib/hooks/strategies.test.tsx` (hooks)

**Interfaces:**
- Consumes: existing `SentinelApi`, `mockApi`, `qk`, `getApi()`, `scoreToLevel`.
- Produces (later tasks rely on these exact names/types):
  - `StrategyStatus = "draft" | "backtesting" | "paper" | "shadow" | "live" | "paused" | "archived"`
  - `ConditionOp = ">" | "<" | ">=" | "<=" | "=="`
  - `StrategyCondition { metric: string; op: ConditionOp; value: number; unit?: string }`
  - `StrategyRow { id: string; name: string; status: StrategyStatus; timeframe: string; winRatePct: number; profitFactor: number; maxDrawdownPct: number; totalTrades: number; netPnlSol: number; lastSignal: string }`
  - `StrategyPerformance { winRatePct; profitFactor; maxDrawdownPct; sharpe; sortino; totalTrades; netPnlSol; expectancy: number }`
  - `EquityPoint { t: number; v: number }`
  - `BacktestSummary { netPnlSol; winRatePct; profitFactor; sharpe; maxDrawdownPct; trades; avgHoldingHours; rugExposurePct: number }`
  - `StrategyVersion { version: string; date: string; note: string }`
  - `AuditEntry { time: string; action: string; detail: string }`
  - `StrategyRisk { riskPerTradePct: number; stopLossPct: number; takeProfitLevels: number[]; maxDrawdownStopPct: number }`
  - `StrategySizing { model: string; sizePct: number }`
  - `StrategyDetail { id; name; status; timeframe; description; entry: StrategyCondition[]; exit: StrategyCondition[]; risk: StrategyRisk; sizing: StrategySizing; supportedLaunchpads: string[]; minScores: { creator: number; safety: number }; performance: StrategyPerformance; equityCurve: EquityPoint[]; backtest: BacktestSummary; versions: StrategyVersion[]; audit: AuditEntry[] }`
  - `mockApi.getStrategies(): Promise<StrategyRow[]>` (6 deterministic rows)
  - `mockApi.getStrategy(id: string): Promise<StrategyDetail>` (deterministic by id; falls back to first strategy for unknown id)
  - `qk.strategies` (const tuple), `qk.strategy(id)` → `["strategy", id]`
  - `useStrategies()`, `useStrategy(id: string)`

- [ ] **Step 1: Append the Strategy types to `lib/api/types.ts`**

Append at the end of the file:

```ts
// --- Strategies (Increment 6) ---
export type StrategyStatus = "draft" | "backtesting" | "paper" | "shadow" | "live" | "paused" | "archived";
export type ConditionOp = ">" | "<" | ">=" | "<=" | "==";

export interface StrategyCondition { metric: string; op: ConditionOp; value: number; unit?: string; }

export interface StrategyRow {
  id: string; name: string; status: StrategyStatus; timeframe: string;
  winRatePct: number; profitFactor: number; maxDrawdownPct: number; totalTrades: number;
  netPnlSol: number; lastSignal: string;
}

export interface StrategyPerformance {
  winRatePct: number; profitFactor: number; maxDrawdownPct: number; sharpe: number; sortino: number;
  totalTrades: number; netPnlSol: number; expectancy: number;
}

export interface EquityPoint { t: number; v: number; }

export interface BacktestSummary {
  netPnlSol: number; winRatePct: number; profitFactor: number; sharpe: number; maxDrawdownPct: number;
  trades: number; avgHoldingHours: number; rugExposurePct: number;
}

export interface StrategyVersion { version: string; date: string; note: string; }
export interface AuditEntry { time: string; action: string; detail: string; }
export interface StrategyRisk { riskPerTradePct: number; stopLossPct: number; takeProfitLevels: number[]; maxDrawdownStopPct: number; }
export interface StrategySizing { model: string; sizePct: number; }

export interface StrategyDetail {
  id: string; name: string; status: StrategyStatus; timeframe: string; description: string;
  entry: StrategyCondition[]; exit: StrategyCondition[];
  risk: StrategyRisk; sizing: StrategySizing;
  supportedLaunchpads: string[]; minScores: { creator: number; safety: number };
  performance: StrategyPerformance; equityCurve: EquityPoint[]; backtest: BacktestSummary;
  versions: StrategyVersion[]; audit: AuditEntry[];
}
```

- [ ] **Step 2: Extend `SentinelApi` in `lib/api/contract.ts`**

Update the type import line to add `StrategyRow, StrategyDetail`, and add two methods after `getCreator`:

```ts
import type { Kpi, TokenRow, AlertEvent, RadarPoint, TokenDetail, FeedEvent, WalletGraph, CreatorRow, CreatorProfile, StrategyRow, StrategyDetail } from "./types";
```
```ts
  getCreators(): Promise<CreatorRow[]>;
  getCreator(address: string): Promise<CreatorProfile>;
  getStrategies(): Promise<StrategyRow[]>;
  getStrategy(id: string): Promise<StrategyDetail>;
```

- [ ] **Step 3: Add `notReady` entries in `lib/api/http.ts`**

Add inside the `httpApi` object, after `getCreator: notReady,`:

```ts
  getStrategies: notReady,
  getStrategy: notReady,
```

- [ ] **Step 4: Write the failing adapter test**

Create `apps/web/lib/api/strategies.test.ts`:

```ts
import { mockApi } from "./mock";

test("getStrategies returns deterministic rows with required fields", async () => {
  const a = await mockApi.getStrategies();
  const b = await mockApi.getStrategies();
  expect(a.length).toBeGreaterThanOrEqual(5);
  expect(a).toEqual(b); // deterministic
  for (const r of a) {
    expect(r.id).toBeTruthy();
    expect(r.name).toBeTruthy();
    expect(typeof r.winRatePct).toBe("number");
    expect(typeof r.netPnlSol).toBe("number");
  }
});

test("getStrategy returns a fully populated detail for a known id", async () => {
  const rows = await mockApi.getStrategies();
  const d = await mockApi.getStrategy(rows[0].id);
  expect(d.id).toBe(rows[0].id);
  expect(d.entry.length).toBeGreaterThan(0);
  expect(d.exit.length).toBeGreaterThan(0);
  expect(d.equityCurve.length).toBeGreaterThan(1);
  expect(d.performance.sharpe).toBeDefined();
  expect(d.backtest.trades).toBeGreaterThan(0);
  expect(d.versions.length).toBeGreaterThan(0);
  expect(d.audit.length).toBeGreaterThan(0);
});

test("getStrategy falls back to a valid detail for an unknown id", async () => {
  const d = await mockApi.getStrategy("does-not-exist");
  expect(d.name).toBeTruthy();
  expect(d.entry.length).toBeGreaterThan(0);
});
```

- [ ] **Step 5: Run the test to verify it fails**

Run: `npm test -- strategies.test.ts`
Expected: FAIL — `getStrategies`/`getStrategy` not a function on `mockApi`.

- [ ] **Step 6: Add the mock data + adapter methods in `lib/api/mock.ts`**

Add the Strategy type imports to the existing `import type { ... } from "./types";` line: `StrategyRow, StrategyDetail, StrategyCondition, EquityPoint, StrategyStatus`. Then add this block just above `export const mockApi: SentinelApi = {`:

```ts
// --- Strategies (Increment 6) ---
const STRATEGY_DEFS: { id: string; name: string; status: StrategyStatus; timeframe: string; desc: string }[] = [
  { id: "momentum-scalp", name: "Momentum Scalp", status: "live", timeframe: "1-5 dk", desc: "Yeni mint sonrası ilk dakikalarda güçlü momentum + güvenli creator ararız." },
  { id: "safe-graduation", name: "Güvenli Graduation", status: "paper", timeframe: "15-60 dk", desc: "Yüksek güvenlik skoru + kilitli likidite ile graduation adaylarını izler." },
  { id: "creator-reputation", name: "Creator İtibar Takibi", status: "shadow", timeframe: "5-30 dk", desc: "Kanıtlanmış creator'ların yeni token'larına erken pozisyon." },
  { id: "liquidity-breakout", name: "Likidite Kırılımı", status: "backtesting", timeframe: "1-10 dk", desc: "Ani likidite artışı + holder büyümesi kombinasyonu." },
  { id: "anti-rug-filter", name: "Anti-Rug Filtre", status: "paused", timeframe: "10-45 dk", desc: "Düşük manipülasyon riski ve dağıtık holder tabanı şartı." },
  { id: "legacy-sniper", name: "Eski Sniper v1", status: "archived", timeframe: "0-2 dk", desc: "Emekliye ayrılmış ilk nesil sniper mantığı." },
];

function seedOf(id: string): number {
  let s = 0;
  for (let i = 0; i < id.length; i++) s = (s * 31 + id.charCodeAt(i)) % 100000;
  return s;
}

function equityCurve(seed: number, len = 40): EquityPoint[] {
  const out: EquityPoint[] = [];
  let v = 100;
  for (let i = 0; i < len; i++) {
    v += Math.sin(seed + i * 0.6) * 3 + ((seed * (i + 1)) % 5) - 1.8;
    out.push({ t: i, v: Math.round(Math.max(20, v) * 100) / 100 });
  }
  return out;
}

function strategyEntry(seed: number): StrategyCondition[] {
  return [
    { metric: "creatorScore", op: ">", value: 70 + (seed % 20) },
    { metric: "tokenSafety", op: ">", value: 65 + (seed % 15) },
    { metric: "liquidity", op: ">", value: 20000 + (seed % 30) * 1000, unit: "USD" },
    { metric: "holderGrowth5m", op: ">", value: 30 + (seed % 40), unit: "%" },
  ];
}
function strategyExit(seed: number): StrategyCondition[] {
  return [
    { metric: "manipulationRisk", op: ">", value: 60 + (seed % 20) },
    { metric: "momentum", op: "<", value: 20 + (seed % 15) },
  ];
}

function strategyRow(def: (typeof STRATEGY_DEFS)[number]): StrategyRow {
  const seed = seedOf(def.id);
  return {
    id: def.id, name: def.name, status: def.status, timeframe: def.timeframe,
    winRatePct: 45 + (seed % 40), profitFactor: Math.round((1 + (seed % 25) / 10) * 100) / 100,
    maxDrawdownPct: 8 + (seed % 25), totalTrades: 40 + (seed % 400),
    netPnlSol: Math.round(((seed % 500) - 120) * 10) / 10, lastSignal: `${1 + (seed % 59)} dk önce`,
  };
}

const strategyRows: StrategyRow[] = STRATEGY_DEFS.map(strategyRow);

function buildStrategy(id: string): StrategyDetail {
  const def = STRATEGY_DEFS.find((d) => d.id === id) ?? STRATEGY_DEFS[0];
  const row = strategyRow(def);
  const seed = seedOf(def.id);
  return {
    id: def.id, name: def.name, status: def.status, timeframe: def.timeframe, description: def.desc,
    entry: strategyEntry(seed), exit: strategyExit(seed),
    risk: { riskPerTradePct: 1 + (seed % 4), stopLossPct: 12 + (seed % 18), takeProfitLevels: [25, 60, 120], maxDrawdownStopPct: 20 + (seed % 15) },
    sizing: { model: seed % 2 === 0 ? "Sabit %" : "Kelly kesirli", sizePct: 2 + (seed % 6) },
    supportedLaunchpads: [LAUNCHPADS[seed % LAUNCHPADS.length], LAUNCHPADS[(seed + 2) % LAUNCHPADS.length]],
    minScores: { creator: 60 + (seed % 25), safety: 55 + (seed % 30) },
    performance: {
      winRatePct: row.winRatePct, profitFactor: row.profitFactor, maxDrawdownPct: row.maxDrawdownPct,
      sharpe: Math.round((0.8 + (seed % 20) / 10) * 100) / 100, sortino: Math.round((1 + (seed % 25) / 10) * 100) / 100,
      totalTrades: row.totalTrades, netPnlSol: row.netPnlSol, expectancy: Math.round(((seed % 30) / 10 - 1) * 100) / 100,
    },
    equityCurve: equityCurve(seed),
    backtest: {
      netPnlSol: row.netPnlSol, winRatePct: row.winRatePct, profitFactor: row.profitFactor,
      sharpe: Math.round((0.8 + (seed % 20) / 10) * 100) / 100, maxDrawdownPct: row.maxDrawdownPct,
      trades: row.totalTrades, avgHoldingHours: 1 + (seed % 12), rugExposurePct: (seed % 8),
    },
    versions: [
      { version: "v1.3", date: "2026-07-28", note: "Stop-loss %2 sıkılaştırıldı" },
      { version: "v1.2", date: "2026-07-20", note: "Min creator skoru 70'e çıkarıldı" },
      { version: "v1.0", date: "2026-07-05", note: "İlk yayın" },
    ],
    audit: [
      { time: "2026-07-30 14:22", action: "Duraklatıldı", detail: "Drawdown eşiği aşıldı" },
      { time: "2026-07-29 09:10", action: "Parametre güncellendi", detail: "Take-profit L2 %60" },
      { time: "2026-07-28 18:45", action: "Yayınlandı", detail: "Shadow → Live" },
    ],
  };
}
```

Then add the two adapter methods to the `mockApi` object (after `getCreator: ...,`):

```ts
  getStrategies: () => delay(strategyRows),
  getStrategy: (id) => delay(buildStrategy(id)),
```

> Note: `LAUNCHPADS` and `delay` are already imported/defined in `mock.ts` (used by earlier increments). If `delay` is not in scope, mirror the existing `delay(...)` call used by `getCreators`.

- [ ] **Step 7: Run the adapter test to verify it passes**

Run: `npm test -- strategies.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 8: Add query keys in `lib/get-query-client.ts`**

Add to the `qk` object after the `creator` line:

```ts
  strategies: ["strategies"] as const,
  strategy: (id: string) => ["strategy", id] as const,
```

- [ ] **Step 9: Write the failing hooks test**

Create `apps/web/lib/hooks/strategies.test.tsx`:

```tsx
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useStrategies, useStrategy } from "./queries";

function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("useStrategies loads rows", async () => {
  const { result } = renderHook(() => useStrategies(), { wrapper });
  await waitFor(() => expect(result.current.data?.length).toBeGreaterThan(0));
});

test("useStrategy loads a detail", async () => {
  const { result } = renderHook(() => useStrategy("momentum-scalp"), { wrapper });
  await waitFor(() => expect(result.current.data?.id).toBe("momentum-scalp"));
});
```

- [ ] **Step 10: Run the hooks test to verify it fails**

Run: `npm test -- hooks/strategies.test.tsx`
Expected: FAIL — `useStrategies`/`useStrategy` not exported.

- [ ] **Step 11: Add hooks in `lib/hooks/queries.ts`**

Append:

```ts
export function useStrategies() {
  return useQuery({ queryKey: qk.strategies, queryFn: () => getApi().getStrategies() });
}
export function useStrategy(id: string) {
  return useQuery({ queryKey: qk.strategy(id), queryFn: () => getApi().getStrategy(id) });
}
```

- [ ] **Step 12: Run both tests to verify they pass**

Run: `npm test -- strategies.test.ts hooks/strategies.test.tsx`
Expected: PASS (5 tests total).

- [ ] **Step 13: Commit**

```bash
git add apps/web/lib/api apps/web/lib/get-query-client.ts apps/web/lib/hooks/queries.ts
git commit -m "feat(strategies): add data seam (types, mock adapter, hooks)"
```

---

### Task 2: Status + condition config registry (OCP)

**Files:**
- Create: `apps/web/lib/strategy/status-defs.ts`
- Test: `apps/web/lib/strategy/status-defs.test.ts`

**Interfaces:**
- Consumes: `StrategyStatus`, `StrategyCondition`, `ConditionOp` from `@/lib/api/types`.
- Produces:
  - `STATUS_DEFS: Record<StrategyStatus, { label: string; color: string }>` (7 entries)
  - `CONDITION_LABELS: Record<string, string>`
  - `formatCondition(c: StrategyCondition): string`

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/strategy/status-defs.test.ts`:

```ts
import { STATUS_DEFS, CONDITION_LABELS, formatCondition } from "./status-defs";
import type { StrategyStatus } from "@/lib/api/types";

test("every status has a label and hex color", () => {
  const statuses: StrategyStatus[] = ["draft", "backtesting", "paper", "shadow", "live", "paused", "archived"];
  for (const s of statuses) {
    expect(STATUS_DEFS[s].label).toBeTruthy();
    expect(STATUS_DEFS[s].color).toMatch(/^#/);
  }
});

test("CONDITION_LABELS maps known metrics to Turkish labels", () => {
  expect(CONDITION_LABELS.creatorScore).toBe("Creator Skoru");
  expect(CONDITION_LABELS.tokenSafety).toBe("Token Güvenliği");
});

test("formatCondition renders label, operator, value and unit", () => {
  expect(formatCondition({ metric: "creatorScore", op: ">", value: 75 })).toBe("Creator Skoru > 75");
  expect(formatCondition({ metric: "liquidity", op: ">", value: 25000, unit: "USD" })).toBe("Likidite > 25000 USD");
});

test("formatCondition falls back to the raw metric key when unmapped", () => {
  expect(formatCondition({ metric: "unknownMetric", op: "<", value: 5 })).toBe("unknownMetric < 5");
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- status-defs.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

Create `apps/web/lib/strategy/status-defs.ts`:

```ts
import type { StrategyStatus, StrategyCondition } from "@/lib/api/types";

export const STATUS_DEFS: Record<StrategyStatus, { label: string; color: string }> = {
  draft: { label: "Taslak", color: "#8A94A6" },
  backtesting: { label: "Backtest", color: "#3E9BFF" },
  paper: { label: "Kâğıt", color: "#7C5CFF" },
  shadow: { label: "Gölge", color: "#FFB020" },
  live: { label: "Canlı", color: "#2FD98B" },
  paused: { label: "Duraklatıldı", color: "#F0476B" },
  archived: { label: "Arşiv", color: "#5A6474" },
};

export const CONDITION_LABELS: Record<string, string> = {
  creatorScore: "Creator Skoru",
  tokenSafety: "Token Güvenliği",
  liquidity: "Likidite",
  holderGrowth5m: "Holder Büyümesi 5dk",
  manipulationRisk: "Manipülasyon Riski",
  momentum: "Momentum",
  ageSeconds: "Yaş",
};

export function formatCondition(c: StrategyCondition): string {
  const label = CONDITION_LABELS[c.metric] ?? c.metric;
  const unit = c.unit ? ` ${c.unit}` : "";
  return `${label} ${c.op} ${c.value}${unit}`;
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm test -- status-defs.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/strategy
git commit -m "feat(strategies): add status + condition config registry"
```

---

### Task 3: StatusBadge component

**Files:**
- Create: `apps/web/components/strategy/StatusBadge.tsx`
- Test: `apps/web/components/strategy/StatusBadge.test.tsx`

**Interfaces:**
- Consumes: `STATUS_DEFS`, `StrategyStatus`.
- Produces: `StatusBadge({ status }: { status: StrategyStatus })`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/StatusBadge.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "./StatusBadge";

test("renders the Turkish label for a status", () => {
  render(<StatusBadge status="live" />);
  expect(screen.getByText("Canlı")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- StatusBadge.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

Create `apps/web/components/strategy/StatusBadge.tsx`:

```tsx
import { STATUS_DEFS } from "@/lib/strategy/status-defs";
import type { StrategyStatus } from "@/lib/api/types";

export function StatusBadge({ status }: { status: StrategyStatus }) {
  const d = STATUS_DEFS[status];
  return (
    <span
      className="inline-flex items-center rounded-md px-2 py-0.5 font-medium"
      style={{ fontSize: 11, color: d.color, background: `${d.color}1A`, border: `1px solid ${d.color}40` }}
    >
      {d.label}
    </span>
  );
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm test -- StatusBadge.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/strategy/StatusBadge.tsx apps/web/components/strategy/StatusBadge.test.tsx
git commit -m "feat(strategies): add StatusBadge"
```

---

### Task 4: ConditionList component

**Files:**
- Create: `apps/web/components/strategy/ConditionList.tsx`
- Test: `apps/web/components/strategy/ConditionList.test.tsx`

**Interfaces:**
- Consumes: `formatCondition`, `StrategyCondition`.
- Produces: `ConditionList({ title, conditions }: { title: string; conditions: StrategyCondition[] })`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/ConditionList.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { ConditionList } from "./ConditionList";

test("renders the title and each condition via formatCondition", () => {
  render(
    <ConditionList
      title="Giriş (IF)"
      conditions={[
        { metric: "creatorScore", op: ">", value: 75 },
        { metric: "liquidity", op: ">", value: 25000, unit: "USD" },
      ]}
    />
  );
  expect(screen.getByText("Giriş (IF)")).toBeInTheDocument();
  expect(screen.getByText("Creator Skoru > 75")).toBeInTheDocument();
  expect(screen.getByText("Likidite > 25000 USD")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- ConditionList.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

Create `apps/web/components/strategy/ConditionList.tsx`:

```tsx
import { formatCondition } from "@/lib/strategy/status-defs";
import type { StrategyCondition } from "@/lib/api/types";

export function ConditionList({ title, conditions }: { title: string; conditions: StrategyCondition[] }) {
  return (
    <div className="space-y-2">
      <div className="text-muted-foreground" style={{ fontSize: 12 }}>{title}</div>
      <div className="flex flex-wrap gap-2">
        {conditions.map((c, i) => (
          <span
            key={`${c.metric}-${i}`}
            className="rounded-md border border-border bg-surface-2 px-2 py-1 font-mono"
            style={{ fontSize: 12 }}
          >
            {formatCondition(c)}
          </span>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm test -- ConditionList.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/strategy/ConditionList.tsx apps/web/components/strategy/ConditionList.test.tsx
git commit -m "feat(strategies): add ConditionList"
```

---

### Task 5: StrategyCard component

**Files:**
- Create: `apps/web/components/strategy/StrategyCard.tsx`
- Test: `apps/web/components/strategy/StrategyCard.test.tsx`

**Interfaces:**
- Consumes: `StrategyRow`, `StatusBadge`, `next/link`.
- Produces: `StrategyCard({ row }: { row: StrategyRow })` — links to `/strategies/<id>`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/StrategyCard.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { StrategyCard } from "./StrategyCard";
import type { StrategyRow } from "@/lib/api/types";

const row: StrategyRow = {
  id: "momentum-scalp", name: "Momentum Scalp", status: "live", timeframe: "1-5 dk",
  winRatePct: 62, profitFactor: 2.1, maxDrawdownPct: 14, totalTrades: 220,
  netPnlSol: 128.4, lastSignal: "3 dk önce",
};

test("shows name, status badge, metrics and links to the detail page", () => {
  render(<StrategyCard row={row} />);
  expect(screen.getByText("Momentum Scalp")).toBeInTheDocument();
  expect(screen.getByText("Canlı")).toBeInTheDocument();
  expect(screen.getByText(/62/)).toBeInTheDocument();
  const link = screen.getByRole("link");
  expect(link.getAttribute("href")).toBe("/strategies/momentum-scalp");
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- StrategyCard.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

Create `apps/web/components/strategy/StrategyCard.tsx`:

```tsx
import Link from "next/link";
import type { StrategyRow } from "@/lib/api/types";
import { StatusBadge } from "./StatusBadge";

function pnlColor(v: number) {
  return v >= 0 ? "#2FD98B" : "#F0476B";
}

export function StrategyCard({ row }: { row: StrategyRow }) {
  const stats: { label: string; value: string; color?: string }[] = [
    { label: "Kazanç Oranı", value: `%${row.winRatePct}` },
    { label: "Profit Factor", value: row.profitFactor.toFixed(2) },
    { label: "Maks. Drawdown", value: `%${row.maxDrawdownPct}` },
    { label: "İşlem", value: `${row.totalTrades}` },
    { label: "Net PnL", value: `${row.netPnlSol} SOL`, color: pnlColor(row.netPnlSol) },
  ];
  return (
    <Link
      href={`/strategies/${row.id}`}
      className="block rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/50"
    >
      <div className="flex items-center justify-between gap-2">
        <div className="font-medium">{row.name}</div>
        <StatusBadge status={row.status} />
      </div>
      <div className="mt-1 text-muted-foreground" style={{ fontSize: 11 }}>
        {row.timeframe} · son sinyal {row.lastSignal}
      </div>
      <div className="mt-3 grid grid-cols-3 gap-2">
        {stats.map((s) => (
          <div key={s.label}>
            <div className="text-muted-foreground" style={{ fontSize: 10 }}>{s.label}</div>
            <div className="font-mono tabular-nums" style={{ fontSize: 14, color: s.color }}>{s.value}</div>
          </div>
        ))}
      </div>
    </Link>
  );
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm test -- StrategyCard.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/strategy/StrategyCard.tsx apps/web/components/strategy/StrategyCard.test.tsx
git commit -m "feat(strategies): add StrategyCard"
```

---

### Task 6: StrategiesListContent (list + status filter)

**Files:**
- Create: `apps/web/components/strategy/StrategiesListContent.tsx`
- Test: `apps/web/components/strategy/StrategiesListContent.test.tsx`

**Interfaces:**
- Consumes: `useStrategies`, `StrategyCard`, `STATUS_DEFS`, `StrategyStatus`.
- Produces: `StrategiesListContent()` — client component; renders card grid + status filter chips.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/StrategiesListContent.test.tsx`:

```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { StrategiesListContent } from "./StrategiesListContent";

function renderList() {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <StrategiesListContent />
    </QueryClientProvider>
  );
}

test("renders strategy cards from the seam", async () => {
  renderList();
  await waitFor(() => expect(screen.getByText("Momentum Scalp")).toBeInTheDocument());
});

test("status filter narrows the visible cards", async () => {
  renderList();
  await waitFor(() => expect(screen.getByText("Momentum Scalp")).toBeInTheDocument());
  // "Canlı" filter keeps the live strategy, drops an archived one
  fireEvent.click(screen.getByRole("button", { name: "Canlı" }));
  await waitFor(() => {
    expect(screen.getByText("Momentum Scalp")).toBeInTheDocument();
    expect(screen.queryByText("Eski Sniper v1")).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- StrategiesListContent.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

Create `apps/web/components/strategy/StrategiesListContent.tsx`:

```tsx
"use client";
import { useState } from "react";
import { useStrategies } from "@/lib/hooks/queries";
import { STATUS_DEFS } from "@/lib/strategy/status-defs";
import type { StrategyStatus } from "@/lib/api/types";
import { StrategyCard } from "./StrategyCard";

const STATUS_ORDER: StrategyStatus[] = ["live", "shadow", "paper", "backtesting", "draft", "paused", "archived"];

export function StrategiesListContent() {
  const { data } = useStrategies();
  const [active, setActive] = useState<StrategyStatus | null>(null);
  const rows = data ?? [];
  const visible = active ? rows.filter((r) => r.status === active) : rows;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <h1>Stratejiler</h1>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {STATUS_ORDER.map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => setActive(active === s ? null : s)}
            className="rounded-md border px-2.5 py-1"
            style={{
              fontSize: 12,
              borderColor: active === s ? STATUS_DEFS[s].color : "var(--border)",
              color: active === s ? STATUS_DEFS[s].color : "inherit",
            }}
          >
            {STATUS_DEFS[s].label}
          </button>
        ))}
        {active && (
          <button type="button" onClick={() => setActive(null)} className="text-muted-foreground" style={{ fontSize: 12 }}>
            Temizle
          </button>
        )}
      </div>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        {visible.map((r) => (
          <StrategyCard key={r.id} row={r} />
        ))}
      </div>
    </div>
  );
}
```

> Note: the filter button uses `role="button"` implicitly via `<button>`; the test's `getByRole("button", { name: "Canlı" })` matches the status chip. The border uses the CSS var `var(--border)` already defined by the Tailwind theme.

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm test -- StrategiesListContent.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/strategy/StrategiesListContent.tsx apps/web/components/strategy/StrategiesListContent.test.tsx
git commit -m "feat(strategies): add StrategiesListContent with status filter"
```

---

### Task 7: Performance + backtest panels (MetricTile reuse)

**Files:**
- Create: `apps/web/components/strategy/StrategyPerformancePanel.tsx`
- Create: `apps/web/components/strategy/BacktestSummaryPanel.tsx`
- Test: `apps/web/components/strategy/StrategyPanels.test.tsx`

**Interfaces:**
- Consumes: `StrategyPerformance`, `BacktestSummary` from `@/lib/api/types`; `MetricTile` from `@/components/sentinel/MetricTile`.
- Produces:
  - `StrategyPerformancePanel({ performance }: { performance: StrategyPerformance })`
  - `BacktestSummaryPanel({ backtest }: { backtest: BacktestSummary })`

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/StrategyPanels.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { StrategyPerformancePanel } from "./StrategyPerformancePanel";
import { BacktestSummaryPanel } from "./BacktestSummaryPanel";
import type { StrategyPerformance, BacktestSummary } from "@/lib/api/types";

const perf: StrategyPerformance = {
  winRatePct: 62, profitFactor: 2.1, maxDrawdownPct: 14, sharpe: 1.8, sortino: 2.3,
  totalTrades: 220, netPnlSol: 128.4, expectancy: 0.4,
};
const bt: BacktestSummary = {
  netPnlSol: 128.4, winRatePct: 62, profitFactor: 2.1, sharpe: 1.8, maxDrawdownPct: 14,
  trades: 220, avgHoldingHours: 6, rugExposurePct: 3,
};

test("performance panel shows sharpe and win rate tiles", () => {
  render(<StrategyPerformancePanel performance={perf} />);
  expect(screen.getByText("Sharpe")).toBeInTheDocument();
  expect(screen.getByText("1.8")).toBeInTheDocument();
});

test("backtest panel shows trades and rug exposure", () => {
  render(<BacktestSummaryPanel backtest={bt} />);
  expect(screen.getByText("İşlem Sayısı")).toBeInTheDocument();
  expect(screen.getByText("Rug Maruziyeti")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- StrategyPanels.test.tsx`
Expected: FAIL — modules not found.

- [ ] **Step 3: Write `StrategyPerformancePanel.tsx`**

Create `apps/web/components/strategy/StrategyPerformancePanel.tsx`:

```tsx
import type { StrategyPerformance } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";

export function StrategyPerformancePanel({ performance: p }: { performance: StrategyPerformance }) {
  const tiles: { label: string; value: string }[] = [
    { label: "Kazanç Oranı", value: `%${p.winRatePct}` },
    { label: "Profit Factor", value: p.profitFactor.toFixed(2) },
    { label: "Maks. Drawdown", value: `%${p.maxDrawdownPct}` },
    { label: "Sharpe", value: `${p.sharpe}` },
    { label: "Sortino", value: `${p.sortino}` },
    { label: "İşlem", value: `${p.totalTrades}` },
    { label: "Net PnL", value: `${p.netPnlSol} SOL` },
    { label: "Beklenen Değer", value: `${p.expectancy} SOL` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} />)}
    </div>
  );
}
```

- [ ] **Step 4: Write `BacktestSummaryPanel.tsx`**

Create `apps/web/components/strategy/BacktestSummaryPanel.tsx`:

```tsx
import type { BacktestSummary } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";

export function BacktestSummaryPanel({ backtest: b }: { backtest: BacktestSummary }) {
  const tiles: { label: string; value: string }[] = [
    { label: "Net PnL", value: `${b.netPnlSol} SOL` },
    { label: "Kazanç Oranı", value: `%${b.winRatePct}` },
    { label: "Profit Factor", value: b.profitFactor.toFixed(2) },
    { label: "Sharpe", value: `${b.sharpe}` },
    { label: "Maks. Drawdown", value: `%${b.maxDrawdownPct}` },
    { label: "İşlem Sayısı", value: `${b.trades}` },
    { label: "Ort. Tutma", value: `${b.avgHoldingHours}s` },
    { label: "Rug Maruziyeti", value: `%${b.rugExposurePct}` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} />)}
    </div>
  );
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `npm test -- StrategyPanels.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 6: Commit**

```bash
git add apps/web/components/strategy/StrategyPerformancePanel.tsx apps/web/components/strategy/BacktestSummaryPanel.tsx apps/web/components/strategy/StrategyPanels.test.tsx
git commit -m "feat(strategies): add performance + backtest panels"
```

---

### Task 8: EquityCurve (Recharts)

**Files:**
- Create: `apps/web/components/strategy/EquityCurve.tsx`
- Test: `apps/web/components/strategy/EquityCurve.test.tsx`

**Interfaces:**
- Consumes: `EquityPoint` from `@/lib/api/types`; Recharts.
- Produces: `EquityCurve({ data }: { data: EquityPoint[] })`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/EquityCurve.test.tsx`:

```tsx
import { render } from "@testing-library/react";
import { EquityCurve } from "./EquityCurve";
import type { EquityPoint } from "@/lib/api/types";

test("renders without crashing given equity points (smoke)", () => {
  const data: EquityPoint[] = Array.from({ length: 10 }, (_, i) => ({ t: i, v: 100 + i }));
  const { container } = render(
    <div style={{ width: 400, height: 200 }}>
      <EquityCurve data={data} />
    </div>
  );
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
```

> Note: Recharts `ResponsiveContainer` needs a sized parent in JSDOM. Wrapping in a fixed-size `div` is the pattern used by the existing OpportunityRadar/OverviewTab tests; if the container query is flaky, assert on `container.querySelector("svg, .recharts-wrapper")` instead — either satisfies the smoke intent.

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- EquityCurve.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

Create `apps/web/components/strategy/EquityCurve.tsx`:

```tsx
"use client";
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { EquityPoint } from "@/lib/api/types";

export function EquityCurve({ data }: { data: EquityPoint[] }) {
  const color = "#2FD98B";
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Equity Curve</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="equity-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="t" hide />
            <YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip
              contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }}
              labelFormatter={() => ""}
            />
            <Area type="monotone" dataKey="v" stroke={color} strokeWidth={1.5} fill="url(#equity-grad)" />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm test -- EquityCurve.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/strategy/EquityCurve.tsx apps/web/components/strategy/EquityCurve.test.tsx
git commit -m "feat(strategies): add EquityCurve chart"
```

---

### Task 9: VersionHistory + AuditLog

**Files:**
- Create: `apps/web/components/strategy/VersionHistory.tsx`
- Create: `apps/web/components/strategy/AuditLog.tsx`
- Test: `apps/web/components/strategy/StrategyLogs.test.tsx`

**Interfaces:**
- Consumes: `StrategyVersion`, `AuditEntry` from `@/lib/api/types`.
- Produces:
  - `VersionHistory({ versions }: { versions: StrategyVersion[] })`
  - `AuditLog({ audit }: { audit: AuditEntry[] })`

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/StrategyLogs.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { VersionHistory } from "./VersionHistory";
import { AuditLog } from "./AuditLog";
import type { StrategyVersion, AuditEntry } from "@/lib/api/types";

const versions: StrategyVersion[] = [{ version: "v1.3", date: "2026-07-28", note: "Stop-loss sıkılaştırıldı" }];
const audit: AuditEntry[] = [{ time: "2026-07-30 14:22", action: "Duraklatıldı", detail: "Drawdown eşiği" }];

test("version history lists versions with notes", () => {
  render(<VersionHistory versions={versions} />);
  expect(screen.getByText("v1.3")).toBeInTheDocument();
  expect(screen.getByText("Stop-loss sıkılaştırıldı")).toBeInTheDocument();
});

test("audit log lists actions with details", () => {
  render(<AuditLog audit={audit} />);
  expect(screen.getByText("Duraklatıldı")).toBeInTheDocument();
  expect(screen.getByText("Drawdown eşiği")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- StrategyLogs.test.tsx`
Expected: FAIL — modules not found.

- [ ] **Step 3: Write `VersionHistory.tsx`**

Create `apps/web/components/strategy/VersionHistory.tsx`:

```tsx
import type { StrategyVersion } from "@/lib/api/types";

export function VersionHistory({ versions }: { versions: StrategyVersion[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="mb-3 font-medium" style={{ fontSize: 13 }}>Sürüm Geçmişi</div>
      <ul className="space-y-2">
        {versions.map((v) => (
          <li key={v.version} className="flex items-start gap-3">
            <span className="font-mono text-primary" style={{ fontSize: 12 }}>{v.version}</span>
            <span className="text-muted-foreground" style={{ fontSize: 11 }}>{v.date}</span>
            <span style={{ fontSize: 12 }}>{v.note}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 4: Write `AuditLog.tsx`**

Create `apps/web/components/strategy/AuditLog.tsx`:

```tsx
import type { AuditEntry } from "@/lib/api/types";

export function AuditLog({ audit }: { audit: AuditEntry[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="mb-3 font-medium" style={{ fontSize: 13 }}>Denetim Kaydı</div>
      <ul className="space-y-2">
        {audit.map((a, i) => (
          <li key={`${a.time}-${i}`} className="flex items-start gap-3">
            <span className="font-mono text-muted-foreground" style={{ fontSize: 11 }}>{a.time}</span>
            <span className="font-medium" style={{ fontSize: 12 }}>{a.action}</span>
            <span className="text-muted-foreground" style={{ fontSize: 12 }}>{a.detail}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `npm test -- StrategyLogs.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 6: Commit**

```bash
git add apps/web/components/strategy/VersionHistory.tsx apps/web/components/strategy/AuditLog.tsx apps/web/components/strategy/StrategyLogs.test.tsx
git commit -m "feat(strategies): add version history + audit log"
```

---

### Task 10: StrategyDetailContent (composition + loading/error)

**Files:**
- Create: `apps/web/components/strategy/StrategyDetailContent.tsx`
- Test: `apps/web/components/strategy/StrategyDetailContent.test.tsx`

**Interfaces:**
- Consumes: `useStrategy`, `StatusBadge`, `ConditionList`, `StrategyPerformancePanel`, `BacktestSummaryPanel`, `EquityCurve`, `VersionHistory`, `AuditLog`, `MetricTile`, `Skeleton`, `formatUsd`.
- Produces: `StrategyDetailContent({ id }: { id: string })`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/strategy/StrategyDetailContent.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { StrategyDetailContent } from "./StrategyDetailContent";

test("renders the composed detail (header + conditions + backtest section)", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <StrategyDetailContent id="momentum-scalp" />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("Momentum Scalp")).toBeInTheDocument());
  expect(screen.getByText("Giriş (IF)")).toBeInTheDocument();
  expect(screen.getByText("Çıkış (THEN)")).toBeInTheDocument();
  expect(screen.getByText("Backtest Özeti")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- StrategyDetailContent.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the implementation**

Create `apps/web/components/strategy/StrategyDetailContent.tsx`:

```tsx
"use client";
import { useStrategy } from "@/lib/hooks/queries";
import { formatUsd } from "@/lib/format";
import { Skeleton } from "@/components/ui/skeleton";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { StatusBadge } from "./StatusBadge";
import { ConditionList } from "./ConditionList";
import { StrategyPerformancePanel } from "./StrategyPerformancePanel";
import { BacktestSummaryPanel } from "./BacktestSummaryPanel";
import { EquityCurve } from "./EquityCurve";
import { VersionHistory } from "./VersionHistory";
import { AuditLog } from "./AuditLog";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-3">
      <h2 style={{ fontSize: 14 }}>{title}</h2>
      {children}
    </section>
  );
}

export function StrategyDetailContent({ id }: { id: string }) {
  const { data: s, isError } = useStrategy(id);
  if (isError) {
    return <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Strateji bulunamadı: {id}</div>;
  }
  if (!s) {
    return <div className="space-y-4"><Skeleton className="h-24 w-full" /><Skeleton className="h-40 w-full" /></div>;
  }
  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-3">
            <h1>{s.name}</h1>
            <StatusBadge status={s.status} />
          </div>
          <div className="mt-1 text-muted-foreground" style={{ fontSize: 12 }}>{s.timeframe} · {s.description}</div>
        </div>
      </div>

      <Section title="Koşullar">
        <div className="grid grid-cols-1 gap-4 rounded-lg border border-border bg-card p-4 md:grid-cols-2">
          <ConditionList title="Giriş (IF)" conditions={s.entry} />
          <ConditionList title="Çıkış (THEN)" conditions={s.exit} />
        </div>
      </Section>

      <Section title="Risk & Pozisyon Boyutlandırma">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MetricTile label="İşlem Başı Risk" value={`%${s.risk.riskPerTradePct}`} />
          <MetricTile label="Stop-Loss" value={`%${s.risk.stopLossPct}`} />
          <MetricTile label="Take-Profit" value={s.risk.takeProfitLevels.map((t) => `%${t}`).join(" / ")} />
          <MetricTile label="Drawdown Stop" value={`%${s.risk.maxDrawdownStopPct}`} />
          <MetricTile label="Boyut Modeli" value={s.sizing.model} />
          <MetricTile label="Pozisyon Boyutu" value={`%${s.sizing.sizePct}`} />
          <MetricTile label="Min Creator Skoru" value={`${s.minScores.creator}`} />
          <MetricTile label="Min Güvenlik Skoru" value={`${s.minScores.safety}`} />
        </div>
      </Section>

      <Section title="Performans">
        <StrategyPerformancePanel performance={s.performance} />
      </Section>

      <Section title="Equity Curve">
        <EquityCurve data={s.equityCurve} />
      </Section>

      <Section title="Backtest Özeti">
        <BacktestSummaryPanel backtest={s.backtest} />
      </Section>

      <Section title="Desteklenen Launchpad'ler">
        <div className="flex flex-wrap gap-2">
          {s.supportedLaunchpads.map((l) => (
            <span key={l} className="rounded-md border border-border bg-surface-2 px-2 py-1" style={{ fontSize: 12 }}>{l}</span>
          ))}
        </div>
      </Section>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <VersionHistory versions={s.versions} />
        <AuditLog audit={s.audit} />
      </div>
    </div>
  );
}
```

> Note: `formatUsd` is imported for parity with sibling screens even if unused here; if the lint config flags unused imports, drop the import. Keep `MetricTile` reuse for the risk/sizing block (no bespoke tile component — DRY).

- [ ] **Step 4: Run the test to verify it passes**

Run: `npm test -- StrategyDetailContent.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/strategy/StrategyDetailContent.tsx apps/web/components/strategy/StrategyDetailContent.test.tsx
git commit -m "feat(strategies): compose StrategyDetailContent"
```

---

### Task 11: Pages (list + detail) with RSC prefetch

**Files:**
- Modify: `apps/web/app/(app)/strategies/page.tsx` (replace placeholder)
- Create: `apps/web/app/(app)/strategies/[id]/page.tsx`

**Interfaces:**
- Consumes: `getQueryClient`, `qk`, `getApi`, `StrategiesListContent`, `StrategyDetailContent`, `dehydrate`, `HydrationBoundary`.
- Produces: `/strategies` (list, real) and `/strategies/[id]` (detail) routes.

- [ ] **Step 1: Replace the list page placeholder**

Overwrite `apps/web/app/(app)/strategies/page.tsx`:

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { StrategiesListContent } from "@/components/strategy/StrategiesListContent";

export default async function StrategiesPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.strategies, queryFn: () => getApi().getStrategies() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <StrategiesListContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 2: Create the detail page**

Create `apps/web/app/(app)/strategies/[id]/page.tsx`:

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { StrategyDetailContent } from "@/components/strategy/StrategyDetailContent";

export default async function StrategyDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.strategy(id), queryFn: () => getApi().getStrategy(id) });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <StrategyDetailContent id={id} />
    </HydrationBoundary>
  );
}
```

> Note: Next 16 App Router passes `params` as a Promise — match the `await params` pattern used by `app/(app)/creators/[address]/page.tsx` and `app/(app)/tokens/[mint]/page.tsx`. Confirm the exact param typing against those files before writing.

- [ ] **Step 3: Run the full test suite**

Run: `npm test`
Expected: PASS — all prior tests (81) plus the new Strategies tests, zero failures. Pristine output (no unhandled warnings).

- [ ] **Step 4: Build to verify production compile + typecheck**

Run: `npm run build`
Expected: Successful build; `/strategies` and `/strategies/[id]` appear in the route list.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/(app)/strategies
git commit -m "feat(strategies): wire list + detail pages with RSC prefetch"
```

---

### Task 12: Living-document updates + visual verification

**Files:**
- Modify: `docs/progress.md`
- Modify: `docs/superpowers/specs/2026-07-31-sentinel-strategies-design.md` (mark Durum: Tamam)
- Update memory: `sentinel-frontend-stack-and-plan.md`

- [ ] **Step 1: Visual check**

Run: `npm run dev`, open `/strategies`, verify: card grid renders, status filter chips narrow cards + "Temizle" clears, clicking a card navigates to `/strategies/[id]`, detail shows header + status + IF/THEN conditions + risk/sizing tiles + performance + equity curve chart + backtest summary + launchpads + version history + audit log. Confirm Turkish labels and dark theme.

- [ ] **Step 2: Update `docs/progress.md`**

Flip the Increment 6 table row to ✅ with the merge commit + test count, and add a dated entry to "Kararlar günlüğü" summarizing the Strategies increment (seam `getStrategies`/`getStrategy`, `STATUS_DEFS`/`CONDITION_LABELS` registry, `MetricTile` reuse, read-only scope, builder deferred). Update the "Sırada" section to Increment 7 (Portfolio / Positions).

- [ ] **Step 3: Mark the spec complete**

In `docs/superpowers/specs/2026-07-31-sentinel-strategies-design.md`, change `**Durum:** Onay bekliyor` → `**Durum:** Tamam (master'a merge)`.

- [ ] **Step 4: Update project memory**

Update `sentinel-frontend-stack-and-plan.md` with an "Increment 6 (Strategies) TAMAM" paragraph mirroring the Increment 5 entry style, and add `/strategies` + `/strategies/[id]` to the completed-screens list.

- [ ] **Step 5: Commit**

```bash
git add docs/progress.md docs/superpowers/specs/2026-07-31-sentinel-strategies-design.md
git commit -m "docs: mark Increment 6 (Strategies) complete"
```

---

## Self-Review

**Spec coverage:**
- `/strategies` list + status filter → Tasks 5, 6, 11. ✅
- `/strategies/[id]` read-only detail (conditions, risk/sizing, performance, equity curve, backtest, launchpads/min-scores, versions, audit) → Tasks 4, 7, 8, 9, 10, 11. ✅
- Seam `getStrategies`/`getStrategy` + hooks → Task 1. ✅
- `STATUS_DEFS`/`CONDITION_LABELS`/`formatCondition` (OCP) → Task 2. ✅
- `MetricTile` reuse, Recharts equity curve → Tasks 7, 8, 10. ✅
- DIP (no direct mock import), ISP (narrow props) → enforced in every component task. ✅
- Out-of-scope (builder/deploy/execution) → not implemented; `httpApi` → `notReady` (Task 1). ✅
- Test strategy groups (spec §5) → covered by Tasks 1–10 tests. ✅
- Acceptance criteria (spec §6) → verified in Tasks 11–12. ✅

**Type consistency:** `StrategyRow`/`StrategyDetail` field names (`winRatePct`, `profitFactor`, `maxDrawdownPct`, `netPnlSol`, `equityCurve`, `minScores`) are used identically across Tasks 1, 5, 7, 8, 10. `getStrategies`/`getStrategy`, `qk.strategies`/`qk.strategy(id)`, `useStrategies`/`useStrategy` names match between definition (Task 1) and consumers (Tasks 6, 10, 11). `STATUS_DEFS`/`CONDITION_LABELS`/`formatCondition` match between Task 2 and Tasks 3, 4. ✅

**Placeholder scan:** No TBD/TODO/"handle edge cases" — every code step has concrete content. The only conditional notes are fallback instructions for JSDOM/lint quirks, each with a specific action. ✅

## Execution Handoff

Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — Execute tasks in this session with checkpoints.
