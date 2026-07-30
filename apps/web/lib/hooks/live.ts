"use client";
import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getApi } from "@/lib/api";
import { qk } from "@/lib/get-query-client";
import type { TokenRow, AlertEvent, FeedEvent } from "@/lib/api/types";

export function useLiveTokens() {
  const qc = useQueryClient();
  useEffect(() => getApi().subscribeTokens((tokens: TokenRow[]) => {
    qc.setQueryData(qk.tokens, tokens);
  }), [qc]);
}

export function useLiveAlerts() {
  const qc = useQueryClient();
  useEffect(() => getApi().subscribeAlerts((alert: AlertEvent) => {
    qc.setQueryData<AlertEvent[]>(qk.alerts, (prev) => [alert, ...(prev ?? [])].slice(0, 20));
  }), [qc]);
}

export function useLiveEvents() {
  const qc = useQueryClient();
  useEffect(() => getApi().subscribeEvents((e: FeedEvent) => {
    qc.setQueryData<FeedEvent[]>(qk.events, (prev) => [e, ...(prev ?? [])].slice(0, 200));
  }), [qc]);
}
