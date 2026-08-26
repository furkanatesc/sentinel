import { NODE_TYPE_DEFS, EDGE_TYPE_DEFS } from "./graph-defs";

test("registries cover all 9 node + 9 edge types with label/color", () => {
  expect(NODE_TYPE_DEFS).toHaveLength(9);
  expect(EDGE_TYPE_DEFS).toHaveLength(9);
  for (const d of [...NODE_TYPE_DEFS, ...EDGE_TYPE_DEFS]) { expect(d.label).toBeTruthy(); expect(d.color).toMatch(/^#/); }
});
