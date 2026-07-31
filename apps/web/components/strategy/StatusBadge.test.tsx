import { render, screen } from "@testing-library/react";
import { StatusBadge } from "./StatusBadge";

test("renders the Turkish label for a status", () => {
  render(<StatusBadge status="live" />);
  expect(screen.getByText("Canlı")).toBeInTheDocument();
});
