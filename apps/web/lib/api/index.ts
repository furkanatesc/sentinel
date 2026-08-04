import type { SentinelApi } from "./contract";
import { mockApi } from "./mock";
import { httpApi } from "./http";
import { LIVE_ENDPOINTS } from "./live-endpoints";

/**
 * Hibrit adapter: DATA_SOURCE=http iken LIVE_ENDPOINTS'teki metotlar gerçek httpApi'ye,
 * kalanı mockApi'ye bağlanır. Canlı endpoint runtime'da mock'a DÜŞMEZ. DATA_SOURCE!=http
 * iken her şey mock. Bileşenler yalnız getApi() görür (DIP).
 */
export function getApi(): SentinelApi {
  if (process.env.NEXT_PUBLIC_DATA_SOURCE !== "http") return mockApi;
  return new Proxy(mockApi, {
    get(target, prop) {
      if (typeof prop === "string" && LIVE_ENDPOINTS.has(prop as keyof SentinelApi)) {
        return httpApi[prop as keyof SentinelApi];
      }
      return target[prop as keyof SentinelApi];
    },
  }) as SentinelApi;
}

export type { SentinelApi };
