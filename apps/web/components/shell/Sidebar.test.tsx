import { render, screen } from "@testing-library/react";
import { Sidebar } from "./Sidebar";
vi.mock("next/navigation", () => ({ usePathname: () => "/tokens" }));

test("Sidebar renders all nav items and marks active", () => {
  render(<Sidebar />);
  expect(screen.getByText("Overview")).toBeInTheDocument();
  const tokens = screen.getByText("Tokens").closest("a")!;
  expect(tokens.className).toContain("bg-sidebar-accent");
});
