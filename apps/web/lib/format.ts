export type RiskLevel = "critical" | "high" | "medium" | "good" | "strong";
export type AlertSeverity = "info" | "positive" | "warning" | "critical";

export function scoreToLevel(score: number): RiskLevel {
  if (score <= 24) return "critical";
  if (score <= 49) return "high";
  if (score <= 69) return "medium";
  if (score <= 84) return "good";
  return "strong";
}

export const riskMeta: Record<RiskLevel, { label: string; color: string; bg: string; border: string }> = {
  critical: { label: "Critical", color: "#F0476B", bg: "rgba(240,71,107,0.12)", border: "rgba(240,71,107,0.35)" },
  high: { label: "High Risk", color: "#FFB020", bg: "rgba(255,176,32,0.12)", border: "rgba(255,176,32,0.35)" },
  medium: { label: "Medium", color: "#3E9BFF", bg: "rgba(62,155,255,0.12)", border: "rgba(62,155,255,0.35)" },
  good: { label: "Good", color: "#2FD98B", bg: "rgba(47,217,139,0.12)", border: "rgba(47,217,139,0.35)" },
  strong: { label: "Strong", color: "#2FD98B", bg: "rgba(47,217,139,0.16)", border: "rgba(47,217,139,0.45)" },
};

export const severityMeta: Record<AlertSeverity, { color: string; dot: string }> = {
  info: { color: "#3E9BFF", dot: "#3E9BFF" },
  positive: { color: "#2FD98B", dot: "#2FD98B" },
  warning: { color: "#FFB020", dot: "#FFB020" },
  critical: { color: "#F0476B", dot: "#F0476B" },
};

export function formatAge(s: number): string {
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  return `${m}m ${s % 60}s`;
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
