import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertRuleForm from "./AlertRuleForm";

const toastSuccess = vi.fn();
vi.mock("sonner", () => ({
  toast: Object.assign((...a: unknown[]) => toastSuccess(...a), {
    success: (...a: unknown[]) => toastSuccess(...a),
    error: (...a: unknown[]) => toastSuccess(...a),
  }),
}));
// createAlertRule mutation: geçerli submit'te onSuccess'i tetikle (toast + onDone yolu).
vi.mock("@/lib/hooks/mutations", () => ({
  useCreateAlertRule: () => ({
    mutate: (_draft: unknown, opts: { onSuccess?: () => void }) => opts?.onSuccess?.(),
  }),
}));

describe("AlertRuleForm", () => {
  it("boş isim → hata gösterir, submit engellenir", () => {
    render(<AlertRuleForm />);
    fireEvent.click(screen.getByRole("button", { name: /kaydet/i }));
    expect(screen.getByText(/zorunlu/i)).toBeInTheDocument();
    expect(toastSuccess).not.toHaveBeenCalled();
  });
  it("geçerli form → simüle toast", () => {
    render(<AlertRuleForm />);
    fireEvent.change(screen.getByLabelText("İsim"), { target: { value: "Yeni kural" } });
    fireEvent.click(screen.getByRole("button", { name: /kaydet/i }));
    expect(toastSuccess).toHaveBeenCalled();
  });
});
