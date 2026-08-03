import { render } from "@testing-library/react";
import { PnlByStrategyChart } from "./PnlByStrategyChart";
import { RiskAllocationChart } from "./RiskAllocationChart";
import { WinLossChart } from "./WinLossChart";

const wrap = (ui: React.ReactNode) => render(<div style={{ width: 400, height: 240 }}>{ui}</div>);

test("PnL-by-strategy chart renders (smoke)", () => {
  const { container } = wrap(<PnlByStrategyChart data={[{ strategyId: "s1", name: "S1", pnlSol: 12 }, { strategyId: "s2", name: "S2", pnlSol: -4 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});

test("risk-allocation chart renders (smoke)", () => {
  const { container } = wrap(<RiskAllocationChart data={[{ label: "Güçlü", pct: 60, color: "#2FD98B" }, { label: "Orta", pct: 40, color: "#3E9BFF" }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});

test("win/loss chart renders (smoke)", () => {
  const { container } = wrap(<WinLossChart data={[{ label: "Kazanç", count: 10 }, { label: "Kayıp", count: 4 }]} />);
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
