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
