import { render, screen } from "@testing-library/react";
import { EventTypeBadge } from "./EventTypeBadge";

test("renders the Turkish label for the type", () => {
  render(<EventTypeBadge type="liquidity_removed" />);
  expect(screen.getByText("Likidite Çekildi")).toBeInTheDocument();
});
