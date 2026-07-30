import { render, screen } from "@testing-library/react";
import { Button } from "@/components/ui/button";

test("Button renders its children", () => {
  render(<Button>Trade</Button>);
  expect(screen.getByRole("button", { name: "Trade" })).toBeInTheDocument();
});
