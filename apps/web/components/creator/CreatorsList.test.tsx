import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { CreatorsList } from "./CreatorsList";

test("lists creators with profile links", async () => {
  render(<QueryClientProvider client={getQueryClient()}><CreatorsList /></QueryClientProvider>);
  await waitFor(() => expect(screen.getAllByText("Profil →").length).toBeGreaterThan(0));
  const link = screen.getAllByText("Profil →")[0].closest("a")!;
  expect(link.getAttribute("href")).toMatch(/^\/creators\//);
});
