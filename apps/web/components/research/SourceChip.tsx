import Link from "next/link";
import type { ResearchSource } from "@/lib/api/types";
import { SOURCE_KIND_DEFS, hrefForSource } from "@/lib/research/source-defs";
import { cn } from "@/lib/utils";

export function SourceChip({ source }: { source: ResearchSource }) {
  const def = SOURCE_KIND_DEFS[source.kind];
  const Icon = def.icon;
  const href = hrefForSource(source);
  const cls = cn(
    "inline-flex items-center gap-1 rounded-full border border-border/60 bg-card px-2 py-0.5 text-[11px] text-muted-foreground",
    href && "hover:border-primary/60 hover:text-foreground transition-colors",
  );
  const inner = (
    <>
      <Icon className="h-3 w-3" />
      <span className="opacity-70">{def.label}:</span>
      <span className="font-medium text-foreground/90">{source.label}</span>
    </>
  );
  return href ? <Link href={href} className={cls}>{inner}</Link> : <span className={cls}>{inner}</span>;
}
