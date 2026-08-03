"use client";
import { useTokens } from "@/lib/hooks/queries";
import { cn } from "@/lib/utils";

export function TokenWatchlistPanel({ activeMint, onSelect }: { activeMint: string; onSelect: (mint: string) => void }) {
  const { data: tokens } = useTokens();
  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2 font-medium" style={{ fontSize: 13 }}>Tokenlar</div>
      <div className="flex flex-col overflow-y-auto">
        {(tokens ?? []).map((t) => (
          <button
            key={t.mint}
            onClick={() => onSelect(t.mint)}
            data-active={t.mint === activeMint}
            className={cn(
              "flex items-center justify-between px-3 py-2 text-left hover:bg-accent",
              t.mint === activeMint && "bg-accent"
            )}
            style={{ fontSize: 12 }}
          >
            <span className="font-medium">{t.symbol}</span>
            <span className="text-muted-foreground">{t.price}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
