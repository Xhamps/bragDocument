import { vi } from "vitest";
import type { ReactNode } from "react";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryRouter, RouterProvider } from "react-router";
import { render } from "@testing-library/react";
import { AuthProvider } from "../auth/AuthProvider";
import { routes } from "../router";
import type { Me } from "../lib/types";

export const me: Me = {
  id: "u1",
  email: "a@acme.com",
  display_name: "Ada",
  role: "admin",
  tenant: { id: "t1", name: "Acme" },
};

// ponytail: hoisted + mocked here so the mock exists before this module's own
// imports (router → Root → AuthProvider → lib/supabase) are evaluated.
const supabaseMock = vi.hoisted(() => ({
  auth: {
    getSession: vi.fn(async () => ({
      data: { session: { access_token: "tok" } },
    })),
    onAuthStateChange: vi.fn<
      (cb: (event: string, session: unknown) => void) => {
        data: { subscription: { unsubscribe(): void } };
      }
    >(() => ({ data: { subscription: { unsubscribe() {} } } })),
    signInWithPassword: vi.fn(async () => ({ error: null })),
    signUp: vi.fn(async () => ({ error: null })),
    signInWithOtp: vi.fn(async () => ({ error: null })),
    signInWithOAuth: vi.fn(async () => ({ error: null })),
    signOut: vi.fn(async () => ({ error: null })),
    resetPasswordForEmail: vi.fn(async () => ({ error: null })),
    updateUser: vi.fn(async () => ({ error: null })),
  },
}));

vi.mock("../lib/supabase", () => ({ supabase: supabaseMock }));
export { supabaseMock };

/**
 * Route table for fetch: key "METHOD /path" → JSON body, `{ status, body }`,
 * or a function of the request init returning any of those or a Response.
 */
export type Routes = Record<string, unknown>;

export function mockFetch(table: Routes) {
  const calls: {
    method: string;
    path: string;
    search: string;
    body?: unknown;
  }[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input));
      const method = init?.method ?? "GET";
      const key = `${method} ${url.pathname}`;
      calls.push({
        method,
        path: url.pathname,
        search: url.search,
        body: init?.body ? JSON.parse(String(init.body)) : undefined,
      });
      if (!(key in table))
        return new Response(JSON.stringify({ message: "no route " + key }), {
          status: 404,
        });
      const v = table[key];
      const r: unknown =
        typeof v === "function"
          ? await (v as (i?: RequestInit) => unknown)(init) // may be async
          : v;
      if (r instanceof Response) return r;
      const { status, body } =
        r !== null &&
        typeof r === "object" &&
        "status" in r &&
        typeof r.status === "number" // a Log's own `status` is a string
          ? (r as { status: number; body?: unknown })
          : { status: method === "POST" ? 201 : 200, body: r };
      if (body === undefined) return new Response(null, { status: 204 });
      return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );
  return calls;
}

export function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] });
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const tree: ReactNode = createElement(
    QueryClientProvider,
    { client: qc },
    createElement(
      AuthProvider,
      null,
      createElement(RouterProvider, { router }),
    ),
  );
  return { ...render(tree), router };
}
