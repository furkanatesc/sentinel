import { describe, it, expect } from "vitest";
import { validateAlertRule, ALERT_TRIGGER_DEFS, DELIVERY_CHANNEL_DEFS, type AlertRuleDraft } from "./alert-defs";

const valid: AlertRuleDraft = {
  name: "X", trigger: "new_mint", scope: "Tüm tokenlar",
  minLiquidity: 0, minCreatorScore: 50, maxRisk: "medium", channels: ["slack"],
};

describe("validateAlertRule", () => {
  it("geçerli taslak → hata yok", () => {
    expect(validateAlertRule(valid)).toEqual([]);
  });
  it("boş isim → name hatası", () => {
    expect(validateAlertRule({ ...valid, name: " " }).some((e) => e.field === "name")).toBe(true);
  });
  it("negatif likidite → minLiquidity hatası", () => {
    expect(validateAlertRule({ ...valid, minLiquidity: -1 }).some((e) => e.field === "minLiquidity")).toBe(true);
  });
  it("skor 0-100 dışı → minCreatorScore hatası", () => {
    expect(validateAlertRule({ ...valid, minCreatorScore: 150 }).some((e) => e.field === "minCreatorScore")).toBe(true);
  });
  it("kanal yok → channels hatası", () => {
    expect(validateAlertRule({ ...valid, channels: [] }).some((e) => e.field === "channels")).toBe(true);
  });
});

describe("registries", () => {
  it("her trigger etiketli", () => {
    expect(ALERT_TRIGGER_DEFS.new_mint.label).toBeTruthy();
    expect(Object.keys(DELIVERY_CHANNEL_DEFS)).toContain("slack");
  });
});
