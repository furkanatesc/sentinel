// ScoreCard.test.tsx — Manipulation Risk ters mantık + Neden callback
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ScoreCard } from "./ScoreCard";
import type { ScoreDetail } from "@/lib/api/types";

const mk = (key: any, value: number): ScoreDetail => ({ key, value, confidence: 80, updatedAt: "az önce", breakdown: [{ label: "x", weight: 100, detail: "d" }] });

test("high manipulation score shows Kritik (inverted)", () => {
  render(<ScoreCard def={{ key: "manipulationRisk", label: "Manipülasyon Riski", higherIsBetter: false }} score={mk("manipulationRisk", 90)} selected={false} onExplain={() => {}} />);
  expect(screen.getByText("90")).toBeInTheDocument();
  expect(screen.getByText("Kritik")).toBeInTheDocument();
});

test("Neden bu skor triggers onExplain", async () => {
  const onExplain = vi.fn();
  render(<ScoreCard def={{ key: "tokenSafety", label: "Token Güvenliği", higherIsBetter: true }} score={mk("tokenSafety", 80)} selected={false} onExplain={onExplain} />);
  await userEvent.click(screen.getByText("Neden bu skor?"));
  expect(onExplain).toHaveBeenCalled();
});
