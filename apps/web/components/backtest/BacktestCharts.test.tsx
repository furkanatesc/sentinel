import { render } from "@testing-library/react";
import { DrawdownChart } from "./DrawdownChart";
import { MonthlyReturnChart } from "./MonthlyReturnChart";
import { TradeDistributionChart } from "./TradeDistributionChart";
import { PnlByScoreChart } from "./PnlByScoreChart";

const wrap = (ui: React.ReactNode) => render(<div style={{ width: 400, height: 240 }}>{ui}</div>);

test("drawdown chart renders (smoke)", () => {
  const { container } = wrap(<DrawdownChart data={[{ t: 0, v: 0 }, { t: 1, v: -5 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
test("monthly return chart renders (smoke)", () => {
  const { container } = wrap(<MonthlyReturnChart data={[{ label: "Oca", pct: 8 }, { label: "Şub", pct: -3 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
test("trade distribution chart renders (smoke)", () => {
  const { container } = wrap(<TradeDistributionChart data={[{ label: "0..5", count: 12 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
test("pnl-by-score chart renders (smoke)", () => {
  const { container } = wrap(<PnlByScoreChart data={[{ scoreBucket: "70-84", pnlSol: 20 }, { scoreBucket: "0-24", pnlSol: -10 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
