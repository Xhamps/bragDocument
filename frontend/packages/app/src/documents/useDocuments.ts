import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Document, DocumentList } from "../lib/types";

const key = ["documents"];

export function useDocuments() {
  return useQuery({
    queryKey: key,
    queryFn: () => api<DocumentList>("/documents"),
  });
}

function useInvalidating<TVars>(fn: (v: TVars) => Promise<unknown>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
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
