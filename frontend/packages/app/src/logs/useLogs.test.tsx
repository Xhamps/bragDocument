import { createElement, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { mockFetch } from "../test/mocks";
import { useLog } from "./useLogs";

test("useLog does not retry a missing log", async () => {
  const calls = mockFetch({
    "GET /documents/d1/logs/gone": { status: 404, body: { message: "nope" } },
  });
  const qc = new QueryClient(); // app defaults: 3 retries with backoff
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
  const { result } = renderHook(() => useLog("d1", "gone"), { wrapper });
  await waitFor(() => expect(result.current.isError).toBe(true));
  expect(calls).toHaveLength(1);
});
