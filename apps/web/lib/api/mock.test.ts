import { mockApi } from "./mock";

test("mockApi returns seed collections", async () => {
  expect((await mockApi.getKpis()).length).toBe(8);
  expect((await mockApi.getTokens()).length).toBeGreaterThan(0);
  expect((await mockApi.getAlerts()).length).toBeGreaterThan(0);
  expect((await mockApi.getRadar()).length).toBe((await mockApi.getTokens()).length);
});

test("subscribeTokens emits and returns an unsubscribe fn", async () => {
  await new Promise<void>((resolve, reject) => {
    const stop = mockApi.subscribeTokens((tokens) => {
      expect(Array.isArray(tokens)).toBe(true);
      stop();
      resolve();
    });
    expect(typeof stop).toBe("function");
    setTimeout(() => reject(new Error("no emit")), 4000);
  });
});
