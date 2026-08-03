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
