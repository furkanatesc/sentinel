import { buildStylesheet } from "./stylesheet";

test("stylesheet has a selector per node and edge type", () => {
  const s = buildStylesheet();
  expect(s.some((x) => x.selector === "node.creator_wallet")).toBe(true);
  expect(s.some((x) => x.selector === "edge.shares_funder")).toBe(true);
  expect(s.some((x) => x.selector === ".faded")).toBe(true);
});
test("stylesheet has a border-color selector per risk level", () => {
  const s = buildStylesheet();
  expect(s.some((x) => x.selector === "node.risk-critical")).toBe(true);
});
