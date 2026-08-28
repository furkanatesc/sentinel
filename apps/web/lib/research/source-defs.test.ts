import { describe, it, expect } from "vitest";
import { hrefForSource, SOURCE_KIND_DEFS } from "./source-defs";

describe("SOURCE_KIND_DEFS / hrefForSource", () => {
  it("token → /tokens/[ref]", () => {
    expect(hrefForSource({ id: "s1", kind: "token", label: "AERO", ref: "AERO" })).toBe("/tokens/AERO");
  });
  it("creator → /creators/[ref]", () => {
    expect(hrefForSource({ id: "s2", kind: "creator", label: "6Rt4", ref: "6Rt4abc" })).toBe("/creators/6Rt4abc");
  });
  it("wallet → /wallet-graph", () => {
    expect(hrefForSource({ id: "s3", kind: "wallet", label: "9kQ", ref: "9kQxyz" })).toBe("/wallet-graph");
  });
  it("tx / risk-rule / timestamp have no href", () => {
    expect(hrefForSource({ id: "s4", kind: "tx", label: "5xA…" })).toBeUndefined();
    expect(hrefForSource({ id: "s5", kind: "risk-rule", label: "rug-pull-rule" })).toBeUndefined();
    expect(hrefForSource({ id: "s6", kind: "timestamp", label: "12:04" })).toBeUndefined();
  });
  it("every kind has a def with label + icon", () => {
    (["token","creator","wallet","tx","risk-rule","strategy","timestamp"] as const).forEach((k) => {
      expect(SOURCE_KIND_DEFS[k].label).toBeTruthy();
      expect(SOURCE_KIND_DEFS[k].icon).toBeTruthy();
    });
  });
  it("token/creator without ref have no href", () => {
    expect(hrefForSource({ id: "s7", kind: "token", label: "AERO" })).toBeUndefined();
    expect(hrefForSource({ id: "s8", kind: "creator", label: "6Rt4" })).toBeUndefined();
  });
});
