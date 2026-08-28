"use client";
import { useEffect, useRef } from "react";
import type { ChatMessage } from "@/lib/store/research";
import type { ResearchSuggestion } from "@/lib/api/types";
import { ChatMessageBubble } from "./ChatMessageBubble";
import { SuggestionChips } from "./SuggestionChips";

export function ChatThread({
  messages, suggestions, onPick, isStreaming,
}: {
  messages: ChatMessage[];
  suggestions: ResearchSuggestion[];
  onPick: (text: string) => void;
  isStreaming: boolean;
}) {
  const endRef = useRef<HTMLDivElement>(null);
  useEffect(() => { endRef.current?.scrollIntoView({ behavior: "smooth" }); }, [messages]);

  return (
    <div className="flex-1 space-y-3 overflow-y-auto pr-1">
      {messages.length === 0 ? (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Token, üretici ya da cüzdanlar hakkında kaynaklı bir analiz sorabilirsin. Örnek sorular:
          </p>
          <SuggestionChips suggestions={suggestions} onPick={onPick} disabled={isStreaming} />
        </div>
      ) : (
        messages.map((m) => <ChatMessageBubble key={m.id} message={m} />)
      )}
      <div ref={endRef} />
    </div>
  );
}
