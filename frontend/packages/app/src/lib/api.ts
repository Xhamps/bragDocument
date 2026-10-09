import { env } from "../env";
import { supabase } from "./supabase";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public fields?: Record<string, string>,
  ) {
    super(message);
  }
}

async function authHeaders(init?: HeadersInit) {
  const { data } = await supabase.auth.getSession();
  const headers = new Headers(init);
  const token = data.session?.access_token;
  if (token) headers.set("Authorization", `Bearer ${token}`);
  return headers;
}

/** fetch wrapper: JSON in/out, bearer token from the Supabase session, errors thrown as ApiError. */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = await authHeaders(init.headers);
  headers.set("Content-Type", "application/json");
  const res = await fetch(`${env.apiUrl}${path}`, { ...init, headers });
  if (res.status === 204) return undefined as T;
  const body = await res.json().catch(() => ({}));
  if (!res.ok)
    throw new ApiError(res.status, body.message ?? res.statusText, body.fields);
  return body as T;
}

/** Saves an authenticated binary response: a plain link cannot carry the bearer token. */
export async function download(path: string, filename: string) {
  const res = await fetch(`${env.apiUrl}${path}`, {
    headers: await authHeaders(),
  });
  if (!res.ok) throw new ApiError(res.status, res.statusText);
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
