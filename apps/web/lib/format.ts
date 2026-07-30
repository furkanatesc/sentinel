export type RiskLevel = "critical" | "high" | "medium" | "good" | "strong";
export type AlertSeverity = "info" | "positive" | "warning" | "critical";
export type RiskSeverity = "critical" | "high" | "medium" | "info";

export function scoreToLevel(score: number): RiskLevel {
  if (score <= 24) return "critical";
  if (score <= 49) return "high";
  if (score <= 69) return "medium";
  if (score <= 84) return "good";
  return "strong";
}

export const riskMeta: Record<RiskLevel, { label: string; color: string; bg: string; border: string }> = {
  critical: { label: "Kritik", color: "#F0476B", bg: "rgba(240,71,107,0.12)", border: "rgba(240,71,107,0.35)" },
  high: { label: "Yüksek Risk", color: "#FFB020", bg: "rgba(255,176,32,0.12)", border: "rgba(255,176,32,0.35)" },
  medium: { label: "Orta", color: "#3E9BFF", bg: "rgba(62,155,255,0.12)", border: "rgba(62,155,255,0.35)" },
  good: { label: "İyi", color: "#2FD98B", bg: "rgba(47,217,139,0.12)", border: "rgba(47,217,139,0.35)" },
  strong: { label: "Güçlü", color: "#2FD98B", bg: "rgba(47,217,139,0.16)", border: "rgba(47,217,139,0.45)" },
};

export const severityMeta: Record<AlertSeverity, { color: string; dot: string }> = {
  info: { color: "#3E9BFF", dot: "#3E9BFF" },
  positive: { color: "#2FD98B", dot: "#2FD98B" },
  warning: { color: "#FFB020", dot: "#FFB020" },
  critical: { color: "#F0476B", dot: "#F0476B" },
};

export const riskSeverityMeta: Record<RiskSeverity, { label: string; color: string; bg: string; border: string }> = {
  critical: { label: "Kritik", color: "#F0476B", bg: "rgba(240,71,107,0.12)", border: "rgba(240,71,107,0.35)" },
  high: { label: "Yüksek", color: "#FFB020", bg: "rgba(255,176,32,0.12)", border: "rgba(255,176,32,0.35)" },
  medium: { label: "Orta", color: "#3E9BFF", bg: "rgba(62,155,255,0.12)", border: "rgba(62,155,255,0.35)" },
  info: { label: "Bilgi", color: "#8A94A6", bg: "rgba(138,148,166,0.12)", border: "rgba(138,148,166,0.30)" },
};

export function formatPct(n: number): string {
  return `%${n.toFixed(1)}`;
}

export function formatAge(s: number): string {
  if (s < 60) return `${s}sn`;
  const m = Math.floor(s / 60);
  return `${m}dk ${s % 60}sn`;
}

export function formatPrice(p: number): string {
  if (p >= 1) return `$${p.toFixed(2)}`;
  if (p >= 0.001) return `$${p.toFixed(4)}`;
  return `$${p.toExponential(1)}`;
}

export function formatUsd(n: number): string {
  if (n >= 1_000_000) return `$${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `$${(n / 1_000).toFixed(1)}K`;
  return `$${n}`;
}
