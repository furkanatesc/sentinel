# SENTINEL Frontend Increment 8 — Trading Terminal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the full Screen-7 Trading Terminal at `/terminal` — a 4-quadrant read-only terminal (token watchlist · market data + candlestick chart · order panel · bottom tabs) driven by the mock seam, with fully simulated stateless order submission.

**Architecture:** Server components RSC-prefetch the mock seam and hydrate a client `TerminalContent` that owns `activeMint` state. Leaf panels read data via `component → hook → getApi() → mock` (DIP; no component imports mock). The price chart uses `lightweight-charts` behind a `next/dynamic ssr:false` boundary (Cytoscape pattern). Order flow is fully simulated: preview → confirmation dialog → `toast` (live→`toast.warning`, paper/shadow→simulate). Nothing persists.

**Tech Stack:** Next.js 16 App Router, TypeScript, TanStack Query, Zustand (`tradingMode`), Tailwind v4, shadcn/Base-UI (Tabs + new Dialog), lightweight-charts v4 (candlestick), sonner (toast), Vitest + React Testing Library.

## Global Constraints

- UI language **Turkish** for all visible text; technical tokens/symbols exempt. Dark-only theme.
- **DIP:** no component imports `@/lib/api/mock`; all data flows through hooks → `getApi()`. `httpApi` methods → `notReady` (reject).
- **Clean code + SOLID** is a review criterion (SRP/OCP/DIP/ISP; config-driven registries; small focused files).
- Monorepo frontend root: `apps/web/`. Alias `@/` → `apps/web/`. No `src/`.
- Order actions are **read-only/simulated** — never a real trade. Live mode → `toast.warning`; paper/shadow → simulate `toast`. Consistent with Increments 2/6/7.
- Vitest: `globals: true` (no need to import `test`/`expect`/`vi` except where a file already does). Setup file `apps/web/test/setup.ts`. Mock 40 ms delay → data-dependent assertions use `await waitFor(...)`.
- Run all commands from `apps/web/` (e.g. `cd apps/web && npx vitest run <path>`).
- New dependency: `lightweight-charts@^4.2.0` (v4 API: `createChart` + `addCandlestickSeries` + `autoSize`). Pin to v4 — v5 renamed the series API.

---

## File Structure

**Create:**
- `apps/web/lib/terminal/order-defs.ts` — registries, `OrderDraft`, defaults, constants (`DEFAULT_TERMINAL_MINT`, `MOCK_WALLET_SOL`).
- `apps/web/lib/terminal/order-logic.ts` — pure `validateOrder`, `simulateOrder`.
- `apps/web/components/ui/dialog.tsx` — shadcn Base-UI Dialog (confirmation modal primitive).
- `apps/web/components/terminal/TokenWatchlistPanel.tsx` — left; token list + selection.
- `apps/web/components/terminal/MarketDataHeader.tsx` — center top; price + stats + score badges.
- `apps/web/components/terminal/PriceChartCanvas.tsx` — lightweight-charts instance (client-only).
- `apps/web/components/terminal/PriceChart.tsx` — `next/dynamic ssr:false` wrapper + `useCandles`.
- `apps/web/components/terminal/OrderPanel.tsx` — order form + simulation status.
- `apps/web/components/terminal/OrderConfirmDialog.tsx` — confirmation modal + simulated submit.
- `apps/web/components/terminal/OrdersTable.tsx` — bottom Orders tab.
- `apps/web/components/terminal/TransactionsTable.tsx` — bottom Transactions tab.
- `apps/web/components/terminal/TradeLogsList.tsx` — bottom Logs tab.
- `apps/web/components/terminal/BottomTabsPanel.tsx` — tabs composing the 4 bottom panels.
- `apps/web/components/terminal/TerminalContent.tsx` — composition + `activeMint` state.
- `apps/web/app/(app)/terminal/page.tsx` — server component RSC prefetch.
- Test files colocated (`*.test.ts`/`*.test.tsx`).

**Modify:**
- `apps/web/lib/api/types.ts` — add Trading Terminal types.
- `apps/web/lib/api/contract.ts` — add 5 methods.
- `apps/web/lib/api/http.ts` — add 5 `notReady` stubs.
- `apps/web/lib/api/mock.ts` — add 5 deterministic adapters + builders.
- `apps/web/lib/get-query-client.ts` — add 5 query keys.
- `apps/web/lib/hooks/queries.ts` — add 5 hooks.
- `apps/web/components/shell/nav.ts` — rename "Emirler"/`/orders` → "Terminal"/`/terminal`.

**Delete:**
- `apps/web/app/(app)/orders/page.tsx` — placeholder replaced by `/terminal`.

---

## Task 1: Data seam (types, contract, http, mock, qk, hooks)

**Files:**
- Modify: `apps/web/lib/api/types.ts`
- Modify: `apps/web/lib/api/contract.ts`
- Modify: `apps/web/lib/api/http.ts`
- Modify: `apps/web/lib/api/mock.ts`
- Modify: `apps/web/lib/get-query-client.ts`
- Modify: `apps/web/lib/hooks/queries.ts`
- Test: `apps/web/lib/api/terminal.test.ts`, `apps/web/lib/hooks/terminal-hooks.test.tsx`

**Interfaces:**
- Produces (types): `Candle`, `MarketData`, `OrderSide`, `OrderType`, `OrderStatus`, `Order`, `Txn`, `TradeLog`.
- Produces (api): `getCandles(mint: string): Promise<Candle[]>`, `getMarketData(mint: string): Promise<MarketData>`, `getOrders(): Promise<Order[]>`, `getTransactions(): Promise<Txn[]>`, `getTradeLogs(): Promise<TradeLog[]>`.
- Produces (qk): `qk.candles(mint)`, `qk.marketData(mint)`, `qk.orders`, `qk.transactions`, `qk.tradeLogs`.
- Produces (hooks): `useCandles(mint)`, `useMarketData(mint)`, `useOrders()`, `useTransactions()`, `useTradeLogs()`.
- Consumes: existing `tokens`/`POSITION_TOKENS` arrays, `seedOf`, `delay` in `mock.ts`; `useTokens` already exists.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/api/terminal.test.ts`:
```ts
import { mockApi } from "./mock";

test("getCandles is deterministic and returns OHLC series for a known mint", async () => {
  const a = await mockApi.getCandles("9xQeWv...4Fk2");
  const b = await mockApi.getCandles("9xQeWv...4Fk2");
  expect(a).toEqual(b);
  expect(a.length).toBeGreaterThanOrEqual(30);
  for (const c of a) {
    expect(c.high).toBeGreaterThanOrEqual(c.low);
    expect(typeof c.time).toBe("number");
  }
});

test("getMarketData resolves a token symbol/mint to price + scores", async () => {
  const m = await mockApi.getMarketData("PULSE");
  expect(m.symbol).toBe("PULSE");
  expect(m.price).toBeGreaterThan(0);
  expect(m.tokenScore).toBeGreaterThanOrEqual(0);
  expect(m.creatorScore).toBeGreaterThanOrEqual(0);
});

test("orders/transactions/logs are non-empty and deterministic", async () => {
  expect(await mockApi.getOrders()).toEqual(await mockApi.getOrders());
  expect((await mockApi.getOrders()).length).toBeGreaterThan(0);
  expect((await mockApi.getTransactions()).length).toBeGreaterThan(0);
  const logs = await mockApi.getTradeLogs();
  expect(logs.length).toBeGreaterThan(0);
  expect(["info", "warn", "error"]).toContain(logs[0].level);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/api/terminal.test.ts`
Expected: FAIL — `getCandles`/`getMarketData`/etc. not a function.

- [ ] **Step 3: Add types**

In `apps/web/lib/api/types.ts`, append (after the last increment banner):
```ts
// --- Trading Terminal (Increment 8) ---
export interface Candle { time: number; open: number; high: number; low: number; close: number; }
export interface MarketData {
  mint: string; symbol: string;
  price: number; change24hPct: number;
  liquiditySol: number; volume24hSol: number; marketCapSol: number;
  tokenScore: number; creatorScore: number;
}
export type OrderSide = "buy" | "sell";
export type OrderType = "market" | "limit";
export type OrderStatus = "open" | "filled" | "cancelled";
export interface Order {
  id: string; tokenSymbol: string; tokenMint: string;
  side: OrderSide; type: OrderType; status: OrderStatus;
  price: number; amountSol: number; createdAt: string;
}
export interface Txn {
  id: string; hash: string; kind: "buy" | "sell" | "approve";
  tokenSymbol: string; amountSol: number; status: "success" | "pending" | "failed"; time: string;
}
export interface TradeLog { id: string; level: "info" | "warn" | "error"; message: string; time: string; }
```

- [ ] **Step 4: Add contract methods**

In `apps/web/lib/api/contract.ts`, add the imports to the existing `import type { ... } from "./types";` line: `Candle, MarketData, Order, Txn, TradeLog`. Then inside `SentinelApi`, after `getPositions()`:
```ts
  getCandles(mint: string): Promise<Candle[]>;
  getMarketData(mint: string): Promise<MarketData>;
  getOrders(): Promise<Order[]>;
  getTransactions(): Promise<Txn[]>;
  getTradeLogs(): Promise<TradeLog[]>;
```

- [ ] **Step 5: Add http stubs**

In `apps/web/lib/api/http.ts`, add to the `httpApi` object (with the other read methods):
```ts
  getCandles: notReady,
  getMarketData: notReady,
  getOrders: notReady,
  getTransactions: notReady,
  getTradeLogs: notReady,
```

- [ ] **Step 6: Add mock builders + adapters**

In `apps/web/lib/api/mock.ts`:

Add `Candle, MarketData, Order, OrderStatus, Txn, TradeLog` to the big `import type { ... } from "./types";` line.

Add builders (module scope, near the other builders). `tokens`, `POSITION_TOKENS`, `seedOf`, `delay` already exist:
```ts
const px = (n: number) => Math.round(n * 1e6) / 1e6;

function buildCandles(mint: string): Candle[] {
  const seed = seedOf(mint);
  const out: Candle[] = [];
  let base = 0.001 + (seed % 60) / 20000;
  for (let i = 0; i < 60; i++) {
    const open = base;
    const drift = Math.sin(seed + i * 0.7) * base * 0.05 + (((seed * (i + 1)) % 7) - 3) * base * 0.01;
    const close = Math.max(base * 0.4, open + drift);
    const high = Math.max(open, close) * (1 + ((seed + i) % 4) / 100);
    const low = Math.min(open, close) * (1 - ((seed + i) % 3) / 100);
    out.push({ time: 1_700_000_000 + i * 300, open: px(open), high: px(high), low: px(low), close: px(close) });
    base = close;
  }
  return out;
}

function buildMarketData(mint: string): MarketData {
  const q = mint.toLowerCase();
  const row = tokens.find((t) => t.mint.toLowerCase() === q || t.symbol.toLowerCase() === q || t.id.toLowerCase() === q) ?? tokens[0];
  const seed = seedOf(row.mint);
  return {
    mint: row.mint, symbol: row.symbol,
    price: row.price, change24hPct: (seed % 40) - 15,
    liquiditySol: Math.round(row.liquidity / 100),
    volume24hSol: Math.round((row.vol5m * 12) / 100),
    marketCapSol: Math.round((row.liquidity * 4) / 100),
    tokenScore: row.safetyScore, creatorScore: row.creatorScore,
  };
}

const ORDER_STATUSES: OrderStatus[] = ["open", "filled", "cancelled"];
const orders: Order[] = POSITION_TOKENS.slice(0, 6).map((t, i) => {
  const seed = seedOf(t.mint) + i;
  return {
    id: `ord-${i + 1}`, tokenSymbol: t.symbol, tokenMint: t.mint,
    side: seed % 2 === 0 ? "buy" : "sell",
    type: seed % 3 === 0 ? "limit" : "market",
    status: ORDER_STATUSES[seed % 3],
    price: px(0.001 + (seed % 40) / 10000),
    amountSol: 5 + (seed % 25), createdAt: `${2 + (seed % 40)} dk önce`,
  };
});

const TXN_KINDS = ["buy", "sell", "approve"] as const;
const TXN_STATUSES = ["success", "pending", "failed"] as const;
const transactions: Txn[] = POSITION_TOKENS.map((t, i) => {
  const seed = seedOf(t.mint) + i * 3;
  return {
    id: `tx-${i + 1}`,
    hash: `${t.mint.slice(0, 4)}${seed.toString(16)}...${(seed * 7).toString(16).slice(-4)}`,
    kind: TXN_KINDS[seed % 3], tokenSymbol: t.symbol, amountSol: 3 + (seed % 30),
    status: TXN_STATUSES[seed % 3], time: `${1 + (seed % 55)} dk önce`,
  };
});

const LOG_MESSAGES = [
  "Sinyal alındı: momentum eşiği aşıldı",
  "Emir simülasyonu tamamlandı",
  "Likidite kontrolü geçti",
  "Creator skoru güncellendi",
  "Slippage toleransı yeniden hesaplandı",
  "Risk limiti kontrolü: uygun",
];
const LOG_LEVELS = ["info", "warn", "error"] as const;
const tradeLogs: TradeLog[] = Array.from({ length: 10 }, (_, i) => {
  const seed = 41 + i * 7;
  return { id: `log-${i + 1}`, level: LOG_LEVELS[seed % 3], message: LOG_MESSAGES[i % LOG_MESSAGES.length], time: `${i * 2 + 1} dk önce` };
});
```

Add adapters to `mockApi` (after `getPositions`):
```ts
  getCandles: (mint) => delay(buildCandles(mint)),
  getMarketData: (mint) => delay(buildMarketData(mint)),
  getOrders: () => delay(orders),
  getTransactions: () => delay(transactions),
  getTradeLogs: () => delay(tradeLogs),
```

- [ ] **Step 7: Run adapter test to verify it passes**

Run: `cd apps/web && npx vitest run lib/api/terminal.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 8: Add query keys + hooks + hook test**

In `apps/web/lib/get-query-client.ts`, add to `qk`:
```ts
  candles: (mint: string) => ["candles", mint] as const,
  marketData: (mint: string) => ["market-data", mint] as const,
  orders: ["orders"] as const,
  transactions: ["transactions"] as const,
  tradeLogs: ["trade-logs"] as const,
```

In `apps/web/lib/hooks/queries.ts`, append:
```ts
export function useCandles(mint: string) {
  return useQuery({ queryKey: qk.candles(mint), queryFn: () => getApi().getCandles(mint) });
}
export function useMarketData(mint: string) {
  return useQuery({ queryKey: qk.marketData(mint), queryFn: () => getApi().getMarketData(mint) });
}
export function useOrders() {
  return useQuery({ queryKey: qk.orders, queryFn: () => getApi().getOrders() });
}
export function useTransactions() {
  return useQuery({ queryKey: qk.transactions, queryFn: () => getApi().getTransactions() });
}
export function useTradeLogs() {
  return useQuery({ queryKey: qk.tradeLogs, queryFn: () => getApi().getTradeLogs() });
}
```

Create `apps/web/lib/hooks/terminal-hooks.test.tsx`:
```tsx
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useMarketData, useOrders } from "./queries";

function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("useMarketData resolves market data", async () => {
  const { result } = renderHook(() => useMarketData("PULSE"), { wrapper });
  await waitFor(() => expect(result.current.data?.symbol).toBe("PULSE"));
});

test("useOrders resolves a non-empty list", async () => {
  const { result } = renderHook(() => useOrders(), { wrapper });
  await waitFor(() => expect((result.current.data ?? []).length).toBeGreaterThan(0));
});
```

- [ ] **Step 9: Run hook test to verify it passes**

Run: `cd apps/web && npx vitest run lib/hooks/terminal-hooks.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 10: Commit**

```bash
git add apps/web/lib/api/types.ts apps/web/lib/api/contract.ts apps/web/lib/api/http.ts apps/web/lib/api/mock.ts apps/web/lib/get-query-client.ts apps/web/lib/hooks/queries.ts apps/web/lib/api/terminal.test.ts apps/web/lib/hooks/terminal-hooks.test.tsx
git commit -m "feat(terminal): data seam for candles/market/orders/txns/logs"
```

---

## Task 2: Terminal config + pure order logic

**Files:**
- Create: `apps/web/lib/terminal/order-defs.ts`
- Create: `apps/web/lib/terminal/order-logic.ts`
- Test: `apps/web/lib/terminal/order-logic.test.ts`

**Interfaces:**
- Produces: `ORDER_SIDE_DEFS`, `ORDER_TYPE_DEFS`, `TERMINAL_TAB_DEFS`, `OrderDraft`, `DEFAULT_ORDER_DRAFT`, `MOCK_WALLET_SOL`, `DEFAULT_TERMINAL_MINT` (order-defs); `OrderErrors`, `validateOrder(d, market)`, `OrderSimulation`, `simulateOrder(d, market)` (order-logic).
- Consumes: `MarketData`, `OrderSide`, `OrderType` from `@/lib/api/types`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/terminal/order-logic.test.ts`:
```ts
import { validateOrder, simulateOrder } from "./order-logic";
import { DEFAULT_ORDER_DRAFT, MOCK_WALLET_SOL } from "./order-defs";
import type { MarketData } from "@/lib/api/types";

const market: MarketData = {
  mint: "9xQeWv...4Fk2", symbol: "PULSE", price: 0.004, change24hPct: 5,
  liquiditySol: 800, volume24hSol: 400, marketCapSol: 3200, tokenScore: 78, creatorScore: 82,
};

test("a valid default draft has no errors", () => {
  expect(validateOrder(DEFAULT_ORDER_DRAFT, market)).toEqual({});
});

test("zero/negative amount and over-balance are rejected", () => {
  expect(validateOrder({ ...DEFAULT_ORDER_DRAFT, amountSol: 0 }, market).amountSol).toBeTruthy();
  expect(validateOrder({ ...DEFAULT_ORDER_DRAFT, amountSol: MOCK_WALLET_SOL + 1 }, market).amountSol).toBeTruthy();
});

test("out-of-range slippage and missing limit price are rejected", () => {
  expect(validateOrder({ ...DEFAULT_ORDER_DRAFT, slippagePct: 80 }, market).slippagePct).toBeTruthy();
  expect(validateOrder({ ...DEFAULT_ORDER_DRAFT, type: "limit" }, market).limitPrice).toBeTruthy();
});

test("simulateOrder derives impact, minReceived and fee deterministically", () => {
  const a = simulateOrder(DEFAULT_ORDER_DRAFT, market);
  const b = simulateOrder(DEFAULT_ORDER_DRAFT, market);
  expect(a).toEqual(b);
  expect(a.estPrice).toBe(market.price);
  expect(a.priceImpactPct).toBeGreaterThanOrEqual(0);
  expect(a.minReceived).toBeGreaterThan(0);
  expect(a.route).toBe("Jupiter");
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/terminal/order-logic.test.ts`
Expected: FAIL — modules not found.

- [ ] **Step 3: Write order-defs**

Create `apps/web/lib/terminal/order-defs.ts`:
```ts
import type { OrderSide, OrderType } from "@/lib/api/types";

export interface OrderDraft {
  side: OrderSide; type: OrderType;
  amountSol: number; sizePct: number;
  limitPrice?: number; slippagePct: number; priorityFee: number;
  stopLossPct?: number; takeProfitPct?: number; trailingPct?: number;
}

export const DEFAULT_ORDER_DRAFT: OrderDraft = {
  side: "buy", type: "market", amountSol: 1, sizePct: 25, slippagePct: 1, priorityFee: 0.0001,
};

export const MOCK_WALLET_SOL = 210;
export const DEFAULT_TERMINAL_MINT = "9xQeWv...4Fk2"; // PULSE

export const ORDER_SIDE_DEFS: { key: OrderSide; label: string; color: string }[] = [
  { key: "buy", label: "Al", color: "#2FD98B" },
  { key: "sell", label: "Sat", color: "#F0476B" },
];
export const ORDER_TYPE_DEFS: { key: OrderType; label: string }[] = [
  { key: "market", label: "Market" },
  { key: "limit", label: "Limit" },
];
export const TERMINAL_TAB_DEFS: { key: string; label: string }[] = [
  { key: "positions", label: "Pozisyonlar" },
  { key: "orders", label: "Emirler" },
  { key: "transactions", label: "İşlemler" },
  { key: "logs", label: "Loglar" },
];
```

- [ ] **Step 4: Write order-logic**

Create `apps/web/lib/terminal/order-logic.ts`:
```ts
import type { MarketData } from "@/lib/api/types";
import { type OrderDraft, MOCK_WALLET_SOL } from "./order-defs";

export interface OrderErrors { [field: string]: string }

export function validateOrder(d: OrderDraft, _market: MarketData): OrderErrors {
  const e: OrderErrors = {};
  if (!(d.amountSol > 0)) e.amountSol = "Miktar 0'dan büyük olmalı";
  else if (d.amountSol > MOCK_WALLET_SOL) e.amountSol = "Bakiye yetersiz";
  if (d.slippagePct < 0 || d.slippagePct > 50) e.slippagePct = "Slippage %0–50 arası olmalı";
  if (d.priorityFee < 0) e.priorityFee = "Öncelik ücreti negatif olamaz";
  if (d.type === "limit" && !(d.limitPrice != null && d.limitPrice > 0)) e.limitPrice = "Limit fiyatı gir";
  if (d.stopLossPct != null && (d.stopLossPct < 0 || d.stopLossPct > 100)) e.stopLossPct = "SL %0–100 arası";
  if (d.takeProfitPct != null && d.takeProfitPct < 0) e.takeProfitPct = "TP negatif olamaz";
  return e;
}

export interface OrderSimulation {
  estPrice: number; priceImpactPct: number; minReceived: number; estFeeSol: number; route: string;
}

export function simulateOrder(d: OrderDraft, market: MarketData): OrderSimulation {
  const estPrice = d.type === "limit" && d.limitPrice ? d.limitPrice : market.price;
  const priceImpactPct = Math.min(15, (d.amountSol / Math.max(1, market.liquiditySol)) * 100);
  const grossTokens = estPrice > 0 ? d.amountSol / estPrice : 0;
  const minReceived = grossTokens * (1 - d.slippagePct / 100) * (1 - priceImpactPct / 100);
  const estFeeSol = 0.000005 + (d.priorityFee / 1e9) * 200000;
  const r = (n: number, p: number) => Math.round(n * 10 ** p) / 10 ** p;
  return {
    estPrice, priceImpactPct: r(priceImpactPct, 2), minReceived: r(minReceived, 4),
    estFeeSol: r(estFeeSol, 6), route: "Jupiter",
  };
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd apps/web && npx vitest run lib/terminal/order-logic.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 6: Commit**

```bash
git add apps/web/lib/terminal/
git commit -m "feat(terminal): order registries + pure validate/simulate logic"
```

---

## Task 3: shadcn Dialog primitive

**Files:**
- Create: `apps/web/components/ui/dialog.tsx`
- Test: `apps/web/components/ui/dialog.test.tsx`

**Interfaces:**
- Produces: `Dialog`, `DialogTrigger`, `DialogClose`, `DialogContent`, `DialogHeader`, `DialogFooter`, `DialogTitle`, `DialogDescription`. `Dialog` accepts `open` / `onOpenChange` (Base-UI `Dialog.Root`).
- Consumes: `@base-ui/react/dialog`, `cn` from `@/lib/utils`, `XIcon` from `lucide-react`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/ui/dialog.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { Dialog, DialogContent, DialogTitle } from "./dialog";

test("renders dialog content when open", () => {
  render(
    <Dialog open>
      <DialogContent><DialogTitle>Onay</DialogTitle></DialogContent>
    </Dialog>
  );
  expect(screen.getByText("Onay")).toBeInTheDocument();
});

test("does not render content when closed", () => {
  render(
    <Dialog open={false}>
      <DialogContent><DialogTitle>Gizli</DialogTitle></DialogContent>
    </Dialog>
  );
  expect(screen.queryByText("Gizli")).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/ui/dialog.test.tsx`
Expected: FAIL — `./dialog` not found.

- [ ] **Step 3: Write dialog.tsx**

Create `apps/web/components/ui/dialog.tsx` (mirrors `sheet.tsx`; centered modal variant):
```tsx
"use client";
import * as React from "react";
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { XIcon } from "lucide-react";
import { cn } from "@/lib/utils";

function Dialog(props: React.ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root data-slot="dialog" {...props} />;
}
function DialogTrigger(props: React.ComponentProps<typeof DialogPrimitive.Trigger>) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />;
}
function DialogClose(props: React.ComponentProps<typeof DialogPrimitive.Close>) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />;
}
function DialogContent({
  className, children, showCloseButton = true, ...props
}: React.ComponentProps<typeof DialogPrimitive.Popup> & { showCloseButton?: boolean }) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Backdrop className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm" />
      <DialogPrimitive.Popup
        data-slot="dialog-content"
        className={cn(
          "fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-card p-5 shadow-xl",
          className
        )}
        {...props}
      >
        {children}
        {showCloseButton && (
          <DialogPrimitive.Close className="absolute right-4 top-4 text-muted-foreground hover:text-foreground">
            <XIcon size={16} />
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Popup>
    </DialogPrimitive.Portal>
  );
}
function DialogHeader({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="dialog-header" className={cn("mb-3 flex flex-col gap-1", className)} {...props} />;
}
function DialogFooter({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="dialog-footer" className={cn("mt-5 flex justify-end gap-2", className)} {...props} />;
}
function DialogTitle({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Title>) {
  return <DialogPrimitive.Title data-slot="dialog-title" className={cn("text-base font-semibold", className)} {...props} />;
}
function DialogDescription({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      data-slot="dialog-description"
      className={cn("text-muted-foreground", className)}
      style={{ fontSize: 13 }}
      {...props}
    />
  );
}

export {
  Dialog, DialogTrigger, DialogClose, DialogContent,
  DialogHeader, DialogFooter, DialogTitle, DialogDescription,
};
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/ui/dialog.test.tsx`
Expected: PASS (2 tests). If Base-UI requires a `<DialogPrimitive.Portal>` container in jsdom and the closed case still mounts a portal node, the `queryByText` assertion still holds because Base-UI unmounts popup content when `open={false}`.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/ui/dialog.tsx apps/web/components/ui/dialog.test.tsx
git commit -m "feat(ui): add Base-UI Dialog primitive for confirmation modal"
```

---

## Task 4: TokenWatchlistPanel (left)

**Files:**
- Create: `apps/web/components/terminal/TokenWatchlistPanel.tsx`
- Test: `apps/web/components/terminal/TokenWatchlistPanel.test.tsx`

**Interfaces:**
- Produces: `TokenWatchlistPanel({ activeMint, onSelect }: { activeMint: string; onSelect: (mint: string) => void })`.
- Consumes: `useTokens` from `@/lib/hooks/queries`, `cn` from `@/lib/utils`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/TokenWatchlistPanel.test.tsx`:
```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TokenWatchlistPanel } from "./TokenWatchlistPanel";

function renderPanel(onSelect = () => {}) {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <TokenWatchlistPanel activeMint="9xQeWv...4Fk2" onSelect={onSelect} />
    </QueryClientProvider>
  );
}

test("lists token symbols and reports selection", async () => {
  const onSelect = vi.fn();
  renderPanel(onSelect);
  await waitFor(() => expect(screen.getByText("NOVA")).toBeInTheDocument());
  fireEvent.click(screen.getByText("NOVA"));
  expect(onSelect).toHaveBeenCalled();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/TokenWatchlistPanel.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/TokenWatchlistPanel.tsx`:
```tsx
"use client";
import { useTokens } from "@/lib/hooks/queries";
import { cn } from "@/lib/utils";

export function TokenWatchlistPanel({ activeMint, onSelect }: { activeMint: string; onSelect: (mint: string) => void }) {
  const { data: tokens } = useTokens();
  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2 font-medium" style={{ fontSize: 13 }}>Tokenlar</div>
      <div className="flex flex-col overflow-y-auto">
        {(tokens ?? []).map((t) => (
          <button
            key={t.mint}
            onClick={() => onSelect(t.mint)}
            data-active={t.mint === activeMint}
            className={cn(
              "flex items-center justify-between px-3 py-2 text-left hover:bg-accent",
              t.mint === activeMint && "bg-accent"
            )}
            style={{ fontSize: 12 }}
          >
            <span className="font-medium">{t.symbol}</span>
            <span className="text-muted-foreground">{t.price}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/TokenWatchlistPanel.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/TokenWatchlistPanel.tsx apps/web/components/terminal/TokenWatchlistPanel.test.tsx
git commit -m "feat(terminal): TokenWatchlistPanel with active selection"
```

---

## Task 5: MarketDataHeader (center top)

**Files:**
- Create: `apps/web/components/terminal/MarketDataHeader.tsx`
- Test: `apps/web/components/terminal/MarketDataHeader.test.tsx`

**Interfaces:**
- Produces: `MarketDataHeader({ mint }: { mint: string })`.
- Consumes: `useMarketData`, `pnlColor` (`@/lib/position/risk-filter`), `scoreToLevel`/`riskMeta` (`@/lib/format`).

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/MarketDataHeader.test.tsx`:
```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { MarketDataHeader } from "./MarketDataHeader";

test("shows symbol, price and score labels", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <MarketDataHeader mint="PULSE" />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("PULSE")).toBeInTheDocument());
  expect(screen.getByText("Likidite")).toBeInTheDocument();
  expect(screen.getByText(/Token \d+/)).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/MarketDataHeader.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/MarketDataHeader.tsx`:
```tsx
"use client";
import { useMarketData } from "@/lib/hooks/queries";
import { pnlColor } from "@/lib/position/risk-filter";
import { scoreToLevel, riskMeta } from "@/lib/format";

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-muted-foreground" style={{ fontSize: 11 }}>{label}</span>
      <span style={{ fontSize: 13 }}>{value}</span>
    </div>
  );
}
function ScoreBadge({ label, score }: { label: string; score: number }) {
  const m = riskMeta[scoreToLevel(score)];
  return (
    <span className="rounded-md px-2 py-0.5" style={{ fontSize: 11, color: m.color, backgroundColor: m.bg, border: `1px solid ${m.border}` }}>
      {label} {score}
    </span>
  );
}

export function MarketDataHeader({ mint }: { mint: string }) {
  const { data } = useMarketData(mint);
  if (!data) return <div className="h-16 rounded-lg border border-border bg-card" />;
  return (
    <div className="flex flex-wrap items-center gap-4 rounded-lg border border-border bg-card px-4 py-3">
      <span className="font-semibold">{data.symbol}</span>
      <span style={{ fontSize: 14 }}>{data.price} SOL</span>
      <span style={{ fontSize: 13, color: pnlColor(data.change24hPct) }}>%{data.change24hPct}</span>
      <Stat label="Likidite" value={`${data.liquiditySol} SOL`} />
      <Stat label="Hacim 24s" value={`${data.volume24hSol} SOL`} />
      <Stat label="Piyasa Değeri" value={`${data.marketCapSol} SOL`} />
      <div className="ml-auto flex items-center gap-2">
        <ScoreBadge label="Token" score={data.tokenScore} />
        <ScoreBadge label="Creator" score={data.creatorScore} />
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/MarketDataHeader.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/MarketDataHeader.tsx apps/web/components/terminal/MarketDataHeader.test.tsx
git commit -m "feat(terminal): MarketDataHeader with price + score badges"
```

---

## Task 6: PriceChart (lightweight-charts, ssr:false)

**Files:**
- Create: `apps/web/components/terminal/PriceChartCanvas.tsx`
- Create: `apps/web/components/terminal/PriceChart.tsx`
- Test: `apps/web/components/terminal/PriceChartCanvas.test.tsx`
- Modify: `apps/web/package.json` (add `lightweight-charts`)

**Interfaces:**
- Produces: `PriceChartCanvas({ candles }: { candles: Candle[] })` (named export), `PriceChart({ mint }: { mint: string })`.
- Consumes: `lightweight-charts` (`createChart`, `ColorType`), `useCandles`, `Skeleton` (`@/components/ui/skeleton`), `Candle` type.

- [ ] **Step 1: Install lightweight-charts (pinned v4)**

Run: `cd apps/web && npm install lightweight-charts@^4.2.0`
Expected: `package.json` gains `"lightweight-charts": "^4.2.0"` under dependencies.

- [ ] **Step 2: Write the failing test**

Create `apps/web/components/terminal/PriceChartCanvas.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { PriceChartCanvas } from "./PriceChartCanvas";

vi.mock("lightweight-charts", () => ({
  ColorType: { Solid: "solid" },
  createChart: vi.fn(() => ({
    addCandlestickSeries: vi.fn(() => ({ setData: vi.fn() })),
    timeScale: vi.fn(() => ({ fitContent: vi.fn() })),
    remove: vi.fn(),
  })),
}));
import { createChart } from "lightweight-charts";

test("creates a candlestick chart and renders a container", () => {
  render(<PriceChartCanvas candles={[{ time: 1, open: 1, high: 2, low: 0.5, close: 1.5 }]} />);
  expect(createChart).toHaveBeenCalled();
  expect(screen.getByTestId("price-chart")).toBeInTheDocument();
});
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/PriceChartCanvas.test.tsx`
Expected: FAIL — `./PriceChartCanvas` not found.

- [ ] **Step 4: Write PriceChartCanvas**

Create `apps/web/components/terminal/PriceChartCanvas.tsx`:
```tsx
"use client";
import { useEffect, useRef } from "react";
import { createChart, ColorType, type IChartApi } from "lightweight-charts";
import type { Candle } from "@/lib/api/types";

export function PriceChartCanvas({ candles }: { candles: Candle[] }) {
  const ref = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  useEffect(() => {
    if (!ref.current) return;
    const chart = createChart(ref.current, {
      autoSize: true,
      layout: { background: { type: ColorType.Solid, color: "transparent" }, textColor: "#8A94A6" },
      grid: { vertLines: { color: "rgba(255,255,255,0.05)" }, horzLines: { color: "rgba(255,255,255,0.05)" } },
      rightPriceScale: { borderColor: "rgba(255,255,255,0.07)" },
      timeScale: { borderColor: "rgba(255,255,255,0.07)" },
    });
    const series = chart.addCandlestickSeries({
      upColor: "#2FD98B", downColor: "#F0476B",
      wickUpColor: "#2FD98B", wickDownColor: "#F0476B", borderVisible: false,
    });
    series.setData(candles as never);
    chart.timeScale().fitContent();
    chartRef.current = chart;
    return () => { chartRef.current = null; chart.remove(); };
  }, [candles]);
  return <div ref={ref} data-testid="price-chart" className="h-[320px] w-full rounded-lg border border-border bg-card" />;
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/PriceChartCanvas.test.tsx`
Expected: PASS.

- [ ] **Step 6: Write PriceChart wrapper**

Create `apps/web/components/terminal/PriceChart.tsx`:
```tsx
"use client";
import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";
import { useCandles } from "@/lib/hooks/queries";

const PriceChartCanvas = dynamic(() => import("./PriceChartCanvas").then((m) => m.PriceChartCanvas), {
  ssr: false, loading: () => <Skeleton className="h-[320px] w-full" />,
});

export function PriceChart({ mint }: { mint: string }) {
  const { data } = useCandles(mint);
  if (!data) return <Skeleton className="h-[320px] w-full" />;
  return <PriceChartCanvas candles={data} />;
}
```

- [ ] **Step 7: Commit**

```bash
git add apps/web/package.json apps/web/package-lock.json apps/web/components/terminal/PriceChart.tsx apps/web/components/terminal/PriceChartCanvas.tsx apps/web/components/terminal/PriceChartCanvas.test.tsx
git commit -m "feat(terminal): candlestick PriceChart via lightweight-charts (ssr:false)"
```

---

## Task 7: OrderConfirmDialog (simulated submit)

**Files:**
- Create: `apps/web/components/terminal/OrderConfirmDialog.tsx`
- Test: `apps/web/components/terminal/OrderConfirmDialog.test.tsx`

**Interfaces:**
- Produces: `OrderConfirmDialog({ open, draft, market, onClose }: { open: boolean; draft: OrderDraft; market: MarketData; onClose: () => void })`.
- Consumes: `Dialog`/`DialogContent`/`DialogHeader`/`DialogTitle`/`DialogFooter` (`@/components/ui/dialog`), `simulateOrder` (`@/lib/terminal/order-logic`), `ORDER_SIDE_DEFS` (`@/lib/terminal/order-defs`), `useSessionStore` (`@/lib/store/session`), `toast` (sonner), `scoreToLevel`/`riskMeta` (`@/lib/format`).

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/OrderConfirmDialog.test.tsx`:
```tsx
import { render, screen, fireEvent, act } from "@testing-library/react";
import { vi } from "vitest";
import { OrderConfirmDialog } from "./OrderConfirmDialog";
import { DEFAULT_ORDER_DRAFT } from "@/lib/terminal/order-defs";
import { useSessionStore } from "@/lib/store/session";
import type { MarketData } from "@/lib/api/types";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { warning: vi.fn() }) }));
import { toast } from "sonner";

const market: MarketData = {
  mint: "9xQeWv...4Fk2", symbol: "PULSE", price: 0.004, change24hPct: 5,
  liquiditySol: 800, volume24hSol: 400, marketCapSol: 3200, tokenScore: 78, creatorScore: 82,
};
const warnMock = () => (toast as unknown as { warning: ReturnType<typeof vi.fn> }).warning;

beforeEach(() => {
  (toast as unknown as ReturnType<typeof vi.fn>).mockClear();
  warnMock().mockClear();
  act(() => useSessionStore.getState().setTradingMode("paper"));
});

test("paper mode confirm fires a simulate toast, not a warning", () => {
  render(<OrderConfirmDialog open draft={DEFAULT_ORDER_DRAFT} market={market} onClose={() => {}} />);
  fireEvent.click(screen.getByRole("button", { name: "Onayla" }));
  expect(toast).toHaveBeenCalled();
  expect(warnMock()).not.toHaveBeenCalled();
});

test("live mode confirm fires a warning and never a real trade", () => {
  act(() => useSessionStore.getState().setTradingMode("live"));
  render(<OrderConfirmDialog open draft={DEFAULT_ORDER_DRAFT} market={market} onClose={() => {}} />);
  fireEvent.click(screen.getByRole("button", { name: "Onayla" }));
  expect(warnMock()).toHaveBeenCalled();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/OrderConfirmDialog.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/OrderConfirmDialog.tsx`:
```tsx
"use client";
import { toast } from "sonner";
import { useSessionStore } from "@/lib/store/session";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { simulateOrder } from "@/lib/terminal/order-logic";
import { ORDER_SIDE_DEFS, MOCK_WALLET_SOL, type OrderDraft } from "@/lib/terminal/order-defs";
import { scoreToLevel, riskMeta } from "@/lib/format";
import type { MarketData } from "@/lib/api/types";

function Row({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="flex items-center justify-between py-1" style={{ fontSize: 13 }}>
      <span className="text-muted-foreground">{label}</span>
      <span style={{ color }}>{value}</span>
    </div>
  );
}

export function OrderConfirmDialog({ open, draft, market, onClose }: {
  open: boolean; draft: OrderDraft; market: MarketData; onClose: () => void;
}) {
  const mode = useSessionStore((s) => s.tradingMode);
  const sim = simulateOrder(draft, market);
  const side = ORDER_SIDE_DEFS.find((s) => s.key === draft.side)!;
  const risky = market.tokenScore < 50 || market.creatorScore < 50;

  const confirm = () => {
    const summary = `${side.label} ${draft.amountSol} SOL · ${market.symbol}`;
    if (mode === "live") {
      toast.warning(`CANLI mod — ${summary}`, { description: "Gerçek para. Bu demoda emir gönderilmez." });
    } else {
      toast(`Emir simüle edildi — ${summary}`, {
        description: `${mode === "paper" ? "Kağıt" : "Gölge"} modda. Etki %${sim.priceImpactPct}, min ${sim.minReceived}.`,
      });
    }
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent>
        <DialogHeader><DialogTitle>Emir Onayı</DialogTitle></DialogHeader>
        <div>
          <Row label="Token" value={market.symbol} />
          <Row label="İşlem" value={side.label} color={side.color} />
          <Row label="Tutar" value={`${draft.amountSol} SOL`} />
          <Row label="Tahmini Fiyat" value={`${sim.estPrice} SOL`} />
          <Row label="Slippage" value={`%${draft.slippagePct}`} />
          <Row label="Fiyat Etkisi" value={`%${sim.priceImpactPct}`} />
          <Row label="Min. Alınan" value={`${sim.minReceived}`} />
          <Row label="Tahmini Ücret" value={`${sim.estFeeSol} SOL`} />
          <Row label="Cüzdan Bakiyesi" value={`${MOCK_WALLET_SOL} SOL`} />
          <Row label="Token Skoru" value={`${market.tokenScore} · ${riskMeta[scoreToLevel(market.tokenScore)].label}`} color={riskMeta[scoreToLevel(market.tokenScore)].color} />
          <Row label="Creator Skoru" value={`${market.creatorScore} · ${riskMeta[scoreToLevel(market.creatorScore)].label}`} color={riskMeta[scoreToLevel(market.creatorScore)].color} />
        </div>
        {risky && (
          <div className="mt-2 rounded-md px-3 py-2" style={{ fontSize: 12, color: "#FFB020", backgroundColor: "rgba(255,176,32,0.1)", border: "1px solid rgba(255,176,32,0.35)" }}>
            Düşük token/creator skoru — yüksek risk.
          </div>
        )}
        {mode === "live" && (
          <div className="mt-2 rounded-md px-3 py-2" style={{ fontSize: 12, color: "#F0476B", backgroundColor: "rgba(240,71,107,0.12)", border: "1px solid rgba(240,71,107,0.4)" }}>
            CANLI işlem — gerçek para riski. Bu demoda emir gönderilmez.
          </div>
        )}
        <DialogFooter>
          <button onClick={onClose} className="rounded-md border border-border px-3 py-1.5" style={{ fontSize: 12 }}>Vazgeç</button>
          <button onClick={confirm} className="rounded-md px-3 py-1.5 text-primary-foreground" style={{ fontSize: 12, fontWeight: 600, backgroundColor: side.color, color: "#08210F" }}>Onayla</button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/OrderConfirmDialog.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/OrderConfirmDialog.tsx apps/web/components/terminal/OrderConfirmDialog.test.tsx
git commit -m "feat(terminal): OrderConfirmDialog with live-warning + simulate toast"
```

---

## Task 8: OrderPanel (right)

**Files:**
- Create: `apps/web/components/terminal/OrderPanel.tsx`
- Test: `apps/web/components/terminal/OrderPanel.test.tsx`

**Interfaces:**
- Produces: `OrderPanel({ mint }: { mint: string })`.
- Consumes: `useMarketData`, `validateOrder`/`simulateOrder`, `DEFAULT_ORDER_DRAFT`/`ORDER_SIDE_DEFS`/`ORDER_TYPE_DEFS`/`OrderDraft`, `OrderConfirmDialog`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/OrderPanel.test.tsx`:
```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { OrderPanel } from "./OrderPanel";

function renderPanel() {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <OrderPanel mint="PULSE" />
    </QueryClientProvider>
  );
}

test("renders order fields and a simulation summary", async () => {
  renderPanel();
  await waitFor(() => expect(screen.getByLabelText("Miktar (SOL)")).toBeInTheDocument());
  expect(screen.getByText("Fiyat Etkisi")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Al" })).toBeInTheDocument();
});

test("invalid amount shows an error and disables preview", async () => {
  renderPanel();
  const amount = await screen.findByLabelText("Miktar (SOL)");
  fireEvent.change(amount, { target: { value: "0" } });
  expect(screen.getByText("Miktar 0'dan büyük olmalı")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Önizle" })).toBeDisabled();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/OrderPanel.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/OrderPanel.tsx`:
```tsx
"use client";
import { useState } from "react";
import { useMarketData } from "@/lib/hooks/queries";
import { validateOrder, simulateOrder } from "@/lib/terminal/order-logic";
import { DEFAULT_ORDER_DRAFT, ORDER_SIDE_DEFS, ORDER_TYPE_DEFS, type OrderDraft } from "@/lib/terminal/order-defs";
import { OrderConfirmDialog } from "./OrderConfirmDialog";
import { cn } from "@/lib/utils";

function NumberField({ id, label, value, onChange }: { id: string; label: string; value: number; onChange: (v: number) => void }) {
  return (
    <label htmlFor={id} className="flex flex-col gap-1" style={{ fontSize: 12 }}>
      <span className="text-muted-foreground">{label}</span>
      <input
        id={id} aria-label={label} type="number" value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="rounded-md border border-border bg-background px-2 py-1.5"
      />
    </label>
  );
}

export function OrderPanel({ mint }: { mint: string }) {
  const { data: market } = useMarketData(mint);
  const [draft, setDraft] = useState<OrderDraft>(DEFAULT_ORDER_DRAFT);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const set = (patch: Partial<OrderDraft>) => setDraft((d) => ({ ...d, ...patch }));

  if (!market) return <div className="h-full rounded-lg border border-border bg-card p-3 text-muted-foreground" style={{ fontSize: 13 }}>Yükleniyor…</div>;

  const errors = validateOrder(draft, market);
  const sim = simulateOrder(draft, market);
  const valid = Object.keys(errors).length === 0;

  return (
    <div className="flex h-full flex-col gap-3 rounded-lg border border-border bg-card p-3">
      <div className="flex gap-1">
        {ORDER_SIDE_DEFS.map((s) => (
          <button key={s.key} onClick={() => set({ side: s.key })}
            className={cn("flex-1 rounded-md px-2 py-1.5", draft.side === s.key && "text-black")}
            style={{ fontSize: 12, fontWeight: 600, backgroundColor: draft.side === s.key ? s.color : "transparent", color: draft.side === s.key ? "#08210F" : s.color, border: `1px solid ${s.color}` }}>
            {s.label}
          </button>
        ))}
      </div>
      <div className="flex gap-1">
        {ORDER_TYPE_DEFS.map((t) => (
          <button key={t.key} onClick={() => set({ type: t.key })}
            className={cn("flex-1 rounded-md border px-2 py-1", draft.type === t.key ? "border-primary bg-accent" : "border-border")}
            style={{ fontSize: 12 }}>
            {t.label}
          </button>
        ))}
      </div>

      <NumberField id="amount" label="Miktar (SOL)" value={draft.amountSol} onChange={(v) => set({ amountSol: v })} />
      {errors.amountSol && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.amountSol}</span>}

      {draft.type === "limit" && (
        <>
          <NumberField id="limit" label="Limit Fiyatı" value={draft.limitPrice ?? 0} onChange={(v) => set({ limitPrice: v })} />
          {errors.limitPrice && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.limitPrice}</span>}
        </>
      )}

      <NumberField id="size" label="Pozisyon %" value={draft.sizePct} onChange={(v) => set({ sizePct: v })} />
      <NumberField id="slippage" label="Slippage %" value={draft.slippagePct} onChange={(v) => set({ slippagePct: v })} />
      {errors.slippagePct && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.slippagePct}</span>}
      <NumberField id="fee" label="Öncelik Ücreti (SOL)" value={draft.priorityFee} onChange={(v) => set({ priorityFee: v })} />
      <NumberField id="sl" label="Stop-Loss %" value={draft.stopLossPct ?? 0} onChange={(v) => set({ stopLossPct: v })} />
      <NumberField id="tp" label="Take-Profit %" value={draft.takeProfitPct ?? 0} onChange={(v) => set({ takeProfitPct: v })} />
      <NumberField id="trail" label="Trailing %" value={draft.trailingPct ?? 0} onChange={(v) => set({ trailingPct: v })} />

      <div className="mt-auto rounded-md border border-border p-2" style={{ fontSize: 12 }}>
        <div className="flex justify-between"><span className="text-muted-foreground">Tahmini Fiyat</span><span>{sim.estPrice} SOL</span></div>
        <div className="flex justify-between"><span className="text-muted-foreground">Fiyat Etkisi</span><span>%{sim.priceImpactPct}</span></div>
        <div className="flex justify-between"><span className="text-muted-foreground">Min. Alınan</span><span>{sim.minReceived}</span></div>
        <div className="flex justify-between"><span className="text-muted-foreground">Rota</span><span>{sim.route}</span></div>
      </div>

      <button disabled={!valid} onClick={() => setConfirmOpen(true)}
        className="rounded-md px-3 py-2 disabled:opacity-40"
        style={{ fontSize: 13, fontWeight: 600, backgroundColor: "#3E9BFF", color: "#04121F" }}>
        Önizle
      </button>

      <OrderConfirmDialog open={confirmOpen} draft={draft} market={market} onClose={() => setConfirmOpen(false)} />
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/OrderPanel.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/OrderPanel.tsx apps/web/components/terminal/OrderPanel.test.tsx
git commit -m "feat(terminal): OrderPanel form + live simulation status"
```

---

## Task 9: OrdersTable (bottom)

**Files:**
- Create: `apps/web/components/terminal/OrdersTable.tsx`
- Test: `apps/web/components/terminal/OrdersTable.test.tsx`

**Interfaces:**
- Produces: `OrdersTable()`.
- Consumes: `useOrders`, `toast` (sonner), `ORDER_SIDE_DEFS`, `Skeleton`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/OrdersTable.test.tsx`:
```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { OrdersTable } from "./OrdersTable";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { warning: vi.fn() }) }));
import { toast } from "sonner";

test("lists orders and cancels an open one with a simulate toast", async () => {
  render(<QueryClientProvider client={getQueryClient()}><OrdersTable /></QueryClientProvider>);
  await waitFor(() => expect(screen.getAllByText(/Al|Sat/).length).toBeGreaterThan(0));
  const cancelButtons = screen.queryAllByRole("button", { name: "İptal" });
  if (cancelButtons.length > 0) {
    fireEvent.click(cancelButtons[0]);
    expect(toast).toHaveBeenCalled();
  }
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/OrdersTable.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/OrdersTable.tsx`:
```tsx
"use client";
import { toast } from "sonner";
import { useOrders } from "@/lib/hooks/queries";
import { ORDER_SIDE_DEFS, ORDER_TYPE_DEFS } from "@/lib/terminal/order-defs";
import { Skeleton } from "@/components/ui/skeleton";

const STATUS_LABEL: Record<string, string> = { open: "Açık", filled: "Doldu", cancelled: "İptal" };

export function OrdersTable() {
  const { data, isLoading } = useOrders();
  if (isLoading || !data) return <Skeleton className="h-40 w-full" />;
  if (data.length === 0) return <div className="p-6 text-center text-muted-foreground" style={{ fontSize: 13 }}>Emir yok</div>;
  const cancel = (sym: string) => toast(`Emir iptal — ${sym}`, { description: "Bu demoda simüle edilir." });

  return (
    <table className="w-full" style={{ fontSize: 12 }}>
      <thead className="text-muted-foreground">
        <tr>
          <th className="px-3 py-2 text-left">Token</th><th className="px-3 py-2 text-left">Yön</th>
          <th className="px-3 py-2 text-left">Tür</th><th className="px-3 py-2 text-left">Durum</th>
          <th className="px-3 py-2 text-right">Fiyat</th><th className="px-3 py-2 text-right">Miktar</th>
          <th className="px-3 py-2 text-right">Aksiyon</th>
        </tr>
      </thead>
      <tbody>
        {data.map((o) => {
          const side = ORDER_SIDE_DEFS.find((s) => s.key === o.side)!;
          const type = ORDER_TYPE_DEFS.find((t) => t.key === o.type)!;
          return (
            <tr key={o.id} className="border-t border-border">
              <td className="px-3 py-2 font-medium">{o.tokenSymbol}</td>
              <td className="px-3 py-2" style={{ color: side.color }}>{side.label}</td>
              <td className="px-3 py-2">{type.label}</td>
              <td className="px-3 py-2">{STATUS_LABEL[o.status]}</td>
              <td className="px-3 py-2 text-right">{o.price}</td>
              <td className="px-3 py-2 text-right">{o.amountSol} SOL</td>
              <td className="px-3 py-2 text-right">
                {o.status === "open" && (
                  <button onClick={() => cancel(o.tokenSymbol)} className="rounded-md px-2 py-1" style={{ color: "#F0476B", border: "1px solid rgba(240,71,107,0.4)" }}>İptal</button>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/OrdersTable.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/OrdersTable.tsx apps/web/components/terminal/OrdersTable.test.tsx
git commit -m "feat(terminal): OrdersTable with simulated cancel"
```

---

## Task 10: TransactionsTable (bottom)

**Files:**
- Create: `apps/web/components/terminal/TransactionsTable.tsx`
- Test: `apps/web/components/terminal/TransactionsTable.test.tsx`

**Interfaces:**
- Produces: `TransactionsTable()`.
- Consumes: `useTransactions`, `Skeleton`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/TransactionsTable.test.tsx`:
```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TransactionsTable } from "./TransactionsTable";

test("lists transactions with an explorer link", async () => {
  render(<QueryClientProvider client={getQueryClient()}><TransactionsTable /></QueryClientProvider>);
  await waitFor(() => expect(screen.getAllByRole("link").length).toBeGreaterThan(0));
  expect(screen.getAllByRole("link")[0]).toHaveAttribute("href", expect.stringContaining("solscan.io"));
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/TransactionsTable.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/TransactionsTable.tsx`:
```tsx
"use client";
import { useTransactions } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";

const KIND_LABEL: Record<string, string> = { buy: "Alım", sell: "Satım", approve: "Onay" };
const STATUS_COLOR: Record<string, string> = { success: "#2FD98B", pending: "#FFB020", failed: "#F0476B" };
const STATUS_LABEL: Record<string, string> = { success: "Başarılı", pending: "Bekliyor", failed: "Başarısız" };

export function TransactionsTable() {
  const { data, isLoading } = useTransactions();
  if (isLoading || !data) return <Skeleton className="h-40 w-full" />;
  if (data.length === 0) return <div className="p-6 text-center text-muted-foreground" style={{ fontSize: 13 }}>İşlem yok</div>;
  return (
    <table className="w-full" style={{ fontSize: 12 }}>
      <thead className="text-muted-foreground">
        <tr>
          <th className="px-3 py-2 text-left">Hash</th><th className="px-3 py-2 text-left">Tür</th>
          <th className="px-3 py-2 text-left">Token</th><th className="px-3 py-2 text-right">Tutar</th>
          <th className="px-3 py-2 text-left">Durum</th><th className="px-3 py-2 text-right">Zaman</th>
        </tr>
      </thead>
      <tbody>
        {data.map((t) => (
          <tr key={t.id} className="border-t border-border">
            <td className="px-3 py-2">
              <a href={`https://solscan.io/tx/${t.hash}`} target="_blank" rel="noreferrer" className="text-primary hover:underline">{t.hash}</a>
            </td>
            <td className="px-3 py-2">{KIND_LABEL[t.kind]}</td>
            <td className="px-3 py-2 font-medium">{t.tokenSymbol}</td>
            <td className="px-3 py-2 text-right">{t.amountSol} SOL</td>
            <td className="px-3 py-2" style={{ color: STATUS_COLOR[t.status] }}>{STATUS_LABEL[t.status]}</td>
            <td className="px-3 py-2 text-right text-muted-foreground">{t.time}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/TransactionsTable.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/TransactionsTable.tsx apps/web/components/terminal/TransactionsTable.test.tsx
git commit -m "feat(terminal): TransactionsTable with explorer links"
```

---

## Task 11: TradeLogsList (bottom)

**Files:**
- Create: `apps/web/components/terminal/TradeLogsList.tsx`
- Test: `apps/web/components/terminal/TradeLogsList.test.tsx`

**Interfaces:**
- Produces: `TradeLogsList()`.
- Consumes: `useTradeLogs`, `Skeleton`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/TradeLogsList.test.tsx`:
```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TradeLogsList } from "./TradeLogsList";

test("renders log messages", async () => {
  render(<QueryClientProvider client={getQueryClient()}><TradeLogsList /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText(/Emir simülasyonu tamamlandı|Sinyal alındı/)).toBeInTheDocument());
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/TradeLogsList.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/TradeLogsList.tsx`:
```tsx
"use client";
import { useTradeLogs } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";

const LEVEL_COLOR: Record<string, string> = { info: "#8A94A6", warn: "#FFB020", error: "#F0476B" };

export function TradeLogsList() {
  const { data, isLoading } = useTradeLogs();
  if (isLoading || !data) return <Skeleton className="h-40 w-full" />;
  return (
    <ul className="divide-y divide-border font-mono" style={{ fontSize: 12 }}>
      {data.map((l) => (
        <li key={l.id} className="flex items-center gap-3 px-3 py-1.5">
          <span className="uppercase" style={{ color: LEVEL_COLOR[l.level], width: 44 }}>{l.level}</span>
          <span className="flex-1">{l.message}</span>
          <span className="text-muted-foreground">{l.time}</span>
        </li>
      ))}
    </ul>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/TradeLogsList.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/TradeLogsList.tsx apps/web/components/terminal/TradeLogsList.test.tsx
git commit -m "feat(terminal): TradeLogsList with level colors"
```

---

## Task 12: BottomTabsPanel (composition + Positions reuse)

**Files:**
- Create: `apps/web/components/terminal/BottomTabsPanel.tsx`
- Test: `apps/web/components/terminal/BottomTabsPanel.test.tsx`

**Interfaces:**
- Produces: `BottomTabsPanel()`.
- Consumes: `Tabs`/`TabsList`/`TabsTrigger`/`TabsContent` (`@/components/ui/tabs`), `TERMINAL_TAB_DEFS`, `usePositions`, `PositionsTable`+`SortKey` (`@/components/position/PositionsTable`), `OrdersTable`, `TransactionsTable`, `TradeLogsList`, `Skeleton`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/BottomTabsPanel.test.tsx`:
```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { BottomTabsPanel } from "./BottomTabsPanel";

function renderPanel() {
  return render(<QueryClientProvider client={getQueryClient()}><BottomTabsPanel /></QueryClientProvider>);
}

test("shows tab triggers and switches to the Orders tab", async () => {
  renderPanel();
  expect(screen.getByRole("tab", { name: "Pozisyonlar" })).toBeInTheDocument();
  fireEvent.click(screen.getByRole("tab", { name: "Emirler" }));
  await waitFor(() => expect(screen.getByText("Yön")).toBeInTheDocument());
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/BottomTabsPanel.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/BottomTabsPanel.tsx`:
```tsx
"use client";
import { useState } from "react";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { TERMINAL_TAB_DEFS } from "@/lib/terminal/order-defs";
import { usePositions } from "@/lib/hooks/queries";
import { PositionsTable, type SortKey } from "@/components/position/PositionsTable";
import { Skeleton } from "@/components/ui/skeleton";
import { OrdersTable } from "./OrdersTable";
import { TransactionsTable } from "./TransactionsTable";
import { TradeLogsList } from "./TradeLogsList";

function sortPositions<T extends { pnlSol: number; sizeSol: number; ageLabel: string }>(rows: T[], key: SortKey): T[] {
  return [...rows].sort((a, b) => {
    if (key === "ageLabel") return parseInt(b.ageLabel, 10) - parseInt(a.ageLabel, 10);
    return (b[key] as number) - (a[key] as number);
  });
}

function PositionsTab() {
  const { data } = usePositions();
  const [sortKey, setSortKey] = useState<SortKey>("pnlSol");
  if (!data) return <Skeleton className="h-40 w-full" />;
  return <PositionsTable rows={sortPositions(data, sortKey)} sortKey={sortKey} onSort={setSortKey} onRowClick={() => {}} />;
}

export function BottomTabsPanel() {
  return (
    <div className="rounded-lg border border-border bg-card">
      <Tabs defaultValue="positions" className="w-full">
        <TabsList className="flex flex-wrap">
          {TERMINAL_TAB_DEFS.map((t) => <TabsTrigger key={t.key} value={t.key}>{t.label}</TabsTrigger>)}
        </TabsList>
        <TabsContent value="positions" className="mt-2 overflow-x-auto"><PositionsTab /></TabsContent>
        <TabsContent value="orders" className="mt-2 overflow-x-auto"><OrdersTable /></TabsContent>
        <TabsContent value="transactions" className="mt-2 overflow-x-auto"><TransactionsTable /></TabsContent>
        <TabsContent value="logs" className="mt-2"><TradeLogsList /></TabsContent>
      </Tabs>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/BottomTabsPanel.test.tsx`
Expected: PASS. (Note: "Yön" is the Orders table header from Task 9.)

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/BottomTabsPanel.tsx apps/web/components/terminal/BottomTabsPanel.test.tsx
git commit -m "feat(terminal): BottomTabsPanel composing positions/orders/txns/logs"
```

---

## Task 13: TerminalContent (composition + activeMint)

**Files:**
- Create: `apps/web/components/terminal/TerminalContent.tsx`
- Test: `apps/web/components/terminal/TerminalContent.test.tsx`

**Interfaces:**
- Produces: `TerminalContent()`.
- Consumes: `TokenWatchlistPanel`, `MarketDataHeader`, `PriceChart`, `OrderPanel`, `BottomTabsPanel`, `DEFAULT_TERMINAL_MINT`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/terminal/TerminalContent.test.tsx`:
```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TerminalContent } from "./TerminalContent";

vi.mock("lightweight-charts", () => ({
  ColorType: { Solid: "solid" },
  createChart: vi.fn(() => ({
    addCandlestickSeries: vi.fn(() => ({ setData: vi.fn() })),
    timeScale: vi.fn(() => ({ fitContent: vi.fn() })),
    remove: vi.fn(),
  })),
}));

test("renders the four terminal regions", async () => {
  render(<QueryClientProvider client={getQueryClient()}><TerminalContent /></QueryClientProvider>);
  expect(screen.getByRole("heading", { name: "Terminal" })).toBeInTheDocument();
  await waitFor(() => expect(screen.getByText("Tokenlar")).toBeInTheDocument());   // watchlist
  await waitFor(() => expect(screen.getByText("PULSE")).toBeInTheDocument());      // market header
  expect(screen.getByRole("tab", { name: "Pozisyonlar" })).toBeInTheDocument();    // bottom tabs
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/terminal/TerminalContent.test.tsx`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the component**

Create `apps/web/components/terminal/TerminalContent.tsx`:
```tsx
"use client";
import { useState } from "react";
import { DEFAULT_TERMINAL_MINT } from "@/lib/terminal/order-defs";
import { TokenWatchlistPanel } from "./TokenWatchlistPanel";
import { MarketDataHeader } from "./MarketDataHeader";
import { PriceChart } from "./PriceChart";
import { OrderPanel } from "./OrderPanel";
import { BottomTabsPanel } from "./BottomTabsPanel";

export function TerminalContent() {
  const [activeMint, setActiveMint] = useState(DEFAULT_TERMINAL_MINT);
  return (
    <div className="flex flex-col gap-3">
      <h1>Terminal</h1>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-[220px_1fr_300px]">
        <TokenWatchlistPanel activeMint={activeMint} onSelect={setActiveMint} />
        <div className="flex flex-col gap-3">
          <MarketDataHeader mint={activeMint} />
          <PriceChart mint={activeMint} />
        </div>
        <OrderPanel mint={activeMint} />
      </div>
      <BottomTabsPanel />
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/terminal/TerminalContent.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/terminal/TerminalContent.tsx apps/web/components/terminal/TerminalContent.test.tsx
git commit -m "feat(terminal): TerminalContent composition with active-token state"
```

---

## Task 14: /terminal page + nav rename + remove /orders placeholder

**Files:**
- Create: `apps/web/app/(app)/terminal/page.tsx`
- Delete: `apps/web/app/(app)/orders/page.tsx`
- Modify: `apps/web/components/shell/nav.ts`

**Interfaces:**
- Consumes: `getApi`, `qk`, `getQueryClient`, `HydrationBoundary`/`dehydrate`, `TerminalContent`, `DEFAULT_TERMINAL_MINT`.

- [ ] **Step 1: Rename the nav entry**

In `apps/web/components/shell/nav.ts`: in the lucide import block, replace `ListOrdered` with `Terminal`. Change line 19 from:
```ts
  { label: "Emirler", path: "/orders", icon: ListOrdered },
```
to:
```ts
  { label: "Terminal", path: "/terminal", icon: Terminal },
```

- [ ] **Step 2: Create the page**

Create `apps/web/app/(app)/terminal/page.tsx`:
```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { TerminalContent } from "@/components/terminal/TerminalContent";
import { DEFAULT_TERMINAL_MINT } from "@/lib/terminal/order-defs";

export default async function TerminalPage() {
  const queryClient = getQueryClient();
  const api = getApi();
  await Promise.all([
    queryClient.prefetchQuery({ queryKey: qk.marketData(DEFAULT_TERMINAL_MINT), queryFn: () => api.getMarketData(DEFAULT_TERMINAL_MINT) }),
    queryClient.prefetchQuery({ queryKey: qk.candles(DEFAULT_TERMINAL_MINT), queryFn: () => api.getCandles(DEFAULT_TERMINAL_MINT) }),
    queryClient.prefetchQuery({ queryKey: qk.tokens, queryFn: () => api.getTokens() }),
    queryClient.prefetchQuery({ queryKey: qk.orders, queryFn: () => api.getOrders() }),
    queryClient.prefetchQuery({ queryKey: qk.transactions, queryFn: () => api.getTransactions() }),
    queryClient.prefetchQuery({ queryKey: qk.tradeLogs, queryFn: () => api.getTradeLogs() }),
    queryClient.prefetchQuery({ queryKey: qk.positions, queryFn: () => api.getPositions() }),
  ]);
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <TerminalContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 3: Delete the old placeholder**

Run: `git rm "apps/web/app/(app)/orders/page.tsx"`
(Removes the `/orders` route; nav no longer references it.)

- [ ] **Step 4: Run the full suite + build**

Run: `cd apps/web && npx vitest run`
Expected: PASS — all prior suites + new terminal suites green.

Run: `cd apps/web && npm run build`
Expected: SUCCESS — `/terminal` appears in the route list; no `/orders`. If `next build`'s tsc surfaces a type error not caught by vitest, fix it type-correctly and re-run.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/(app)/terminal/page.tsx apps/web/components/shell/nav.ts
git commit -m "feat(terminal): wire /terminal page with RSC prefetch + rename Emirler nav"
```

---

## Task 15: Living docs + visual verification

**Files:**
- Modify: `docs/progress.md`, `docs/superpowers/specs/2026-08-03-sentinel-trading-terminal-design.md`, `docs/superpowers/followups-frontend.md`
- Modify (memory): `sentinel-frontend-stack-and-plan.md`

- [ ] **Step 1: Visual check**

Run `cd apps/web && npm run dev`, open `/terminal`. Verify: left token list selects active token (updates center + right); MarketDataHeader shows price/stats/score badges; candlestick chart renders (lightweight-charts); OrderPanel fields + simulation status; invalid amount → error + Önizle disabled; Önizle → confirmation dialog (token/side/amount/impact/scores); Onayla in paper mode → simulate toast; switching TradingMode to live (header badge) → Onayla → `toast.warning`; bottom tabs switch across Pozisyonlar (reused table) / Emirler (cancel toast) / İşlemler (explorer link) / Loglar. If the browser extension is unavailable, note it and proceed with docs.

- [ ] **Step 2: Update living docs**

- `docs/progress.md`: set Increment 8 row to ✅ (branch complete, merge awaiting); add a decision-log entry dated 2026-08-03 summarizing the terminal (route rename, simulated stateless orders, lightweight-charts, Dialog primitive, controlled form); update "Sırada" to Increment 9 (Backtesting + Event Replay); bump "Son güncelleme".
- Spec `Durum:` → `Uygulandı (2026-08-03) — branch ..., N/N test, build + review temiz; merge onayı bekliyor.`
- `docs/superpowers/followups-frontend.md`: add a "Trading Terminal (Increment 8)" section for any deferred minors surfaced during review (e.g. lightweight-charts v4→v5 upgrade note, no ascending sort in terminal positions tab, wallet balance is a constant, order form has no RHF/Zod).
- Memory `sentinel-frontend-stack-and-plan.md`: add the Increment 8 entry + update completed-screens + next-increments line.

- [ ] **Step 3: Commit docs**

```bash
git add docs/progress.md docs/superpowers/specs/2026-08-03-sentinel-trading-terminal-design.md docs/superpowers/followups-frontend.md
git commit -m "docs(terminal): mark Increment 8 complete + followups"
```

---

## Self-Review (completed during authoring)

**Spec coverage:** §1 routes/rename → Tasks 14; §3.1 seam → Task 1; §3.2 config+logic → Task 2; §3.3 components → Tasks 3–13; §3.4 state → Task 13; §3.5 order flow/safety → Tasks 7–8; §3.6 page → Task 14; §5 tests → each task's TDD steps; §6 acceptance → Tasks 4–14 + Task 15 visual. Dialog primitive (§2 reuse) → Task 3. All spec sections map to a task.

**Placeholder scan:** No TBD/TODO; every code step contains real code; test bodies are concrete.

**Type consistency:** `OrderDraft`/`MarketData`/`Order`/`Candle` names identical across Tasks 1–13. `simulateOrder`/`validateOrder` signatures match between Task 2 (definition) and Tasks 7/8 (consumers). `SortKey` reused verbatim from `PositionsTable` in Task 12. `DEFAULT_TERMINAL_MINT` defined in Task 2, consumed in Tasks 13–14. `qk.*` keys defined in Task 1 match page prefetch in Task 14.

**Known integration risks (flagged for implementers):**
- lightweight-charts **must** be v4 (`addCandlestickSeries` + `autoSize`); v5 renamed the series API. Pinned in Task 6 Step 1 and mirrored in the test mock.
- Base-UI `Dialog` popup content unmounts when `open={false}` — Task 3 Step 4 notes the closed-case assertion depends on this.
- `next build` tsc may catch type errors vitest misses (see Increment 7 precedent) — Task 14 Step 4 handles it explicitly.
