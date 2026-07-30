// ExplainableScore.test.tsx
import { render, screen } from "@testing-library/react";
import { ExplainableScore } from "./ExplainableScore";

test("renders weighted breakdown items", () => {
  render(<ExplainableScore def={{ key: "creatorReputation", label: "Üretici İtibarı", higherIsBetter: true }}
    score={{ key: "creatorReputation", value: 27, confidence: 70, updatedAt: "az önce", breakdown: [{ label: "Geçmiş performans", weight: 40, detail: "kötü" }] }} />);
  expect(screen.getByText("Geçmiş performans")).toBeInTheDocument();
  expect(screen.getByText("%40")).toBeInTheDocument();
});
