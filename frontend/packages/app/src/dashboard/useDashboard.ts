import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Dashboard } from "../lib/types";
import type { Range } from "./periods";

export function useDashboard(docId: string, { from, to }: Range) {
  return useQuery({
    queryKey: ["dashboard", docId, from, to],
    queryFn: () =>
      api<Dashboard>(
        `/documents/${docId}/dashboard?${new URLSearchParams({ from, to })}`,
      ),
    placeholderData: keepPreviousData,
  });
}
