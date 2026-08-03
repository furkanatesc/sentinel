import { validateParams } from "./validate";
import { DEFAULT_BACKTEST_PARAMS } from "./backtest-defs";

test("default params validate cleanly", () => {
  expect(validateParams(DEFAULT_BACKTEST_PARAMS)).toEqual({});
});

test("invalid capital / positions / score are rejected", () => {
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, initialCapitalSol: 0 }).initialCapitalSol).toBeTruthy();
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, maxPositions: 0 }).maxPositions).toBeTruthy();
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, minCreatorScore: 120 }).minCreatorScore).toBeTruthy();
  expect(validateParams({ ...DEFAULT_BACKTEST_PARAMS, strategyId: "" }).strategyId).toBeTruthy();
});
