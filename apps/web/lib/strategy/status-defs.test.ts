import { STATUS_DEFS, CONDITION_LABELS, formatCondition } from "./status-defs";
import type { StrategyStatus } from "@/lib/api/types";

test("every status has a label and hex color", () => {
  const statuses: StrategyStatus[] = ["draft", "backtesting", "paper", "shadow", "live", "paused", "archived"];
  for (const s of statuses) {
    expect(STATUS_DEFS[s].label).toBeTruthy();
    expect(STATUS_DEFS[s].color).toMatch(/^#/);
  }
});

test("CONDITION_LABELS maps known metrics to Turkish labels", () => {
  expect(CONDITION_LABELS.creatorScore).toBe("Creator Skoru");
  expect(CONDITION_LABELS.tokenSafety).toBe("Token Güvenliği");
});

test("formatCondition renders label, operator, value and unit", () => {
  expect(formatCondition({ metric: "creatorScore", op: ">", value: 75 })).toBe("Creator Skoru > 75");
  expect(formatCondition({ metric: "liquidity", op: ">", value: 25000, unit: "USD" })).toBe("Likidite > 25000 USD");
});

test("formatCondition falls back to the raw metric key when unmapped", () => {
  expect(formatCondition({ metric: "unknownMetric", op: "<", value: 5 })).toBe("unknownMetric < 5");
});
