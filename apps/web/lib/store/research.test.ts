import { describe, it, expect, beforeEach } from "vitest";
import { useResearchStore } from "./research";

const reset = () => useResearchStore.getState().reset();

describe("research store", () => {
  beforeEach(reset);

  it("addUserMessage adds user + streaming assistant placeholder", () => {
    const id = useResearchStore.getState().addUserMessage("merhaba");
    const { messages, isStreaming } = useResearchStore.getState();
    expect(messages).toHaveLength(2);
    expect(messages[0]).toMatchObject({ role: "user", text: "merhaba" });
    expect(messages[1]).toMatchObject({ role: "assistant", text: "", status: "streaming", id });
    expect(isStreaming).toBe(true);
  });

  it("appendChunk accumulates assistant text", () => {
    const id = useResearchStore.getState().addUserMessage("q");
    useResearchStore.getState().appendChunk(id, "Merhaba");
    useResearchStore.getState().appendChunk(id, " dünya");
    expect(useResearchStore.getState().messages[1].text).toBe("Merhaba dünya");
  });

  it("finishAnswer attaches sources, marks done, clears isStreaming", () => {
    const id = useResearchStore.getState().addUserMessage("q");
    useResearchStore.getState().appendChunk(id, "cevap");
    useResearchStore.getState().finishAnswer(id, [{ id: "s1", kind: "token", label: "AERO", ref: "AERO" }]);
    const msg = useResearchStore.getState().messages[1];
    expect(msg.status).toBe("done");
    expect(msg.sources).toHaveLength(1);
    expect(useResearchStore.getState().isStreaming).toBe(false);
  });

  it("reset clears everything", () => {
    useResearchStore.getState().addUserMessage("q");
    reset();
    expect(useResearchStore.getState().messages).toEqual([]);
    expect(useResearchStore.getState().isStreaming).toBe(false);
  });
});
