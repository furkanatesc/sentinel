import type { GraphNodeType, GraphEdgeType } from "@/lib/api/types";

export interface NodeTypeDef { key: GraphNodeType; label: string; color: string; shape: string; }
export const NODE_TYPE_DEFS: NodeTypeDef[] = [
  { key: "creator_wallet", label: "Creator Cüzdanı", color: "#7C5CFF", shape: "ellipse" },
  { key: "funding_wallet", label: "Fon Cüzdanı", color: "#3E9BFF", shape: "ellipse" },
  { key: "authority_wallet", label: "Yetki Cüzdanı", color: "#3E9BFF", shape: "ellipse" },
  { key: "token", label: "Token", color: "#2FD98B", shape: "round-rectangle" },
  { key: "liquidity_pool", label: "Likidite Havuzu", color: "#FFB020", shape: "diamond" },
  { key: "trader_wallet", label: "Trader Cüzdanı", color: "#8A94A6", shape: "ellipse" },
  { key: "smart_wallet", label: "Akıllı Cüzdan", color: "#2FD98B", shape: "star" },
  { key: "suspicious_wallet", label: "Şüpheli Cüzdan", color: "#F0476B", shape: "ellipse" },
  { key: "exchange_wallet", label: "Borsa Cüzdanı", color: "#C4CBD8", shape: "hexagon" },
];

export interface EdgeTypeDef { key: GraphEdgeType; label: string; color: string; }
export const EDGE_TYPE_DEFS: EdgeTypeDef[] = [
  { key: "funded", label: "Fonladı", color: "#3E9BFF" },
  { key: "created", label: "Oluşturdu", color: "#7C5CFF" },
  { key: "bought", label: "Aldı", color: "#2FD98B" },
  { key: "sold", label: "Sattı", color: "#F0476B" },
  { key: "transferred", label: "Transfer", color: "#8A94A6" },
  { key: "provided_liquidity", label: "Likidite Sağladı", color: "#FFB020" },
  { key: "removed_liquidity", label: "Likidite Çekti", color: "#F0476B" },
  { key: "shares_funder", label: "Ortak Fonlayıcı", color: "#9B6BFF" },
  { key: "controls_authority", label: "Yetki Kontrolü", color: "#3E9BFF" },
];
