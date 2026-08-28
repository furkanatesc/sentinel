"use client";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Send, Square } from "lucide-react";

export function ChatComposer({
  onSend, onStop, isStreaming,
}: { onSend: (q: string) => void; onStop: () => void; isStreaming: boolean }) {
  const [value, setValue] = useState("");

  const submit = () => {
    const q = value.trim();
    if (!q || isStreaming) return;
    onSend(q);
    setValue("");
  };

  return (
    <div className="flex items-end gap-2 border-t border-border/60 pt-3">
      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); submit(); } }}
        disabled={isStreaming}
        placeholder="Bir soru sor (ör. GFROG neden riskli?)"
        rows={1}
        className="min-h-10 max-h-40 flex-1 resize-y rounded-md border border-border/60 bg-card px-3 py-2 text-sm outline-none focus:border-primary/60 disabled:opacity-50"
      />
      {isStreaming ? (
        <Button variant="outline" size="sm" onClick={onStop} className="gap-1">
          <Square className="h-3.5 w-3.5" /> Durdur
        </Button>
      ) : (
        <Button size="sm" onClick={submit} disabled={!value.trim()} className="gap-1">
          <Send className="h-3.5 w-3.5" /> Gönder
        </Button>
      )}
    </div>
  );
}
