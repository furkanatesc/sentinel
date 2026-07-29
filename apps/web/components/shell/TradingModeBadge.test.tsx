import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TradingModeBadge } from "./TradingModeBadge";
import { useSessionStore } from "@/lib/store/session";

test("selecting Live updates the session store", async () => {
  useSessionStore.setState({ tradingMode: "paper" });
  render(<TradingModeBadge />);
  await userEvent.click(screen.getByTitle("Trading mode"));
  await userEvent.click(screen.getByText("Live"));
  expect(useSessionStore.getState().tradingMode).toBe("live");
});
