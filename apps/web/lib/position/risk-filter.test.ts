import { POSITION_RISK_LEVELS, pnlColor } from "./risk-filter";

test("POSITION_RISK_LEVELS covers the five risk levels", () => {
  expect(POSITION_RISK_LEVELS).toEqual(["strong", "good", "medium", "high", "critical"]);
});

test("pnlColor is green for >=0 and red for <0", () => {
  expect(pnlColor(0)).toBe("#2FD98B");
  expect(pnlColor(5)).toBe("#2FD98B");
  expect(pnlColor(-1)).toBe("#F0476B");
});
