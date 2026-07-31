import { render, screen } from "@testing-library/react";
import { ConditionList } from "./ConditionList";

test("renders the title and each condition via formatCondition", () => {
  render(
    <ConditionList
      title="Giriş (IF)"
      conditions={[
        { metric: "creatorScore", op: ">", value: 75 },
        { metric: "liquidity", op: ">", value: 25000, unit: "USD" },
      ]}
    />
  );
  expect(screen.getByText("Giriş (IF)")).toBeInTheDocument();
  expect(screen.getByText("Creator Skoru > 75")).toBeInTheDocument();
  expect(screen.getByText("Likidite > 25000 USD")).toBeInTheDocument();
});
