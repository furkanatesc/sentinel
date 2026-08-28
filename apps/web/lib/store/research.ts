import { create } from "zustand";
import type { ResearchSource } from "@/lib/api/types";

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  text: string;
  sources?: ResearchSource[];
  status: "streaming" | "done";
}

interface ResearchState {
  messages: ChatMessage[];
  isStreaming: boolean;
  addUserMessage: (text: string) => string;
  appendChunk: (assistantId: string, chunk: string) => void;
  finishAnswer: (assistantId: string, sources: ResearchSource[]) => void;
  reset: () => void;
}

let seq = 0;
const nextId = () => `m${Date.now()}-${seq++}`;

export const useResearchStore = create<ResearchState>((set) => ({
  messages: [],
  isStreaming: false,
  addUserMessage: (text) => {
    const assistantId = nextId();
    set((s) => ({
      isStreaming: true,
      messages: [
        ...s.messages,
        { id: nextId(), role: "user", text, status: "done" },
        { id: assistantId, role: "assistant", text: "", status: "streaming" },
      ],
    }));
    return assistantId;
  },
  appendChunk: (assistantId, chunk) =>
    set((s) => ({
      messages: s.messages.map((m) => (m.id === assistantId ? { ...m, text: m.text + chunk } : m)),
    })),
  finishAnswer: (assistantId, sources) =>
    set((s) => ({
      isStreaming: false,
      messages: s.messages.map((m) =>
        m.id === assistantId ? { ...m, sources, status: "done" as const } : m,
      ),
    })),
  reset: () => set({ messages: [], isStreaming: false }),
}));
