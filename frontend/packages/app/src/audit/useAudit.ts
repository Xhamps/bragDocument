import {
  type QueryClient,
  useInfiniteQuery,
  useQuery,
} from "@tanstack/react-query";
import { useSearchParams } from "react-router";
import { api } from "../lib/api";
import type { AuditFilters, AuditPage } from "../lib/types";

const keys = ["actor", "document", "action", "from", "to"] as const;
export type AuditKey = (typeof keys)[number];
export type AuditQuery = Partial<Record<AuditKey, string>>;

const nonEmpty = (q: AuditQuery) =>
  Object.fromEntries(Object.entries(q).filter(([, v]) => v)) as Record<
    string,
    string
  >;

/** Filters live in the URL so a filtered view can be shared (PRD-0009 §9). */
export function useAuditParams() {
  const [params, setParams] = useSearchParams();
  const filters = Object.fromEntries(
    keys.map((k) => [k, params.get(k) ?? ""]),
  ) as Required<AuditQuery>;
  const set = (k: AuditKey, v: string) =>
    setParams((p) => {
      const next = new URLSearchParams(p);
      if (v) next.set(k, v);
      else next.delete(k);
      return next;
    });
  return { filters, set, clear: () => setParams({}) };
}
export type AuditParams = ReturnType<typeof useAuditParams>;

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
  return useInfiniteQuery(pages("/audit", nonEmpty(filters), 50));
}

/** The document's Activity tab; only fetched for owners and tenant admins. */
export function useDocumentActivity(
  docId: string,
  filters: AuditQuery,
  enabled: boolean,
) {
  return useInfiniteQuery({
    // The path picks the document.
    ...pages(
      `/documents/${docId}/audit`,
      nonEmpty({ ...filters, document: "" }),
      50,
    ),
    enabled,
  });
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
