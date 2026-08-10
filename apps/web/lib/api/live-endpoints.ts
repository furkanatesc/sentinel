import type { SentinelApi } from "./contract";

// Backend'de gerçekleşmiş endpoint'ler. Her backend alt-projesi buraya ekler (OCP).
export const LIVE_ENDPOINTS = new Set<keyof SentinelApi>([
  "getStrategies",
  "getEvents",
  "getTokens",
  "getToken",
  "getCreators",
  "getCreator",
  "subscribeEvents",
  "subscribeTokens",
]);
