import { render, screen } from "@testing-library/react";
import { NodeDetailPanel } from "./NodeDetailPanel";
import type { WalletGraph, GraphNode } from "@/lib/api/types";
const graph: WalletGraph = { nodes: [], edges: [{ id: "e", source: "T1", target: "C1", type: "created" }] };
const token: GraphNode = { id: "T1", type: "token", label: "PULSE", riskLevel: "good", firstSeen: "x", lastSeen: "y" };
test("empty state when no node", () => {
  render(<NodeDetailPanel node={null} graph={graph} />);
  expect(screen.getByText("Detay için bir düğüm seç")).toBeInTheDocument();
});
test("token node shows type and token link", () => {
  render(<NodeDetailPanel node={token} graph={graph} />);
  expect(screen.getByText("Token")).toBeInTheDocument();
  expect(screen.getByText("Token Detayına Git").closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
});
