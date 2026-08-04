import { render } from "@testing-library/react";
import { EntryExitChart } from "./EntryExitChart";

test("entry/exit chart renders price + markers (smoke)", () => {
  const { container } = render(
    <div style={{ width: 400, height: 240 }}>
      <EntryExitChart
        priceSeries={[{ t: 0, v: 10 }, { t: 1, v: 12 }, { t: 2, v: 9 }]}
        trades={[{ time: 1, price: 12, side: "buy", pnlSol: 2 }, { time: 2, price: 9, side: "sell", pnlSol: -1 }]}
      />
    </div>
  );
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
