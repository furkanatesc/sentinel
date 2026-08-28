"use client";
import { useEffect, useRef } from "react";
import { getApi } from "@/lib/api";
import { useResearchStore } from "@/lib/store/research";

export function useResearch() {
  const messages = useResearchStore((s) => s.messages);
  const isStreaming = useResearchStore((s) => s.isStreaming);
  const cancelRef = useRef<(() => void) | null>(null);
  const activeIdRef = useRef<string | null>(null);

  const send = (question: string) => {
    const q = question.trim();
    if (!q || useResearchStore.getState().isStreaming) return;
    const store = useResearchStore.getState();
    const assistantId = store.addUserMessage(q);
    activeIdRef.current = assistantId;
    cancelRef.current = getApi().streamResearchAnswer(
      q,
      (chunk) => useResearchStore.getState().appendChunk(assistantId, chunk),
      (answer) => {
        useResearchStore.getState().finishAnswer(assistantId, answer.sources);
        cancelRef.current = null;
        activeIdRef.current = null;
      },
    );
  };

  const stop = () => {
    cancelRef.current?.();
    cancelRef.current = null;
    const id = activeIdRef.current;
    if (id) useResearchStore.getState().finishAnswer(id, []);
    activeIdRef.current = null;
  };

  useEffect(() => () => { cancelRef.current?.(); }, []); // cancel on unmount

  return { messages, isStreaming, send, stop };
}
