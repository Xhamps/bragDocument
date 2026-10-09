import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { TelegramCode, TelegramStatus } from "../lib/types";

export function useTelegramStatus() {
  return useQuery({
    queryKey: ["telegram"],
    queryFn: () => api<TelegramStatus>("/me/telegram"),
    refetchOnWindowFocus: true, // flips to linked when the user returns from Telegram
  });
}

export function useTelegramCode() {
  return useMutation({
    mutationFn: () =>
      api<TelegramCode>("/me/telegram/code", { method: "POST" }),
  });
}

export function useTelegramUnlink() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api<void>("/me/telegram", { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["telegram"] }),
  });
}
