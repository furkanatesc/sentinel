import { render, screen, fireEvent, act } from "@testing-library/react";
import { vi } from "vitest";
import { PositionActions } from "./PositionActions";
import { useSessionStore } from "@/lib/store/session";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { warning: vi.fn() }) }));
import { toast } from "sonner";

test("live mode close fires toast.warning; paper mode fires toast", () => {
  act(() => useSessionStore.getState().setTradingMode("live"));
  render(<PositionActions symbol="PULSE" />);
  fireEvent.click(screen.getByRole("button", { name: "Kapat" }));
  expect((toast as unknown as { warning: ReturnType<typeof vi.fn> }).warning).toHaveBeenCalled();

  act(() => useSessionStore.getState().setTradingMode("paper"));
  fireEvent.click(screen.getByRole("button", { name: "SL/TP" }));
  expect(toast).toHaveBeenCalled();
});
