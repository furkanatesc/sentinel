import { render, screen } from "@testing-library/react";
import { TokenHeader } from "./TokenHeader";
import type { TokenDetail } from "@/lib/api/types";
const token = { id: "t1", name: "SolPulse", symbol: "PULSE", mint: "9xQeWv...4Fk2", ageSeconds: 38, price: 0.0042, priceChange24h: 6.2, marketCap: 320000, liquidity: 82400, volume24h: 494400, scores: {} as any, metrics: {} as any, series: {} as any, risks: {} as any } as TokenDetail;

test("header shows identity and price", () => {
  render(<TokenHeader token={token} />);
  expect(screen.getByText("SolPulse")).toBeInTheDocument();
  expect(screen.getByText("PULSE")).toBeInTheDocument();
  expect(screen.getByText("$0.0042")).toBeInTheDocument();
});
