import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { AuditEntry, GrantRole, Sharing } from "../lib/types";

const key = (docId: string) => ["sharing", docId];

export function useSharing(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: key(docId),
    queryFn: () => api<Sharing>(`/documents/${docId}/sharing`),
    enabled,
  });
}

export function useDocumentAudit(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["audit", docId],
    queryFn: () => api<AuditEntry[]>(`/documents/${docId}/audit`),
    enabled,
  });
}

export function useTenantAudit(enabled: boolean) {
  return useQuery({
    queryKey: ["audit", "tenant"],
    queryFn: () => api<AuditEntry[]>("/tenant/audit"),
    enabled,
  });
}

function useSharingMutation<TVars, TOut = void>(
  docId: string,
  fn: (v: TVars) => Promise<TOut>,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: key(docId) });
      void qc.invalidateQueries({ queryKey: ["audit"] });
      void qc.invalidateQueries({ queryKey: ["documents"] }); // role changes after a transfer
    },
  });
}

export const useShare = (docId: string) =>
  useSharingMutation(docId, (body: { email: string; role: GrantRole }) =>
    api<{ kind: "grant" | "invitation" }>(`/documents/${docId}/sharing`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  );

export const useChangeRole = (docId: string) =>
  useSharingMutation(
    docId,
    ({ userId, role }: { userId: string; role: GrantRole }) =>
      api<void>(`/documents/${docId}/grants/${userId}`, {
        method: "PATCH",
        body: JSON.stringify({ role }),
      }),
  );

export const useRevoke = (docId: string) =>
  useSharingMutation(docId, (userId: string) =>
    api<void>(`/documents/${docId}/grants/${userId}`, { method: "DELETE" }),
  );

export const useCancelInvitation = (docId: string) =>
  useSharingMutation(docId, (invId: string) =>
    api<void>(`/documents/${docId}/invitations/${invId}`, { method: "DELETE" }),
  );

export const useTransfer = (docId: string) =>
  useSharingMutation(docId, (userId: string) =>
    api<void>(`/documents/${docId}/transfer`, {
      method: "POST",
      body: JSON.stringify({ user_id: userId }),
    }),
  );
