import { QueryClient } from "@tanstack/react-query";

export function getQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { staleTime: 30_000, refetchOnWindowFocus: false } },
  });
}

export const qk = {
  kpis: ["kpis"] as const,
  tokens: ["tokens"] as const,
  alerts: ["alerts"] as const,
  radar: ["radar"] as const,
  token: (mint: string) => ["token", mint] as const,
};
