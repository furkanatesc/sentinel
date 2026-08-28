import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useResearch } from "./use-research";
import { useResearchStore } from "@/lib/store/research";

beforeEach(() => { useResearchStore.getState().reset(); vi.useFakeTimers(); });
afterEach(() => vi.useRealTimers());

describe("useResearch", () => {
  it("send streams an answer into the store", () => {
    const { result } = renderHook(() => useResearch());
    act(() => result.current.send("GFROG neden riskli?"));
    expect(result.current.isStreaming).toBe(true);
    act(() => vi.runAllTimers());
    const last = result.current.messages.at(-1)!;
    expect(last.role).toBe("assistant");
    expect(last.status).toBe("done");
    expect(last.text.length).toBeGreaterThan(0);
    expect(result.current.isStreaming).toBe(false);
  });

  it("send is ignored while already streaming", () => {
    const { result } = renderHook(() => useResearch());
    act(() => result.current.send("GFROG neden riskli?"));
    act(() => result.current.send("ikinci soru")); // should be ignored
    const userMsgs = result.current.messages.filter((m) => m.role === "user");
    expect(userMsgs).toHaveLength(1);
  });

  it("stop halts streaming and marks the answer done", () => {
    const { result } = renderHook(() => useResearch());
    act(() => result.current.send("GFROG neden riskli?"));
    act(() => vi.advanceTimersByTime(35));
    act(() => result.current.stop());
    expect(result.current.isStreaming).toBe(false);
    expect(result.current.messages.at(-1)!.status).toBe("done");
  });
});
