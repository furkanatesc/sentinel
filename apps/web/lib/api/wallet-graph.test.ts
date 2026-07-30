import { mockApi } from "./mock";

test("getWalletGraph returns connected nodes and edges", async () => {
  const g = await mockApi.getWalletGraph();
  expect(g.nodes.length).toBeGreaterThan(10);
  expect(g.edges.length).toBeGreaterThan(10);
  const ids = new Set(g.nodes.map((n) => n.id));
  for (const e of g.edges) {
    expect(ids.has(e.source)).toBe(true);
    expect(ids.has(e.target)).toBe(true);
  }
  expect(g.nodes.some((n) => n.type === "token")).toBe(true);
});
