import { render, screen } from "@testing-library/react";
import { WalletGraphCanvas } from "./WalletGraphCanvas";
const on = vi.fn();
vi.mock("cytoscape", () => ({ default: vi.fn(() => ({ on, destroy: vi.fn() })) }));
test("mounts the canvas container and inits cytoscape", async () => {
  const cytoscape = (await import("cytoscape")).default as unknown as ReturnType<typeof vi.fn>;
  render(<WalletGraphCanvas elements={[{ data: { id: "a" } }]} stylesheet={[]} onNodeSelect={() => {}} />);
  expect(screen.getByTestId("graph-canvas")).toBeInTheDocument();
  expect(cytoscape).toHaveBeenCalled();
});
