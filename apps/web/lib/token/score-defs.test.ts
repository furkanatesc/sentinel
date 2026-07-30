import { SCORE_DEFS, scoreDisplayLevel } from "./score-defs";

test("SCORE_DEFS has the 4 scores with correct polarity", () => {
  const byKey = Object.fromEntries(SCORE_DEFS.map((d) => [d.key, d]));
  expect(SCORE_DEFS).toHaveLength(4);
  expect(byKey.opportunity.higherIsBetter).toBe(true);
  expect(byKey.creatorReputation.higherIsBetter).toBe(true);
  expect(byKey.tokenSafety.higherIsBetter).toBe(true);
  expect(byKey.manipulationRisk.higherIsBetter).toBe(false);
});

test("scoreDisplayLevel inverts when higher is worse", () => {
  expect(scoreDisplayLevel(90, true)).toBe("strong");   // yüksek iyi
  expect(scoreDisplayLevel(90, false)).toBe("critical"); // yüksek manipülasyon = kötü
  expect(scoreDisplayLevel(10, false)).toBe("strong");   // düşük manipülasyon = iyi
});
