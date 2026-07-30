import { render, screen } from "@testing-library/react";
import { Sidebar } from "./Sidebar";
vi.mock("next/navigation", () => ({ usePathname: () => "/tokens" }));

test("Sidebar renders all nav items and marks active", () => {
  render(<Sidebar />);
  expect(screen.getByText("Genel Bakış")).toBeInTheDocument();
  const tokens = screen.getByText("Tokenlar").closest("a")!;
  expect(tokens.className).toContain("bg-sidebar-accent");
});
