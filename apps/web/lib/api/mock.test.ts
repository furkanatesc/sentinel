import { mockApi } from "./mock";

test("mockApi returns seed collections", async () => {
  expect((await mockApi.getKpis()).length).toBe(8);
  expect((await mockApi.getTokens()).length).toBeGreaterThan(0);
  expect((await mockApi.getAlerts()).length).toBeGreaterThan(0);
  expect((await mockApi.getRadar()).length).toBe((await mockApi.getTokens()).length);
});

test("getAlertRules kurallar döndürür", async () => {
  const rules = await mockApi.getAlertRules();
  expect(Array.isArray(rules)).toBe(true);
  expect(rules.length).toBeGreaterThan(0);
  expect(rules[0]).toHaveProperty("trigger");
  expect(Array.isArray(rules[0].channels)).toBe(true);
});

test("getNotificationConfig Slack config döndürür", async () => {
  const c = await mockApi.getNotificationConfig();
  expect(c.channel).toMatch(/^#/);
  expect(["connected", "disconnected", "error"]).toContain(c.slackState);
  expect(c).toHaveProperty("quietHours");
});

test("createAlertRule + setAlertRuleEnabled diziyi günceller", async () => {
  const before = (await mockApi.getAlertRules()).length;
  const created = await mockApi.createAlertRule({
    name: "Test kuralı", trigger: "new_mint", scope: "Tüm", minLiquidity: 0, minCreatorScore: 0, maxRisk: "medium", channels: ["slack"],
  });
  expect(created.id).toBeTruthy();
  expect(created.enabled).toBe(true);
  const after = await mockApi.getAlertRules();
  expect(after.length).toBe(before + 1);
  // toggle
  await mockApi.setAlertRuleEnabled(created.id, false);
  const toggled = (await mockApi.getAlertRules()).find((r) => r.id === created.id);
  expect(toggled?.enabled).toBe(false);
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

test("getSystemHealth returns workers + gates", async () => {
  const h = await mockApi.getSystemHealth();
  expect(Array.isArray(h.workers)).toBe(true);
  expect(h.workers.length).toBeGreaterThan(0);
  expect(typeof h.dbOk).toBe("boolean");
  expect(h.gates).toHaveProperty("SAFETY_ENABLED");
});
