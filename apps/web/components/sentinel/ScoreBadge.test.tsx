import { render, screen } from "@testing-library/react";
import { ScoreBadge } from "./ScoreBadge";

test("ScoreBadge shows number and level label", () => {
  render(<ScoreBadge score={17} />);
  expect(screen.getByText("17")).toBeInTheDocument();
  expect(screen.getByText("Critical")).toBeInTheDocument();
});

test("ScoreBadge maps 88 to Strong", () => {
  render(<ScoreBadge score={88} />);
  expect(screen.getByText("Strong")).toBeInTheDocument();
});
