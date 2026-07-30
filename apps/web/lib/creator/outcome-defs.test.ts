import { OUTCOME_DEFS, LIQUIDITY_DEFS } from "./outcome-defs";
test("every outcome and liquidity status has label+color", () => {
  for (const o of ["active", "graduated", "dumped", "rug", "dead"] as const) {
    expect(OUTCOME_DEFS[o].label).toBeTruthy(); expect(OUTCOME_DEFS[o].color).toMatch(/^#/);
  }
  for (const l of ["locked", "unlocked", "removed"] as const) {
    expect(LIQUIDITY_DEFS[l].label).toBeTruthy(); expect(LIQUIDITY_DEFS[l].color).toMatch(/^#/);
  }
});
