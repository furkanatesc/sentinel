import { mockApi } from "./mock";

test("getEvents returns a seed stream", async () => {
  const evs = await mockApi.getEvents();
  expect(evs.length).toBeGreaterThan(10);
  expect(evs[0].type).toBeTruthy();
});

test("subscribeEvents emits and unsubscribes", async () => {
  await new Promise<void>((resolve, reject) => {
    const stop = mockApi.subscribeEvents((e) => {
      expect(e.id).toBeTruthy();
      stop();
      resolve();
    });
    expect(typeof stop).toBe("function");
    setTimeout(() => reject(new Error("no emit")), 5000);
  });
});
