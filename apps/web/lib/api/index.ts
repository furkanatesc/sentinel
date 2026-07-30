import type { SentinelApi } from "./contract";
import { mockApi } from "./mock";
import { httpApi } from "./http";

export function getApi(): SentinelApi {
  return process.env.NEXT_PUBLIC_DATA_SOURCE === "http" ? httpApi : mockApi;
}

export type { SentinelApi };
