import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Me } from "../lib/types";

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: () => api<Me>("/me"),
    staleTime: 60_000,
  });
}
