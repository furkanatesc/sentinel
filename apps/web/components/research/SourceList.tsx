import type { ResearchSource } from "@/lib/api/types";
import { SourceChip } from "./SourceChip";

export function SourceList({ sources }: { sources: ResearchSource[] }) {
  if (!sources.length) return null;
  return (
    <div className="mt-2 flex flex-wrap gap-1.5">
      <span className="text-[11px] text-muted-foreground/70">Kaynaklar:</span>
      {sources.map((s) => <SourceChip key={s.id} source={s} />)}
    </div>
  );
}
