import { useUiStore } from "./ui";
import { useSessionStore } from "./session";

test("ui store toggles sidebar", () => {
  expect(useUiStore.getState().sidebarCollapsed).toBe(false);
  useUiStore.getState().toggleSidebar();
  expect(useUiStore.getState().sidebarCollapsed).toBe(true);
});

test("session store sets trading mode", () => {
  useSessionStore.getState().setTradingMode("live");
  expect(useSessionStore.getState().tradingMode).toBe("live");
});
