import { Coins, UserSearch, Share2, Hash, ShieldAlert, Layers, Clock, type LucideIcon } from "lucide-react";
import type { ResearchSource, ResearchSourceKind } from "@/lib/api/types";

export interface SourceKindDef {
  label: string;                                        // human label for the kind
  icon: LucideIcon;
  buildHref?: (src: ResearchSource) => string | undefined; // absent → non-clickable chip; may return undefined if ref missing
}

export const SOURCE_KIND_DEFS: Record<ResearchSourceKind, SourceKindDef> = {
  token:       { label: "Token",     icon: Coins,      buildHref: (s) => (s.ref ? `/tokens/${s.ref}` : undefined) },
  creator:     { label: "Üretici",   icon: UserSearch, buildHref: (s) => (s.ref ? `/creators/${s.ref}` : undefined) },
  wallet:      { label: "Cüzdan",    icon: Share2,     buildHref: () => `/wallet-graph` },
  tx:          { label: "İşlem",     icon: Hash },
  "risk-rule": { label: "Risk Kuralı", icon: ShieldAlert },
  strategy:    { label: "Strateji",  icon: Layers },   // linksiz (spec default); ileride /strategies/[ref]
  timestamp:   { label: "Zaman",     icon: Clock },
};

export function hrefForSource(src: ResearchSource): string | undefined {
  return SOURCE_KIND_DEFS[src.kind].buildHref?.(src);
}
