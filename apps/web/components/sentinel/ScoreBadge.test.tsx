import { render, screen } from "@testing-library/react";
import { ScoreBadge } from "./ScoreBadge";

test("ScoreBadge shows number and level label", () => {
  render(<ScoreBadge score={17} />);
  expect(screen.getByText("17")).toBeInTheDocument();
  expect(screen.getByText("Kritik")).toBeInTheDocument();
});

test("ScoreBadge maps 88 to Strong", () => {
  render(<ScoreBadge score={88} />);
  expect(screen.getByText("Güçlü")).toBeInTheDocument();
});

test("ScoreBadge score 0 dürüst nötr gösterir (fake '0' değil)", () => {
  render(<ScoreBadge score={0} />);
  expect(screen.getByText("—")).toBeInTheDocument();
  expect(screen.queryByText("0")).not.toBeInTheDocument();
  expect(screen.queryByText("Kritik")).not.toBeInTheDocument();
});
