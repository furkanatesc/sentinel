import { EVENT_TYPE_DEFS, EVENT_SEVERITY } from "./event-defs";

test("every event type has a def with label + severity", () => {
  expect(EVENT_TYPE_DEFS).toHaveLength(11);
  for (const d of EVENT_TYPE_DEFS) {
    expect(d.label).toBeTruthy();
    expect(EVENT_SEVERITY[d.key]).toBeTruthy();
  }
});
