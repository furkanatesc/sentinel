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
