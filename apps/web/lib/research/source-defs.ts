import { Coins, UserSearch, Share2, Hash, ShieldAlert, Layers, Clock, type LucideIcon } from "lucide-react";
import type { ResearchSource, ResearchSourceKind } from "@/lib/api/types";

export interface SourceKindDef {
  label: string;                                   // human label for the kind
  icon: LucideIcon;
  buildHref?: (src: ResearchSource) => string;     // absent → non-clickable chip
}

export const SOURCE_KIND_DEFS: Record<ResearchSourceKind, SourceKindDef> = {
  token:       { label: "Token",     icon: Coins,      buildHref: (s) => `/tokens/${s.ref}` },
  creator:     { label: "Üretici",   icon: UserSearch, buildHref: (s) => `/creators/${s.ref}` },
  wallet:      { label: "Cüzdan",    icon: Share2,     buildHref: () => `/wallet-graph` },
  tx:          { label: "İşlem",     icon: Hash },
  "risk-rule": { label: "Risk Kuralı", icon: ShieldAlert },
  strategy:    { label: "Strateji",  icon: Layers },   // linksiz (spec default); ileride /strategies/[ref]
  timestamp:   { label: "Zaman",     icon: Clock },
};

export function hrefForSource(src: ResearchSource): string | undefined {
  const def = SOURCE_KIND_DEFS[src.kind];
  return def.buildHref && src.ref !== undefined ? def.buildHref(src) : def.buildHref?.(src);
}
