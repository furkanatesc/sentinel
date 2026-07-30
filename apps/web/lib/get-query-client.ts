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
  events: ["events"] as const,
  walletGraph: ["wallet-graph"] as const,
  creators: ["creators"] as const,
  creator: (address: string) => ["creator", address] as const,
};
