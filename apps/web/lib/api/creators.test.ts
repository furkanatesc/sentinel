import { mockApi } from "./mock";

test("getCreators returns rows", async () => {
  const rows = await mockApi.getCreators();
  expect(rows.length).toBeGreaterThan(3);
  expect(rows[0].address).toBeTruthy();
});

test("getCreator returns a full profile deterministically", async () => {
  const a = await mockApi.getCreator("CreAxz");
  expect(a.address).toBe("CreAxz");
  expect(a.reputation.key).toBe("creatorReputation");
  expect(a.reputation.breakdown.length).toBeGreaterThan(0);
  expect(a.history.length).toBeGreaterThan(0);
  const b = await mockApi.getCreator("CreAxz");
  expect(b.reputation.value).toBe(a.reputation.value); // deterministic
});
