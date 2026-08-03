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
