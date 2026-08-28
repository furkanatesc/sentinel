import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mockApi } from "./mock";

describe("mockApi.getResearchSuggestions", () => {
  it("returns the suggestion list", async () => {
    const list = await mockApi.getResearchSuggestions();
    expect(list.length).toBeGreaterThanOrEqual(4);
    expect(list[0]).toHaveProperty("text");
  });
});

describe("mockApi.streamResearchAnswer", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("streams chunks then calls onDone with full text + sources", () => {
    const onChunk = vi.fn();
    const onDone = vi.fn();
    mockApi.streamResearchAnswer("GFROG neden riskli?", onChunk, onDone);
    vi.runAllTimers();
    expect(onChunk.mock.calls.length).toBeGreaterThan(1);
    expect(onDone).toHaveBeenCalledTimes(1);
    const answer = onDone.mock.calls[0][0];
    const streamed = onChunk.mock.calls.map((c) => c[0]).join("");
    expect(streamed).toBe(answer.text);
    expect(answer.sources.length).toBeGreaterThan(0);
  });

  it("cancel fn stops streaming (no onDone)", () => {
    const onChunk = vi.fn();
    const onDone = vi.fn();
    const cancel = mockApi.streamResearchAnswer("GFROG neden riskli?", onChunk, onDone);
    vi.advanceTimersByTime(35); // one chunk
    cancel();
    vi.runAllTimers();
    expect(onDone).not.toHaveBeenCalled();
  });
});
