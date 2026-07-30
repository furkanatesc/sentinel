import { create } from "zustand";

export type TradingMode = "paper" | "shadow" | "live";

interface SessionState {
  tradingMode: TradingMode;
  setTradingMode: (m: TradingMode) => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  tradingMode: "paper",
  setTradingMode: (m) => set({ tradingMode: m }),
}));
