import { afterEach, it, expect } from "vitest";
import { getApi } from "./index";
import { mockApi } from "./mock";
import { httpApi } from "./http";

const OLD = process.env.NEXT_PUBLIC_DATA_SOURCE;
afterEach(() => { process.env.NEXT_PUBLIC_DATA_SOURCE = OLD; });

it("mock modunda tüm endpoint'ler mockApi'den", () => {
  process.env.NEXT_PUBLIC_DATA_SOURCE = "mock";
  expect(getApi().getStrategies).toBe(mockApi.getStrategies);
  expect(getApi().getTokens).toBe(mockApi.getTokens);
});

it("http modunda canlı endpoint httpApi'den, kalan mockApi'den", () => {
  process.env.NEXT_PUBLIC_DATA_SOURCE = "http";
  expect(getApi().getStrategies).toBe(httpApi.getStrategies);
  expect(getApi().getTokens).toBe(mockApi.getTokens);
});
