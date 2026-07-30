// TokenTabs.test.tsx — Overview varsayılan + built ve placeholder sekmeler görünür
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TokenTabs } from "./TokenTabs";

const token = {
  series: {
    price: [{ t: 0, v: 1 }],
    liquidity: [{ t: 0, v: 1 }],
    volume: [{ t: 0, v: 1 }],
    holders: [{ t: 0, v: 1 }],
  },
  metrics: {
    holders: 1, uniqueBuyers: 1, buyRatio: 50, sellRatio: 50,
    creatorHoldingPct: 5, top10HolderPct: 20, sniperPct: 5, botActivityPct: 5,
  },
  risks: { contract: [], market: [], creator: [] },
} as any;

test("shows tabs incl. built and placeholder", () => {
  render(<TokenTabs token={token} />);
  expect(screen.getByText("Genel Bakış")).toBeInTheDocument();
  expect(screen.getByText("Risk Analizi")).toBeInTheDocument();
  expect(screen.getByText("Cüzdan Grafiği")).toBeInTheDocument();
});

test("clicking Risk Analizi switches the panel to RiskAnalysisTab content", async () => {
  render(<TokenTabs token={token} />);
  await userEvent.click(screen.getByText("Risk Analizi"));
  expect(await screen.findByText("Kontrat Riski")).toBeInTheDocument();
});
