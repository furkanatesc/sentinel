import { render, screen } from "@testing-library/react";
import { GraphLegend } from "./GraphLegend";
test("legend lists node and edge types", () => {
  render(<GraphLegend />);
  expect(screen.getByText("Creator Cüzdanı")).toBeInTheDocument();
  expect(screen.getByText("Ortak Fonlayıcı")).toBeInTheDocument();
});
