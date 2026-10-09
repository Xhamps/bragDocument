import {
  type QueryClient,
  useInfiniteQuery,
  useQuery,
} from "@tanstack/react-query";
import { api } from "../lib/api";
import type { AuditFilters, AuditPage } from "../lib/types";

export type AuditQuery = Partial<
  Record<"actor" | "document" | "action" | "from" | "to", string>
>;

function pages(path: string, params: Record<string, string>, limit: number) {
  return {
    queryKey: ["audit", path, params, limit],
    initialPageParam: 0,
    queryFn: ({ pageParam }: { pageParam: number }) => {
      const q = new URLSearchParams({ ...params, limit: String(limit) });
      if (pageParam) q.set("before", String(pageParam));
      return api<AuditPage>(`${path}?${q}`);
    },
    getNextPageParam: (last: AuditPage) => last.next_before ?? undefined,
  };
}

/** The Audit log page: filters straight from the URL. */
export function useAudit(filters: AuditQuery) {
  const params = Object.fromEntries(
    Object.entries(filters).filter(([, v]) => v),
  ) as Record<string, string>;
  return useInfiniteQuery(pages("/audit", params, 50));
}

/** The document page's Activity section: the latest few. */
export function useDocumentActivity(docId: string) {
  return useInfiniteQuery(pages(`/documents/${docId}/audit`, {}, 10));
}

export function useAuditFilters() {
  return useQuery({
    queryKey: ["audit", "filters"],
    queryFn: () => api<AuditFilters>("/audit/filters"),
  });
}

/** Entries arrive through the outbox about a second after the action (ADR-0015):
 *  refetch now, and once more when the entry has landed. */
export function refreshAuditSoon(qc: QueryClient) {
  void qc.invalidateQueries({ queryKey: ["audit"] });
  setTimeout(() => void qc.invalidateQueries({ queryKey: ["audit"] }), 2000);
}
