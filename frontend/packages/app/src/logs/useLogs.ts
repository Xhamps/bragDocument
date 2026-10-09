import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Impact, Log, LogLink, LogList, LogStatus } from "../lib/types";

/** `search` is the page's URL query string: same names as the API, passed through verbatim. */
export function useLogs(docId: string, search: string) {
  return useQuery({
    queryKey: ["logs", docId, search],
    queryFn: () =>
      api<LogList>(`/documents/${docId}/logs${search ? `?${search}` : ""}`),
    placeholderData: keepPreviousData,
  });
}

export function useLog(docId: string, logId: string | null) {
  return useQuery({
    queryKey: ["logs", docId, "one", logId],
    queryFn: () => api<Log>(`/documents/${docId}/logs/${logId}`),
    enabled: !!logId,
    retry: false, // a missing log won't appear; don't hold ?edit= in the URL
  });
}

export function useTags() {
  return useQuery({
    queryKey: ["tags"],
    queryFn: () => api<{ tags: string[] }>("/tags"),
    select: (d) => d.tags,
  });
}

export type LogForm = {
  name: string;
  description: string;
  impact: Impact;
  status: LogStatus;
  tags: string[];
  links: LogLink[];
  /** Only sent when the user changed the date (back-dating). */
  created_at?: string;
};

function useLogMutation<TVars, TOut>(
  docId: string,
  fn: (v: TVars) => Promise<TOut>,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["logs", docId] });
      void qc.invalidateQueries({ queryKey: ["documents"] }); // card counters
      void qc.invalidateQueries({ queryKey: ["tags"] });
      void qc.invalidateQueries({ queryKey: ["dashboard", docId] });
      void qc.invalidateQueries({ queryKey: ["audit"] });
    },
  });
}

export function useCreateLog(docId: string) {
  return useLogMutation(docId, (body: LogForm) =>
    api<Log>(`/documents/${docId}/logs`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  );
}

export function useUpdateLog(docId: string) {
  return useLogMutation(docId, ({ id, ...body }: { id: string } & LogForm) =>
    api<Log>(`/documents/${docId}/logs/${id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  );
}

export function useDeleteLog(docId: string) {
  return useLogMutation(docId, (id: string) =>
    api<void>(`/documents/${docId}/logs/${id}`, { method: "DELETE" }),
  );
}

export function useDeleteExamples(docId: string) {
  return useLogMutation(docId, () =>
    api<void>(`/documents/${docId}/example-logs`, { method: "DELETE" }),
  );
}
