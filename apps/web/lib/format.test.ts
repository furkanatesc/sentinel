import { scoreToLevel, formatAge, formatPrice, formatUsd, riskMeta, riskSeverityMeta, formatPct } from "./format";

test("scoreToLevel boundaries", () => {
  expect(scoreToLevel(0)).toBe("critical");
  expect(scoreToLevel(24)).toBe("critical");
  expect(scoreToLevel(25)).toBe("high");
  expect(scoreToLevel(49)).toBe("high");
  expect(scoreToLevel(50)).toBe("medium");
  expect(scoreToLevel(69)).toBe("medium");
  expect(scoreToLevel(70)).toBe("good");
  expect(scoreToLevel(84)).toBe("good");
  expect(scoreToLevel(85)).toBe("strong");
  expect(scoreToLevel(100)).toBe("strong");
});

test("riskMeta has label+color for every level", () => {
  for (const lvl of ["critical", "high", "medium", "good", "strong"] as const) {
    expect(riskMeta[lvl].label).toBeTruthy();
    expect(riskMeta[lvl].color).toMatch(/^#/);
  }
});

test("formatAge seconds and minutes", () => {
  expect(formatAge(38)).toBe("38sn");
  expect(formatAge(95)).toBe("1dk 35sn");
});

test("formatPrice tiers", () => {
  expect(formatPrice(1.5)).toBe("$1.50");
  expect(formatPrice(0.019)).toBe("$0.0190");
  expect(formatPrice(0.0000009)).toBe("$9.0e-7");
});

test("formatUsd tiers", () => {
  expect(formatUsd(320000)).toBe("$320.0K");
  expect(formatUsd(1_500_000)).toBe("$1.5M");
  expect(formatUsd(940)).toBe("$940");
});

test("riskSeverityMeta covers every severity with hex color", () => {
  for (const s of ["critical", "high", "medium", "info"] as const) {
    expect(riskSeverityMeta[s].label).toBeTruthy();
    expect(riskSeverityMeta[s].color).toMatch(/^#/);
  }
});

test("formatPct", () => {
  expect(formatPct(12.34)).toBe("%12.3");
  expect(formatPct(100)).toBe("%100.0");
});
