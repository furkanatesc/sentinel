import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { useLiveAlerts } from "./live";

function Probe() {
  useLiveAlerts();
  return <div>probe</div>;
}

test("useLiveAlerts pushes new alerts into cache", async () => {
  const client = getQueryClient();
  client.setQueryData?.(qk.alerts, []);
  render(<QueryClientProvider client={client}><Probe /></QueryClientProvider>);
  await waitFor(() => {
    const alerts = client.getQueryData(qk.alerts) as unknown[] | undefined;
    expect((alerts?.length ?? 0)).toBeGreaterThan(0);
  }, { timeout: 7000 });
}, 8000);
