import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { CreatorHeader } from "./CreatorHeader";
import type { CreatorProfile } from "@/lib/api/types";
vi.mock("sonner", () => ({ toast: vi.fn() }));
const p = { address: "CreAxz", walletAgeDays: 19, firstSeen: "19g önce", riskLevel: "high", reputation: {} as any, metrics: {} as any, history: [], behavior: {} as any } as CreatorProfile;
test("shows wallet age and Watch triggers toast", async () => {
  render(<CreatorHeader profile={p} />);
  expect(screen.getByText(/Cüzdan yaşı: 19 gün/)).toBeInTheDocument();
  await userEvent.click(screen.getByText("İzle"));
  expect(toast).toHaveBeenCalled();
});
