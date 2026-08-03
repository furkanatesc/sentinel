# SENTINEL Frontend Increment 9 — Backtesting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Backtesting results screen at `/backtesting` — a left parameter form that runs a simulated deterministic-seeded backtest via the mock seam and renders 10 metrics + 6 charts (equity curve, drawdown, monthly return, trade distribution, PnL-by-score, entry/exit scatter).

**Architecture:** Server component prefetches `getStrategies` (for the strategy dropdown) and hydrates a client `BacktestContent` that holds `submittedParams` state. The params form calls `onRun(params)`; `useBacktest(params)` (enabled only after submit, keyed by params) fetches `getApi().runBacktest(params)` — a deterministic result seeded by the params, so different params yield different numbers. All data flows `component → hook → getApi() → mock` (DIP). Event Replay is deferred to a later increment.

**Tech Stack:** Next.js 16 App Router, TypeScript, TanStack Query, Tailwind v4, Recharts (area/bar/composed+scatter), Vitest + React Testing Library. Reuse: `EquityCurve`, `MetricTile`, `pnlColor`, `useStrategies`.

## Global Constraints

- UI language **Turkish** for all visible text; technical tokens/symbols exempt. Dark-only theme.
- **DIP:** no component imports `@/lib/api/mock`; all data flows through hooks → `getApi()`. `httpApi.runBacktest` → `notReady` (reject).
- **Clean code + SOLID** is a review criterion (SRP/OCP/DIP/ISP; config-driven registries; small focused files).
- Monorepo frontend root: `apps/web/`. Alias `@/` → `apps/web/`. No `src/`.
- `runBacktest` mock must be **DETERMINISTIC** (same params → equal result) AND **params-sensitive** (different params → different result).
- Vitest: `globals: true` (no need to import `test`/`expect`/`vi` except where a file already does). Setup `apps/web/test/setup.ts`. Mock 40 ms delay → data-dependent assertions use `await waitFor(...)`.
- Recharts smoke tests: wrap in `<div style={{ width: 400, height: 240 }}>` and assert `container.querySelector(".recharts-responsive-container")` is truthy.
- Run commands from `apps/web/` (e.g. `cd apps/web && npx vitest run <path>`).
- No new dependencies (Recharts already present; native styled `<select>` for dropdowns).

---

## File Structure

**Create:**
- `apps/web/lib/backtest/backtest-defs.ts` — registries, defaults, metric defs.
- `apps/web/lib/backtest/validate.ts` — pure `validateParams`.
- `apps/web/components/backtest/BacktestParamsForm.tsx` — parameter form (+ `useStrategies`).
- `apps/web/components/backtest/BacktestMetrics.tsx` — 10 metric tiles.
- `apps/web/components/backtest/DrawdownChart.tsx` — Recharts area.
- `apps/web/components/backtest/MonthlyReturnChart.tsx` — Recharts bar.
- `apps/web/components/backtest/TradeDistributionChart.tsx` — Recharts bar.
- `apps/web/components/backtest/PnlByScoreChart.tsx` — Recharts bar (pnlColor per-Cell).
- `apps/web/components/backtest/EntryExitChart.tsx` — Recharts ComposedChart (Line + Scatter).
- `apps/web/components/backtest/BacktestContent.tsx` — composition + `submittedParams` state.
- Test files colocated.

**Modify:**
- `apps/web/lib/api/types.ts` — add Backtesting types.
- `apps/web/lib/api/contract.ts` — add `runBacktest`.
- `apps/web/lib/api/http.ts` — add `runBacktest: notReady`.
- `apps/web/lib/api/mock.ts` — add deterministic `runBacktest` builder + adapter.
- `apps/web/lib/get-query-client.ts` — add `qk.backtest`.
- `apps/web/lib/hooks/queries.ts` — add `useBacktest`.
- `apps/web/app/(app)/backtesting/page.tsx` — replace placeholder with RSC prefetch page.

---

## Task 1: Data seam (types, contract, http, mock, qk, hook)

**Files:**
- Modify: `apps/web/lib/api/types.ts`, `apps/web/lib/api/contract.ts`, `apps/web/lib/api/http.ts`, `apps/web/lib/api/mock.ts`, `apps/web/lib/get-query-client.ts`, `apps/web/lib/hooks/queries.ts`
- Test: `apps/web/lib/api/backtest.test.ts`, `apps/web/lib/hooks/backtest-hooks.test.tsx`

**Interfaces:**
- Produces (types): `BacktestParams`, `BacktestMetrics`, `MonthlyReturn`, `DistributionBucket`, `ScorePnl`, `DrawdownPoint`, `BacktestTrade`, `BacktestResult`.
- Produces (api): `runBacktest(params: BacktestParams): Promise<BacktestResult>`.
- Produces (qk): `qk.backtest(params)`. Produces (hook): `useBacktest(params: BacktestParams | null)`.
- Consumes: existing `EquityPoint` type, `seedOf`/`toSeries`/`delay` in `mock.ts`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/api/backtest.test.ts`:
```ts
import { mockApi } from "./mock";
import type { BacktestParams } from "./types";

const base: BacktestParams = {
  strategyId: "momentum-scalp", rangePreset: "30g", initialCapitalSol: 100, maxPositions: 5,
  slippageModel: "dynamic", priorityFee: 0.0001, latencyModel: "realistic", liquidityModel: "constrained",
  minCreatorScore: 60, minTokenSafety: 55,
};

test("runBacktest is deterministic for identical params", async () => {
  expect(await mockApi.runBacktest(base)).toEqual(await mockApi.runBacktest(base));
});

test("runBacktest is params-sensitive", async () => {
  const a = await mockApi.runBacktest(base);
  const b = await mockApi.runBacktest({ ...base, strategyId: "safe-graduation", initialCapitalSol: 500 });
  expect(a.metrics.netPnlSol).not.toBe(b.metrics.netPnlSol);
});

test("runBacktest returns full metrics + series + trades", async () => {
  const r = await mockApi.runBacktest(base);
  expect(r.metrics.trades).toBeGreaterThan(0);
  expect(r.equityCurve.length).toBeGreaterThan(0);
  expect(r.drawdown.every((d) => d.v <= 0)).toBe(true);
  expect(r.monthlyReturns.length).toBeGreaterThan(0);
  expect(r.tradeDistribution.length).toBeGreaterThan(0);
  expect(r.pnlByScore.length).toBeGreaterThan(0);
  expect(r.priceSeries.length).toBeGreaterThan(0);
  const times = new Set(r.priceSeries.map((p) => p.t));
  expect(r.trades.some((t) => t.side === "buy")).toBe(true);
  expect(r.trades.some((t) => t.side === "sell")).toBe(true);
  expect(r.trades.every((t) => times.has(t.time))).toBe(true);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/api/backtest.test.ts`
Expected: FAIL — `runBacktest` not a function.

- [ ] **Step 3: Add types**

In `apps/web/lib/api/types.ts`, append (after the Trading Terminal banner). `EquityPoint` already exists in this file:
```ts
// --- Backtesting (Increment 9) ---
export interface BacktestParams {
  strategyId: string; rangePreset: string;
  initialCapitalSol: number; maxPositions: number;
  slippageModel: string; priorityFee: number;
  latencyModel: string; liquidityModel: string;
  minCreatorScore: number; minTokenSafety: number;
}
export interface BacktestMetrics {
  netPnlSol: number; winRatePct: number; profitFactor: number; sharpe: number; sortino: number;
  maxDrawdownPct: number; avgTradeSol: number; rugExposurePct: number; trades: number; avgHoldingHours: number;
}
export interface MonthlyReturn { label: string; pct: number; }
export interface DistributionBucket { label: string; count: number; }
export interface ScorePnl { scoreBucket: string; pnlSol: number; }
export interface DrawdownPoint { t: number; v: number; }
export interface BacktestTrade { time: number; price: number; side: "buy" | "sell"; pnlSol: number; }
export interface BacktestResult {
  metrics: BacktestMetrics;
  equityCurve: EquityPoint[];
  drawdown: DrawdownPoint[];
  monthlyReturns: MonthlyReturn[];
  tradeDistribution: DistributionBucket[];
  pnlByScore: ScorePnl[];
  priceSeries: EquityPoint[];
  trades: BacktestTrade[];
}
```

- [ ] **Step 4: Add contract method**

In `apps/web/lib/api/contract.ts`, add `BacktestParams, BacktestResult` to the `import type { ... } from "./types";` line, then add to `SentinelApi` (after the terminal methods):
```ts
  runBacktest(params: BacktestParams): Promise<BacktestResult>;
```

- [ ] **Step 5: Add http stub**

In `apps/web/lib/api/http.ts`, add to `httpApi`:
```ts
  runBacktest: notReady,
```

- [ ] **Step 6: Add mock builder + adapter**

In `apps/web/lib/api/mock.ts`: add `BacktestParams, BacktestResult, BacktestMetrics, DrawdownPoint, BacktestTrade` to the `import type { ... } from "./types";` line. `seedOf`, `toSeries`, `delay` already exist. Add the builder (module scope):
```ts
const BT_MONTHS = ["Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu"];
const BT_DIST = ["< -5", "-5..0", "0..5", "5..10", "> 10"];
const BT_SCORE_BUCKETS = ["0-24", "25-49", "50-69", "70-84", "85-100"];

function runBacktestResult(p: BacktestParams): BacktestResult {
  const seed = seedOf(
    p.strategyId + p.rangePreset + p.initialCapitalSol + p.maxPositions +
    p.slippageModel + p.latencyModel + p.liquidityModel + p.minCreatorScore + p.minTokenSafety
  );
  const trades = 40 + (seed % 300);
  const netPnl = (seed % 200) - 60 + p.initialCapitalSol * 0.1;
  const r2 = (n: number) => Math.round(n * 100) / 100;
  const metrics: BacktestMetrics = {
    netPnlSol: Math.round(netPnl * 10) / 10,
    winRatePct: 40 + (seed % 45),
    profitFactor: r2(1 + (seed % 250) / 100),
    sharpe: r2((seed % 300) / 100),
    sortino: r2((seed % 350) / 100),
    maxDrawdownPct: 5 + (seed % 35),
    avgTradeSol: r2(netPnl / Math.max(1, trades)),
    rugExposurePct: seed % 10,
    trades,
    avgHoldingHours: 1 + (seed % 12),
  };
  const equityCurve = toSeries(seed, 30);
  const priceSeries = toSeries(seed + 7, 40);
  const drawdown: DrawdownPoint[] = equityCurve.map((pt, i) => ({ t: pt.t, v: -(((seed + i * 3) % 30)) }));
  const monthlyReturns = BT_MONTHS.map((m, i) => ({ label: m, pct: ((seed + i * 13) % 40) - 15 }));
  const tradeDistribution = BT_DIST.map((b, i) => ({ label: b, count: 3 + ((seed + i * 7) % 25) }));
  const pnlByScore = BT_SCORE_BUCKETS.map((b, i) => ({ scoreBucket: b, pnlSol: ((seed + i * 11) % 60) - 20 }));
  const btTrades: BacktestTrade[] = [];
  priceSeries.forEach((pt, i) => {
    if (i % 6 === 2) btTrades.push({ time: pt.t, price: pt.v, side: btTrades.length % 2 === 0 ? "buy" : "sell", pnlSol: ((seed + i) % 20) - 8 });
  });
  return { metrics, equityCurve, drawdown, monthlyReturns, tradeDistribution, pnlByScore, priceSeries, trades: btTrades };
}
```
Add adapter to `mockApi`:
```ts
  runBacktest: (params) => delay(runBacktestResult(params)),
```

- [ ] **Step 7: Run adapter test to verify it passes**

Run: `cd apps/web && npx vitest run lib/api/backtest.test.ts`
Expected: PASS (3 tests). (Note: `priceSeries` length 40 → `i % 6 === 2` yields indices 2,8,14,20,26,32,38 = 7 trades, alternating buy/sell, so ≥1 of each; all times come from priceSeries.)

- [ ] **Step 8: Add query key + hook + hook test**

In `apps/web/lib/get-query-client.ts`, add `BacktestParams` to the type imports if the file imports types (it does not currently import from types; inline the param type). Add to `qk`:
```ts
  backtest: (params: import("./api/types").BacktestParams) => ["backtest", JSON.stringify(params)] as const,
```

In `apps/web/lib/hooks/queries.ts`, append:
```ts
import type { BacktestParams } from "@/lib/api/types";

export function useBacktest(params: BacktestParams | null) {
  return useQuery({
    queryKey: params ? qk.backtest(params) : ["backtest", "idle"],
    queryFn: () => getApi().runBacktest(params as BacktestParams),
    enabled: !!params,
  });
}
```
(Place the `import type` with the other imports at the top of the file, not mid-file.)

Create `apps/web/lib/hooks/backtest-hooks.test.tsx`:
```tsx
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useBacktest } from "./queries";
import type { BacktestParams } from "@/lib/api/types";

const params: BacktestParams = {
  strategyId: "momentum-scalp", rangePreset: "30g", initialCapitalSol: 100, maxPositions: 5,
  slippageModel: "dynamic", priorityFee: 0.0001, latencyModel: "realistic", liquidityModel: "constrained",
  minCreatorScore: 60, minTokenSafety: 55,
};
function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("useBacktest does not fetch when params is null", () => {
  const { result } = renderHook(() => useBacktest(null), { wrapper });
  expect(result.current.fetchStatus).toBe("idle");
  expect(result.current.data).toBeUndefined();
});

test("useBacktest resolves a result when params are provided", async () => {
  const { result } = renderHook(() => useBacktest(params), { wrapper });
  await waitFor(() => expect(result.current.data?.metrics.trades).toBeGreaterThan(0));
});
```

- [ ] **Step 9: Run hook test to verify it passes**

Run: `cd apps/web && npx vitest run lib/hooks/backtest-hooks.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 10: Commit**

```bash
git add apps/web/lib/api/types.ts apps/web/lib/api/contract.ts apps/web/lib/api/http.ts apps/web/lib/api/mock.ts apps/web/lib/get-query-client.ts apps/web/lib/hooks/queries.ts apps/web/lib/api/backtest.test.ts apps/web/lib/hooks/backtest-hooks.test.tsx
git commit -m "feat(backtest): data seam for runBacktest (deterministic, params-sensitive)"
```

---

## Task 2: Backtest config + pure validateParams

**Files:**
- Create: `apps/web/lib/backtest/backtest-defs.ts`, `apps/web/lib/backtest/validate.ts`
- Test: `apps/web/lib/backtest/validate.test.ts`

**Interfaces:**
- Produces: `RANGE_PRESETS`, `SLIPPAGE_MODELS`, `LATENCY_MODELS`, `LIQUIDITY_MODELS`, `DEFAULT_BACKTEST_PARAMS`, `BACKTEST_METRIC_DEFS` (backtest-defs); `validateParams(p)` (validate).
- Consumes: `BacktestParams`, `BacktestMetrics` from `@/lib/api/types`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/backtest/validate.test.ts`:
```ts
import { validateParams } from "./validate";
import { DEFAULT_BACKTEST_PARAMS } from "./backtest-defs";

test("default params validate cleanly", () => {
  expect(validateParams(DEFAULT_BACKTEST_PARAMS)).toEqual({});
});

test("invalid capital / positions / score are rejected", () => {
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, initialCapitalSol: 0 }).initialCapitalSol).toBeTruthy();
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, maxPositions: 0 }).maxPositions).toBeTruthy();
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, minCreatorScore: 120 }).minCreatorScore).toBeTruthy();
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, strategyId: "" }).strategyId).toBeTruthy();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/backtest/validate.test.ts`
Expected: FAIL — modules not found.

- [ ] **Step 3: Write backtest-defs**

Create `apps/web/lib/backtest/backtest-defs.ts`:
```ts
import type { BacktestParams, BacktestMetrics } from "@/lib/api/types";

export const RANGE_PRESETS: { key: string; label: string }[] = [
  { key: "7g", label: "Son 7 gün" }, { key: "30g", label: "Son 30 gün" },
  { key: "90g", label: "Son 90 gün" }, { key: "1y", label: "Son 1 yıl" },
];
export const SLIPPAGE_MODELS: { key: string; label: string }[] = [
  { key: "fixed", label: "Sabit" }, { key: "dynamic", label: "Dinamik" }, { key: "pessimistic", label: "Kötümser" },
];
export const LATENCY_MODELS: { key: string; label: string }[] = [
  { key: "low", label: "Düşük" }, { key: "realistic", label: "Gerçekçi" }, { key: "high", label: "Yüksek" },
];
export const LIQUIDITY_MODELS: { key: string; label: string }[] = [
  { key: "unconstrained", label: "Kısıtsız" }, { key: "constrained", label: "Likidite-oranlı" },
];

export const DEFAULT_BACKTEST_PARAMS: BacktestParams = {
  strategyId: "momentum-scalp", rangePreset: "30g", initialCapitalSol: 100, maxPositions: 5,
  slippageModel: "dynamic", priorityFee: 0.0001, latencyModel: "realistic", liquidityModel: "constrained",
  minCreatorScore: 60, minTokenSafety: 55,
};

export const BACKTEST_METRIC_DEFS: { key: keyof BacktestMetrics; label: string; kind: "pnl" | "pct" | "num" }[] = [
  { key: "netPnlSol", label: "Net PnL (SOL)", kind: "pnl" },
  { key: "winRatePct", label: "Kazanma Oranı", kind: "pct" },
  { key: "profitFactor", label: "Profit Factor", kind: "num" },
  { key: "sharpe", label: "Sharpe", kind: "num" },
  { key: "sortino", label: "Sortino", kind: "num" },
  { key: "maxDrawdownPct", label: "Maks. Drawdown", kind: "pct" },
  { key: "avgTradeSol", label: "Ort. İşlem (SOL)", kind: "pnl" },
  { key: "rugExposurePct", label: "Rug Maruziyeti", kind: "pct" },
  { key: "trades", label: "İşlem Sayısı", kind: "num" },
  { key: "avgHoldingHours", label: "Ort. Tutma (saat)", kind: "num" },
];
```

- [ ] **Step 4: Write validate**

Create `apps/web/lib/backtest/validate.ts`:
```ts
import type { BacktestParams } from "@/lib/api/types";

export function validateParams(p: BacktestParams): { [field: string]: string } {
  const e: { [field: string]: string } = {};
  if (!p.strategyId) e.strategyId = "Strateji seç";
  if (!(p.initialCapitalSol > 0)) e.initialCapitalSol = "Sermaye 0'dan büyük olmalı";
  if (!(p.maxPositions >= 1)) e.maxPositions = "En az 1 pozisyon";
  if (p.priorityFee < 0) e.priorityFee = "Öncelik ücreti negatif olamaz";
  if (p.minCreatorScore < 0 || p.minCreatorScore > 100) e.minCreatorScore = "Skor 0–100 arası";
  if (p.minTokenSafety < 0 || p.minTokenSafety > 100) e.minTokenSafety = "Skor 0–100 arası";
  return e;
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd apps/web && npx vitest run lib/backtest/validate.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 6: Commit**

```bash
git add apps/web/lib/backtest/
git commit -m "feat(backtest): param registries + pure validateParams"
```

---

## Task 3: BacktestParamsForm

**Files:**
- Create: `apps/web/components/backtest/BacktestParamsForm.tsx`
- Test: `apps/web/components/backtest/BacktestParamsForm.test.tsx`

**Interfaces:**
- Produces: `BacktestParamsForm({ onRun }: { onRun: (p: BacktestParams) => void })`.
- Consumes: `useStrategies` (`@/lib/hooks/queries`), registries + `DEFAULT_BACKTEST_PARAMS` (`@/lib/backtest/backtest-defs`), `validateParams` (`@/lib/backtest/validate`), `BacktestParams` type.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/backtest/BacktestParamsForm.test.tsx`:
```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { BacktestParamsForm } from "./BacktestParamsForm";

function renderForm(onRun = () => {}) {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <BacktestParamsForm onRun={onRun} />
    </QueryClientProvider>
  );
}

test("renders fields and runs with valid params", async () => {
  const onRun = vi.fn();
  renderForm(onRun);
  await waitFor(() => expect(screen.getByLabelText("Başlangıç Sermayesi (SOL)")).toBeInTheDocument());
  fireEvent.click(screen.getByRole("button", { name: "Çalıştır" }));
  expect(onRun).toHaveBeenCalledTimes(1);
});

test("invalid capital blocks run and shows error", async () => {
  const onRun = vi.fn();
  renderForm(onRun);
  const capital = await screen.findByLabelText("Başlangıç Sermayesi (SOL)");
  fireEvent.change(capital, { target: { value: "0" } });
  fireEvent.click(screen.getByRole("button", { name: "Çalıştır" }));
  expect(screen.getByText("Sermaye 0'dan büyük olmalı")).toBeInTheDocument();
  expect(onRun).not.toHaveBeenCalled();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/backtest/BacktestParamsForm.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/backtest/BacktestParamsForm.tsx`:
```tsx
"use client";
import { useState } from "react";
import { useStrategies } from "@/lib/hooks/queries";
import { validateParams } from "@/lib/backtest/validate";
import {
  DEFAULT_BACKTEST_PARAMS, RANGE_PRESETS, SLIPPAGE_MODELS, LATENCY_MODELS, LIQUIDITY_MODELS,
} from "@/lib/backtest/backtest-defs";
import type { BacktestParams } from "@/lib/api/types";

const selectCls = "h-8 w-full rounded-md border border-border bg-input px-2 text-foreground focus:outline-none";
const inputCls = "h-8 w-full rounded-md border border-border bg-input px-2 text-foreground focus:outline-none";

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1" style={{ fontSize: 12 }}>
      <span className="text-muted-foreground">{label}</span>
      {children}
    </label>
  );
}

export function BacktestParamsForm({ onRun }: { onRun: (p: BacktestParams) => void }) {
  const { data: strategies } = useStrategies();
  const [p, setP] = useState<BacktestParams>(DEFAULT_BACKTEST_PARAMS);
  const [errors, setErrors] = useState<{ [f: string]: string }>({});
  const set = (patch: Partial<BacktestParams>) => setP((prev) => ({ ...prev, ...patch }));

  const run = () => {
    const e = validateParams(p);
    setErrors(e);
    if (Object.keys(e).length === 0) onRun(p);
  };

  const Err = ({ f }: { f: string }) => (errors[f] ? <span style={{ fontSize: 11, color: "#F0476B" }}>{errors[f]}</span> : null);

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border bg-card p-3">
      <div className="font-medium" style={{ fontSize: 13 }}>Parametreler</div>

      <Field label="Strateji">
        <select aria-label="Strateji" className={selectCls} style={{ fontSize: 12 }} value={p.strategyId} onChange={(e) => set({ strategyId: e.target.value })}>
          {(strategies ?? []).map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>
      </Field>
      <Err f="strategyId" />

      <Field label="Tarih Aralığı">
        <select aria-label="Tarih Aralığı" className={selectCls} style={{ fontSize: 12 }} value={p.rangePreset} onChange={(e) => set({ rangePreset: e.target.value })}>
          {RANGE_PRESETS.map((r) => <option key={r.key} value={r.key}>{r.label}</option>)}
        </select>
      </Field>

      <Field label="Başlangıç Sermayesi (SOL)">
        <input aria-label="Başlangıç Sermayesi (SOL)" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.initialCapitalSol} onChange={(e) => set({ initialCapitalSol: Number(e.target.value) })} />
      </Field>
      <Err f="initialCapitalSol" />

      <Field label="Maks. Pozisyon">
        <input aria-label="Maks. Pozisyon" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.maxPositions} onChange={(e) => set({ maxPositions: Number(e.target.value) })} />
      </Field>
      <Err f="maxPositions" />

      <Field label="Slippage Modeli">
        <select aria-label="Slippage Modeli" className={selectCls} style={{ fontSize: 12 }} value={p.slippageModel} onChange={(e) => set({ slippageModel: e.target.value })}>
          {SLIPPAGE_MODELS.map((m) => <option key={m.key} value={m.key}>{m.label}</option>)}
        </select>
      </Field>

      <Field label="Öncelik Ücreti (SOL)">
        <input aria-label="Öncelik Ücreti (SOL)" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.priorityFee} onChange={(e) => set({ priorityFee: Number(e.target.value) })} />
      </Field>
      <Err f="priorityFee" />

      <Field label="Gecikme Modeli">
        <select aria-label="Gecikme Modeli" className={selectCls} style={{ fontSize: 12 }} value={p.latencyModel} onChange={(e) => set({ latencyModel: e.target.value })}>
          {LATENCY_MODELS.map((m) => <option key={m.key} value={m.key}>{m.label}</option>)}
        </select>
      </Field>

      <Field label="Likidite Modeli">
        <select aria-label="Likidite Modeli" className={selectCls} style={{ fontSize: 12 }} value={p.liquidityModel} onChange={(e) => set({ liquidityModel: e.target.value })}>
          {LIQUIDITY_MODELS.map((m) => <option key={m.key} value={m.key}>{m.label}</option>)}
        </select>
      </Field>

      <Field label="Min. Creator Skoru">
        <input aria-label="Min. Creator Skoru" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.minCreatorScore} onChange={(e) => set({ minCreatorScore: Number(e.target.value) })} />
      </Field>
      <Err f="minCreatorScore" />

      <Field label="Min. Token Güvenliği">
        <input aria-label="Min. Token Güvenliği" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.minTokenSafety} onChange={(e) => set({ minTokenSafety: Number(e.target.value) })} />
      </Field>
      <Err f="minTokenSafety" />

      <button onClick={run} className="mt-1 rounded-md px-3 py-2" style={{ fontSize: 13, fontWeight: 600, backgroundColor: "#2FD98B", color: "#08210F" }}>
        Çalıştır
      </button>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/backtest/BacktestParamsForm.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/backtest/BacktestParamsForm.tsx apps/web/components/backtest/BacktestParamsForm.test.tsx
git commit -m "feat(backtest): BacktestParamsForm with validation + run"
```

---

## Task 4: BacktestMetrics

**Files:**
- Create: `apps/web/components/backtest/BacktestMetrics.tsx`
- Test: `apps/web/components/backtest/BacktestMetrics.test.tsx`

**Interfaces:**
- Produces: `BacktestMetrics({ metrics }: { metrics: BacktestMetrics })`.
- Consumes: `MetricTile` (`@/components/sentinel/MetricTile`), `pnlColor` (`@/lib/position/risk-filter`), `BACKTEST_METRIC_DEFS` (`@/lib/backtest/backtest-defs`), `BacktestMetrics` type (import aliased to avoid name clash with the component).

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/backtest/BacktestMetrics.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { BacktestMetrics } from "./BacktestMetrics";
import type { BacktestMetrics as Metrics } from "@/lib/api/types";

const m: Metrics = {
  netPnlSol: 128.4, winRatePct: 62, profitFactor: 2.1, sharpe: 1.8, sortino: 2.3,
  maxDrawdownPct: 14, avgTradeSol: 0.58, rugExposurePct: 3, trades: 220, avgHoldingHours: 6,
};

test("renders all 10 metric labels", () => {
  render(<BacktestMetrics metrics={m} />);
  expect(screen.getByText("Net PnL (SOL)")).toBeInTheDocument();
  expect(screen.getByText("Sortino")).toBeInTheDocument();
  expect(screen.getByText("İşlem Sayısı")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/backtest/BacktestMetrics.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/backtest/BacktestMetrics.tsx`:
```tsx
import { MetricTile } from "@/components/sentinel/MetricTile";
import { pnlColor } from "@/lib/position/risk-filter";
import { BACKTEST_METRIC_DEFS } from "@/lib/backtest/backtest-defs";
import type { BacktestMetrics as Metrics } from "@/lib/api/types";

function fmt(kind: "pnl" | "pct" | "num", v: number): string {
  if (kind === "pct") return `%${v}`;
  if (kind === "pnl") return `${v} SOL`;
  return `${v}`;
}

export function BacktestMetrics({ metrics }: { metrics: Metrics }) {
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-5">
      {BACKTEST_METRIC_DEFS.map((d) => {
        const v = metrics[d.key];
        return <MetricTile key={d.key} label={d.label} value={fmt(d.kind, v)} valueColor={d.kind === "pnl" ? pnlColor(v) : undefined} />;
      })}
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/backtest/BacktestMetrics.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/backtest/BacktestMetrics.tsx apps/web/components/backtest/BacktestMetrics.test.tsx
git commit -m "feat(backtest): BacktestMetrics 10-tile grid"
```

---

## Task 5: Simple charts (Drawdown / MonthlyReturn / TradeDistribution / PnlByScore)

**Files:**
- Create: `apps/web/components/backtest/DrawdownChart.tsx`, `MonthlyReturnChart.tsx`, `TradeDistributionChart.tsx`, `PnlByScoreChart.tsx`
- Test: `apps/web/components/backtest/BacktestCharts.test.tsx`

**Interfaces:**
- Produces: `DrawdownChart({ data: DrawdownPoint[] })`, `MonthlyReturnChart({ data: MonthlyReturn[] })`, `TradeDistributionChart({ data: DistributionBucket[] })`, `PnlByScoreChart({ data: ScorePnl[] })`.
- Consumes: recharts, `pnlColor`, the respective types.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/backtest/BacktestCharts.test.tsx`:
```tsx
import { render } from "@testing-library/react";
import { DrawdownChart } from "./DrawdownChart";
import { MonthlyReturnChart } from "./MonthlyReturnChart";
import { TradeDistributionChart } from "./TradeDistributionChart";
import { PnlByScoreChart } from "./PnlByScoreChart";

const wrap = (ui: React.ReactNode) => render(<div style={{ width: 400, height: 240 }}>{ui}</div>);

test("drawdown chart renders (smoke)", () => {
  const { container } = wrap(<DrawdownChart data={[{ t: 0, v: 0 }, { t: 1, v: -5 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
test("monthly return chart renders (smoke)", () => {
  const { container } = wrap(<MonthlyReturnChart data={[{ label: "Oca", pct: 8 }, { label: "Şub", pct: -3 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
test("trade distribution chart renders (smoke)", () => {
  const { container } = wrap(<TradeDistributionChart data={[{ label: "0..5", count: 12 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
test("pnl-by-score chart renders (smoke)", () => {
  const { container } = wrap(<PnlByScoreChart data={[{ scoreBucket: "70-84", pnlSol: 20 }, { scoreBucket: "0-24", pnlSol: -10 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/backtest/BacktestCharts.test.tsx`
Expected: FAIL — components not found.

- [ ] **Step 3: Write DrawdownChart**

Create `apps/web/components/backtest/DrawdownChart.tsx`:
```tsx
"use client";
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { DrawdownPoint } from "@/lib/api/types";

export function DrawdownChart({ data }: { data: DrawdownPoint[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Drawdown (%)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="bt-dd-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#F0476B" stopOpacity={0.05} />
                <stop offset="100%" stopColor="#F0476B" stopOpacity={0.35} />
              </linearGradient>
            </defs>
            <XAxis dataKey="t" hide />
            <YAxis hide domain={["dataMin", 0]} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} labelFormatter={() => ""} formatter={(v) => [`%${Number(v ?? 0)}`, "Drawdown"]} />
            <Area type="monotone" dataKey="v" stroke="#F0476B" strokeWidth={1.5} fill="url(#bt-dd-grad)" />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Write MonthlyReturnChart**

Create `apps/web/components/backtest/MonthlyReturnChart.tsx`:
```tsx
"use client";
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip, Cell } from "recharts";
import type { MonthlyReturn } from "@/lib/api/types";
import { pnlColor } from "@/lib/position/risk-filter";

export function MonthlyReturnChart({ data }: { data: MonthlyReturn[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Aylık Getiri (%)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis dataKey="label" tick={{ fill: "#8A94A6", fontSize: 11 }} />
            <YAxis hide />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} cursor={{ fill: "rgba(255,255,255,0.04)" }} formatter={(v) => [`%${Number(v ?? 0)}`, "Getiri"]} />
            <Bar dataKey="pct" name="Getiri" radius={[4, 4, 0, 0]}>
              {data.map((d) => <Cell key={d.label} fill={pnlColor(d.pct)} />)}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Write TradeDistributionChart**

Create `apps/web/components/backtest/TradeDistributionChart.tsx`:
```tsx
"use client";
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { DistributionBucket } from "@/lib/api/types";

export function TradeDistributionChart({ data }: { data: DistributionBucket[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Trade Dağılımı (PnL kovaları)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis dataKey="label" tick={{ fill: "#8A94A6", fontSize: 11 }} />
            <YAxis hide />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} cursor={{ fill: "rgba(255,255,255,0.04)" }} formatter={(v) => [`${Number(v ?? 0)}`, "İşlem"]} />
            <Bar dataKey="count" name="İşlem" fill="#3E9BFF" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: Write PnlByScoreChart**

Create `apps/web/components/backtest/PnlByScoreChart.tsx`:
```tsx
"use client";
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip, Cell } from "recharts";
import type { ScorePnl } from "@/lib/api/types";
import { pnlColor } from "@/lib/position/risk-filter";

export function PnlByScoreChart({ data }: { data: ScorePnl[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Skora Göre PnL (SOL)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis dataKey="scoreBucket" tick={{ fill: "#8A94A6", fontSize: 11 }} />
            <YAxis hide />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} cursor={{ fill: "rgba(255,255,255,0.04)" }} formatter={(v) => [`${Number(v ?? 0)}`, "PnL (SOL)"]} />
            <Bar dataKey="pnlSol" name="PnL (SOL)" radius={[4, 4, 0, 0]}>
              {data.map((d) => <Cell key={d.scoreBucket} fill={pnlColor(d.pnlSol)} />)}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 7: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/backtest/BacktestCharts.test.tsx`
Expected: PASS (4 tests).

- [ ] **Step 8: Commit**

```bash
git add apps/web/components/backtest/DrawdownChart.tsx apps/web/components/backtest/MonthlyReturnChart.tsx apps/web/components/backtest/TradeDistributionChart.tsx apps/web/components/backtest/PnlByScoreChart.tsx apps/web/components/backtest/BacktestCharts.test.tsx
git commit -m "feat(backtest): drawdown/monthly/distribution/pnl-by-score charts"
```

---

## Task 6: EntryExitChart

**Files:**
- Create: `apps/web/components/backtest/EntryExitChart.tsx`
- Test: `apps/web/components/backtest/EntryExitChart.test.tsx`

**Interfaces:**
- Produces: `EntryExitChart({ priceSeries, trades }: { priceSeries: EquityPoint[]; trades: BacktestTrade[] })`.
- Consumes: recharts (`ComposedChart, Line, Scatter, XAxis, YAxis, ResponsiveContainer, Tooltip`), `EquityPoint`/`BacktestTrade` types.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/backtest/EntryExitChart.test.tsx`:
```tsx
import { render } from "@testing-library/react";
import { EntryExitChart } from "./EntryExitChart";

test("entry/exit chart renders price + markers (smoke)", () => {
  const { container } = render(
    <div style={{ width: 400, height: 240 }}>
      <EntryExitChart
        priceSeries={[{ t: 0, v: 10 }, { t: 1, v: 12 }, { t: 2, v: 9 }]}
        trades={[{ time: 1, price: 12, side: "buy", pnlSol: 2 }, { time: 2, price: 9, side: "sell", pnlSol: -1 }]}
      />
    </div>
  );
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/backtest/EntryExitChart.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/backtest/EntryExitChart.tsx`:
```tsx
"use client";
import { ComposedChart, Line, Scatter, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { EquityPoint, BacktestTrade } from "@/lib/api/types";

export function EntryExitChart({ priceSeries, trades }: { priceSeries: EquityPoint[]; trades: BacktestTrade[] }) {
  const merged = priceSeries.map((p) => {
    const tr = trades.find((t) => t.time === p.t);
    return {
      t: p.t, v: p.v,
      buy: tr?.side === "buy" ? tr.price : undefined,
      sell: tr?.side === "sell" ? tr.price : undefined,
    };
  });
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Giriş / Çıkış Noktaları</div>
      <div style={{ height: 260 }}>
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={merged} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis dataKey="t" hide />
            <YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} labelFormatter={() => ""} />
            <Line type="monotone" dataKey="v" stroke="#8A94A6" strokeWidth={1.5} dot={false} name="Fiyat" />
            <Scatter dataKey="buy" fill="#2FD98B" name="Al" />
            <Scatter dataKey="sell" fill="#F0476B" name="Sat" />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/backtest/EntryExitChart.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/backtest/EntryExitChart.tsx apps/web/components/backtest/EntryExitChart.test.tsx
git commit -m "feat(backtest): EntryExitChart (price line + buy/sell scatter)"
```

---

## Task 7: BacktestContent (composition + submittedParams)

**Files:**
- Create: `apps/web/components/backtest/BacktestContent.tsx`
- Test: `apps/web/components/backtest/BacktestContent.test.tsx`

**Interfaces:**
- Produces: `BacktestContent()`.
- Consumes: `useBacktest` (`@/lib/hooks/queries`), `BacktestParamsForm`, `BacktestMetrics`, `EquityCurve` (`@/components/sentinel/EquityCurve`), `DrawdownChart`, `MonthlyReturnChart`, `TradeDistributionChart`, `PnlByScoreChart`, `EntryExitChart`, `Skeleton`, `BacktestParams` type.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/backtest/BacktestContent.test.tsx`:
```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { BacktestContent } from "./BacktestContent";

function renderContent() {
  return render(<QueryClientProvider client={getQueryClient()}><BacktestContent /></QueryClientProvider>);
}

test("shows empty state before a run, then results after Çalıştır", async () => {
  renderContent();
  expect(screen.getByRole("heading", { name: "Geriye Test" })).toBeInTheDocument();
  expect(screen.getByText("Parametreleri seç ve Çalıştır'a bas")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByLabelText("Strateji")).toBeInTheDocument());
  fireEvent.click(screen.getByRole("button", { name: "Çalıştır" }));
  await waitFor(() => expect(screen.getByText("Net PnL (SOL)")).toBeInTheDocument());
  expect(screen.getByText("Sermaye Eğrisi")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/backtest/BacktestContent.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/backtest/BacktestContent.tsx`:
```tsx
"use client";
import { useState } from "react";
import { useBacktest } from "@/lib/hooks/queries";
import { BacktestParamsForm } from "./BacktestParamsForm";
import { BacktestMetrics } from "./BacktestMetrics";
import { EquityCurve } from "@/components/sentinel/EquityCurve";
import { DrawdownChart } from "./DrawdownChart";
import { MonthlyReturnChart } from "./MonthlyReturnChart";
import { TradeDistributionChart } from "./TradeDistributionChart";
import { PnlByScoreChart } from "./PnlByScoreChart";
import { EntryExitChart } from "./EntryExitChart";
import { Skeleton } from "@/components/ui/skeleton";
import type { BacktestParams } from "@/lib/api/types";

function EmptyState() {
  return (
    <div className="flex h-full items-center justify-center rounded-lg border border-dashed border-border bg-card py-24 text-center text-muted-foreground" style={{ fontSize: 13 }}>
      Parametreleri seç ve Çalıştır'a bas
    </div>
  );
}

export function BacktestContent() {
  const [params, setParams] = useState<BacktestParams | null>(null);
  const { data, isLoading, isError } = useBacktest(params);

  return (
    <div className="flex flex-col gap-3">
      <h1>Geriye Test</h1>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-[300px_1fr]">
        <BacktestParamsForm onRun={setParams} />
        <div>
          {!params && <EmptyState />}
          {params && isLoading && <Skeleton className="h-96 w-full" />}
          {params && isError && (
            <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Backtest çalıştırılamadı.</div>
          )}
          {params && data && (
            <div className="flex flex-col gap-3">
              <BacktestMetrics metrics={data.metrics} />
              <EquityCurve data={data.equityCurve} title="Sermaye Eğrisi" />
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                <DrawdownChart data={data.drawdown} />
                <MonthlyReturnChart data={data.monthlyReturns} />
                <TradeDistributionChart data={data.tradeDistribution} />
                <PnlByScoreChart data={data.pnlByScore} />
              </div>
              <EntryExitChart priceSeries={data.priceSeries} trades={data.trades} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/backtest/BacktestContent.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/backtest/BacktestContent.tsx apps/web/components/backtest/BacktestContent.test.tsx
git commit -m "feat(backtest): BacktestContent composition with empty/loading/result states"
```

---

## Task 8: /backtesting page (RSC prefetch) + full test + build

**Files:**
- Modify: `apps/web/app/(app)/backtesting/page.tsx`

**Interfaces:**
- Consumes: `getApi`, `qk`, `getQueryClient`, `HydrationBoundary`/`dehydrate`, `BacktestContent`.

- [ ] **Step 1: Replace the placeholder page**

Overwrite `apps/web/app/(app)/backtesting/page.tsx` (currently `PlaceholderScreen`) with:
```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { BacktestContent } from "@/components/backtest/BacktestContent";

export default async function BacktestingPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.strategies, queryFn: () => getApi().getStrategies() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <BacktestContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 2: Run the full suite**

Run: `cd apps/web && npx vitest run`
Expected: PASS — all prior suites + new backtest suites green.

- [ ] **Step 3: Run the production build**

Run: `cd apps/web && npm run build`
Expected: SUCCESS — `/backtesting` appears as a route. If `next build`'s tsc surfaces a type error vitest missed (e.g. a Recharts `Formatter`/`ComposedChart` typing mismatch), fix it type-correctly and re-run.

- [ ] **Step 4: Commit**

```bash
git add apps/web/app/(app)/backtesting/page.tsx
git commit -m "feat(backtest): wire /backtesting page with RSC prefetch (getStrategies)"
```

---

## Task 9: Living docs + visual verification

**Files:**
- Modify: `docs/progress.md`, `docs/superpowers/specs/2026-08-03-sentinel-backtesting-design.md`, `docs/superpowers/followups-frontend.md`
- Modify (memory): `sentinel-frontend-stack-and-plan.md`

- [ ] **Step 1: Visual check**

Run `cd apps/web && npm run dev`, open `/backtesting`. Verify: left params form (strategy dropdown populated from strategies, range/model selects, capital/positions/fee/score inputs); empty state on the right ("Parametreleri seç..."); click "Çalıştır" → 10 metric tiles (Net PnL green/red) + Sermaye Eğrisi + Drawdown + Aylık Getiri + Trade Dağılımı + Skora Göre PnL + Giriş/Çıkış (price line with green/red markers); change strategy/capital and re-run → numbers change; set capital 0 → error under the field + no run. If the browser extension is unavailable, note it and proceed with docs.

- [ ] **Step 2: Update living docs**

- `docs/progress.md`: set Increment 9 row to ✅ (branch complete, merge awaiting); add a decision-log entry dated 2026-08-03 (route, simulated deterministic-seeded run, 10 metrics + 6 charts, EquityCurve/MetricTile/useStrategies reuse, Event Replay deferred); update "Sırada" to Increment 10 (Alerts / Telegram); bump "Son güncelleme".
- Spec `Durum:` → `Uygulandı (2026-08-03) — branch ..., N/N test, build + review temiz; merge onayı bekliyor.`
- `docs/superpowers/followups-frontend.md`: add a "Backtesting (Increment 9)" section for deferred minors surfaced during review (e.g. Event Replay deferred, missed-opportunity chart out, params-form has no aria-describedby, native select styling).
- Memory `sentinel-frontend-stack-and-plan.md`: add the Increment 9 entry + update completed-screens + next-increments line.

- [ ] **Step 3: Commit docs**

```bash
git add docs/progress.md docs/superpowers/specs/2026-08-03-sentinel-backtesting-design.md docs/superpowers/followups-frontend.md
git commit -m "docs(backtest): mark Increment 9 complete + followups"
```

---

## Self-Review (completed during authoring)

**Spec coverage:** §1 route/layout/run → Tasks 7–8; §3.1 seam → Task 1; §3.2 config+validate → Task 2; §3.3 components → Tasks 3–7; §3.4 page → Task 8; §5 tests → each task's TDD steps; §6 acceptance → Tasks 3–8 + Task 9 visual. Event Replay correctly absent (deferred). All spec sections map to a task.

**Placeholder scan:** No TBD/TODO; every code step contains real code; test bodies are concrete.

**Type consistency:** `BacktestParams`/`BacktestResult`/`BacktestMetrics`/`DrawdownPoint`/`MonthlyReturn`/`DistributionBucket`/`ScorePnl`/`BacktestTrade` names identical across Tasks 1–7. `runBacktest`/`useBacktest` signatures match between Task 1 (definition) and Tasks 7 (consumer). `BACKTEST_METRIC_DEFS` keys are `keyof BacktestMetrics` (Task 2) consumed in Task 4. `DEFAULT_BACKTEST_PARAMS.strategyId = "momentum-scalp"` is a real `STRATEGY_DEFS` id (mock.ts) so the default dropdown selection resolves. `qk.backtest` key (Task 1) matches the page's use of `qk.strategies` for prefetch (Task 8 prefetches strategies, not backtest — backtest is on-demand).

**Known integration risks (flagged for implementers):**
- The `BacktestMetrics` component and the `BacktestMetrics` type share a name — Task 4 imports the type `as Metrics` to avoid the clash. Enforced in the code.
- `useBacktest(null)` must not fetch — `enabled: !!params` + an "idle" placeholder key (Task 1). Locked by a hook test.
- Recharts `ComposedChart` + `Scatter` typing may trip `next build` tsc even when vitest passes (see Increment 7/8 precedent) — Task 8 Step 3 handles it explicitly.
- The entry/exit overlay relies on `trades[].time` values existing in `priceSeries[].t` — guaranteed by the mock builder (Task 1) which draws trade times from priceSeries points; the seam test asserts this invariant.
