import { render } from "@testing-library/react";
import { EquityCurve } from "./EquityCurve";
import type { EquityPoint } from "@/lib/api/types";

test("renders without crashing given equity points (smoke)", () => {
  const data: EquityPoint[] = Array.from({ length: 10 }, (_, i) => ({ t: i, v: 100 + i }));
  const { container } = render(
    <div style={{ width: 400, height: 200 }}>
      <EquityCurve data={data} />
    </div>
  );
  expect(container.querySelector(".recharts-responsive-container")).toBeTruthy();
});
