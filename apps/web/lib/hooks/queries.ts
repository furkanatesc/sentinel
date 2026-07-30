"use client";
import { useQuery } from "@tanstack/react-query";
import { getApi } from "@/lib/api";
import { qk } from "@/lib/get-query-client";

export function useKpis() {
  return useQuery({ queryKey: qk.kpis, queryFn: () => getApi().getKpis() });
}
export function useTokens() {
  return useQuery({ queryKey: qk.tokens, queryFn: () => getApi().getTokens() });
}
export function useAlerts() {
  return useQuery({ queryKey: qk.alerts, queryFn: () => getApi().getAlerts() });
}
export function useRadar() {
  return useQuery({ queryKey: qk.radar, queryFn: () => getApi().getRadar() });
}
export function useToken(mint: string) {
  return useQuery({ queryKey: qk.token(mint), queryFn: () => getApi().getToken(mint) });
}
export function useEvents() {
  return useQuery({ queryKey: qk.events, queryFn: () => getApi().getEvents() });
}
export function useWalletGraph() {
  return useQuery({ queryKey: qk.walletGraph, queryFn: () => getApi().getWalletGraph() });
}
export function useCreators() {
  return useQuery({ queryKey: qk.creators, queryFn: () => getApi().getCreators() });
}
export function useCreator(address: string) {
  return useQuery({ queryKey: qk.creator(address), queryFn: () => getApi().getCreator(address) });
}
