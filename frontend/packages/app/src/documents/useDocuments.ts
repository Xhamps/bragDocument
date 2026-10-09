import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, api } from "../lib/api";
import type { Document, DocumentList } from "../lib/types";

const key = ["documents"];

export function useDocuments() {
  return useQuery({
    queryKey: key,
    queryFn: () => api<DocumentList>("/documents"),
  });
}

export function useDocument(id: string) {
  return useQuery({
    queryKey: [...key, id],
    queryFn: () => api<Document>(`/documents/${id}`),
    // 4xx won't fix itself: a missing or revoked document should say so at once.
    retry: (n, e) => !(e instanceof ApiError && e.status < 500) && n < 3,
  });
}

function useInvalidating<TVars>(fn: (v: TVars) => Promise<unknown>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: key });
      void qc.invalidateQueries({ queryKey: ["audit"] });
    },
  });
}

export type DocumentForm = { title: string; description: string };

export function useCreateDocument() {
  return useInvalidating((body: DocumentForm) =>
    api<Document>("/documents", { method: "POST", body: JSON.stringify(body) }),
  );
}

export function useUpdateDocument() {
  return useInvalidating(
    ({
      id,
      ...body
    }: { id: string } & Partial<DocumentForm & { state: Document["state"] }>) =>
      api<Document>(`/documents/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
  );
}

export function useDeleteDocument() {
  return useInvalidating((id: string) =>
    api<void>(`/documents/${id}`, { method: "DELETE" }),
  );
}
