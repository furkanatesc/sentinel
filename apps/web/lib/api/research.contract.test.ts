import { describe, it, expect, vi } from "vitest";
import { httpApi } from "./http";

describe("research seam — http not ready", () => {
  it("getResearchSuggestions rejects (backend not connected)", async () => {
    await expect(httpApi.getResearchSuggestions()).rejects.toThrow(/not implemented/i);
  });

  it("streamResearchAnswer throws synchronously (backend not connected)", () => {
    expect(() => httpApi.streamResearchAnswer("q", vi.fn(), vi.fn())).toThrow(/not implemented/i);
  });
});
