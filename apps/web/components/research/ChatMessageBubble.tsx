import type { ChatMessage } from "@/lib/store/research";
import { SourceList } from "./SourceList";
import { cn } from "@/lib/utils";

export function ChatMessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user";
  return (
    <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
      <div className={cn(
        "max-w-[80%] rounded-lg px-3 py-2 text-sm",
        isUser ? "bg-primary/15 text-foreground" : "bg-card border border-border/60",
      )}>
        <p className="whitespace-pre-wrap leading-relaxed">
          {message.text}
          {message.status === "streaming" && (
            <span data-testid="stream-cursor" className="ml-0.5 inline-block animate-pulse">▍</span>
          )}
        </p>
        {!isUser && message.status === "done" && message.sources && <SourceList sources={message.sources} />}
      </div>
    </div>
  );
}
