import { describe, it, expect } from "vitest";
import { pickAnswer, RESEARCH_SUGGESTIONS } from "./match";

describe("pickAnswer", () => {
  it("routes risk questions to a risk answer with sources", () => {
    const a = pickAnswer("Bu token neden riskli?");
    expect(a.text.toLowerCase()).toContain("risk");
    expect(a.sources.length).toBeGreaterThan(0);
  });
  it("routes creator-history questions", () => {
    const a = pickAnswer("Üreticinin geçmişi nasıl?");
    expect(a.text).toBeTruthy();
    expect(a.sources.some((s) => s.kind === "creator")).toBe(true);
  });
  it("routes wallet-cluster questions", () => {
    const a = pickAnswer("Cüzdan kümesi şüpheli mi?");
    expect(a.sources.some((s) => s.kind === "wallet")).toBe(true);
  });
  it("falls back for unknown questions", () => {
    const a = pickAnswer("qwerty zxcv unknown");
    expect(a.text).toBeTruthy();
    expect(a.sources).toEqual([]);
  });
  it("is case/accent tolerant (matches on lowercase keywords)", () => {
    expect(pickAnswer("NEDEN RİSKLİ").text).toBe(pickAnswer("neden riskli").text);
  });
});

describe("RESEARCH_SUGGESTIONS", () => {
  it("has several starter questions with unique ids", () => {
    expect(RESEARCH_SUGGESTIONS.length).toBeGreaterThanOrEqual(4);
    expect(new Set(RESEARCH_SUGGESTIONS.map((s) => s.id)).size).toBe(RESEARCH_SUGGESTIONS.length);
  });
});
