import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, download } from "../lib/api";
import type { ExportJob, ReportSettings } from "../lib/types";

export function useReportSettings(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["report-settings", docId],
    queryFn: () => api<ReportSettings>(`/documents/${docId}/report-settings`),
    enabled,
  });
}

export function useExportHistory(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["exports", docId],
    queryFn: () => api<{ items: ExportJob[] }>(`/documents/${docId}/exports`),
    select: (d) => d.items,
    enabled,
  });
}

const settled = (j?: ExportJob) =>
  j?.status === "done" || j?.status === "failed";

/** Polls one job every second until it is done or failed (FR-5). */
export function useExportJob(docId: string, jobId: string | null) {
  const qc = useQueryClient();
  return useQuery({
    queryKey: ["exports", docId, jobId],
    queryFn: async () => {
      const j = await api<ExportJob>(`/documents/${docId}/exports/${jobId}`);
      if (settled(j))
        void qc.invalidateQueries({
          queryKey: ["exports", docId],
          exact: true,
        });
      return j;
    },
    enabled: !!jobId,
    refetchInterval: (q) =>
      q.state.error || settled(q.state.data) ? false : 1000,
  });
}

export function useCreateExport(docId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ReportSettings & { query: string }) =>
      api<ExportJob>(`/documents/${docId}/exports`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["report-settings", docId] });
      void qc.invalidateQueries({ queryKey: ["exports", docId], exact: true });
    },
  });
}

export function downloadJob(docId: string, j: ExportJob) {
  return download(
    `/documents/${docId}/exports/${j.id}/file`,
    `brag-report-${j.created_at.slice(0, 10)}.pdf`,
  );
}
