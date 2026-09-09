"use client";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { getApi } from "@/lib/api";
import { qk } from "@/lib/get-query-client";
import type { AlertRuleDraft } from "@/lib/alerts/alert-defs";

// useCreateAlertRule, yeni alarm kuralı oluşturur + listeyi invalidate eder (refetch).
export function useCreateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (draft: AlertRuleDraft) => getApi().createAlertRule(draft),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.alertRules }),
  });
}

// useSetAlertRuleEnabled, kuralın aç/kapa durumunu kalıcı çevirir + listeyi invalidate eder.
export function useSetAlertRuleEnabled() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      getApi().setAlertRuleEnabled(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.alertRules }),
  });
}
