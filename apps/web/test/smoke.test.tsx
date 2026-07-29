import { render, screen } from "@testing-library/react";
import Home from "@/app/page";

test("home renders Sentinel heading", () => {
  render(<Home />);
  expect(screen.getByRole("heading", { name: "Sentinel" })).toBeInTheDocument();
});
