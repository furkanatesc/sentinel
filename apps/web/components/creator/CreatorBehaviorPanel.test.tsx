import { render, screen } from "@testing-library/react";
import { CreatorBehaviorPanel } from "./CreatorBehaviorPanel";
test("renders behavior fields", () => {
  render(<CreatorBehaviorPanel behavior={{ deployFrequency: "12 token / 30 gün", avgFirstSellMinutes: 8, repeatedFunders: ["Fnd1...aa"], similarMetadata: true, sameSocial: false, sameLiquidityPattern: true }} />);
  expect(screen.getByText("12 token / 30 gün")).toBeInTheDocument();
  expect(screen.getByText("Benzer metadata")).toBeInTheDocument();
});
