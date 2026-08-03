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
export function useStrategies() {
  return useQuery({ queryKey: qk.strategies, queryFn: () => getApi().getStrategies() });
}
export function useStrategy(id: string) {
  return useQuery({ queryKey: qk.strategy(id), queryFn: () => getApi().getStrategy(id) });
}
export function usePortfolio() {
  return useQuery({ queryKey: qk.portfolio, queryFn: () => getApi().getPortfolio() });
}
export function usePositions() {
  return useQuery({ queryKey: qk.positions, queryFn: () => getApi().getPositions() });
}
export function useCandles(mint: string) {
  return useQuery({ queryKey: qk.candles(mint), queryFn: () => getApi().getCandles(mint) });
}
export function useMarketData(mint: string) {
  return useQuery({ queryKey: qk.marketData(mint), queryFn: () => getApi().getMarketData(mint) });
}
export function useOrders() {
  return useQuery({ queryKey: qk.orders, queryFn: () => getApi().getOrders() });
}
export function useTransactions() {
  return useQuery({ queryKey: qk.transactions, queryFn: () => getApi().getTransactions() });
}
export function useTradeLogs() {
  return useQuery({ queryKey: qk.tradeLogs, queryFn: () => getApi().getTradeLogs() });
}
