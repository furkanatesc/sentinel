import { mockApi } from "./mock";
import { SCORE_DEFS } from "@/lib/token/score-defs";

test("getToken returns a full detail for a known symbol", async () => {
  const d = await mockApi.getToken("PULSE");
  expect(d.symbol).toBe("PULSE");
  for (const { key } of SCORE_DEFS) {
    expect(d.scores[key].value).toBeGreaterThanOrEqual(0);
    expect(d.scores[key].value).toBeLessThanOrEqual(100);
    expect(d.scores[key].breakdown.length).toBeGreaterThan(0);
  }
  expect(d.series.price.length).toBeGreaterThan(0);
  expect(d.risks.contract.length + d.risks.market.length + d.risks.creator.length).toBeGreaterThan(0);
});

test("getToken is case-insensitive and rejects unknown", async () => {
  expect((await mockApi.getToken("pulse")).symbol).toBe("PULSE");
  await expect(mockApi.getToken("NOPE")).rejects.toThrow();
});
