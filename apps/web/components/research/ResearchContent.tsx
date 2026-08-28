"use client";
import { useResearchSuggestions } from "@/lib/hooks/queries";
import { useResearch } from "./use-research";
import { ChatThread } from "./ChatThread";
import { ChatComposer } from "./ChatComposer";
import { InfoDisclaimerBanner } from "./InfoDisclaimerBanner";

export function ResearchContent() {
  const { messages, isStreaming, send, stop } = useResearch();
  const { data: suggestions = [] } = useResearchSuggestions();

  return (
    <div className="mx-auto flex h-[calc(100vh-8rem)] max-w-3xl flex-col gap-3">
      <div>
        <h1 className="text-lg font-semibold">Araştırma Asistanı</h1>
        <p className="text-sm text-muted-foreground">AI destekli, kaynaklı token & cüzdan analizi.</p>
      </div>
      <InfoDisclaimerBanner />
      <ChatThread messages={messages} suggestions={suggestions} onPick={send} isStreaming={isStreaming} />
      <ChatComposer onSend={send} onStop={stop} isStreaming={isStreaming} />
    </div>
  );
}
