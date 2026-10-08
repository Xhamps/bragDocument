// mocks first: it registers the supabase mock before ./api loads the client
import { mockFetch, supabaseMock } from "../test/mocks";
import { api, ApiError } from "./api";

test("attaches the bearer token from the session", async () => {
  mockFetch({ "GET /me": { id: "u1" } });
  await api("/me");
  const headers = new Headers(
    (vi.mocked(fetch).mock.calls[0][1] as RequestInit).headers,
  );
  expect(headers.get("Authorization")).toBe("Bearer tok");
  expect(headers.get("Content-Type")).toBe("application/json");
  expect(supabaseMock.auth.getSession).toHaveBeenCalled();
});

test("204 resolves to undefined", async () => {
  mockFetch({ "DELETE /documents/d1": undefined });
  await expect(
    api("/documents/d1", { method: "DELETE" }),
  ).resolves.toBeUndefined();
});

test("error body becomes an ApiError with status, message and fields", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({ message: "invalid", fields: { title: "required" } }),
          {
            status: 422,
          },
        ),
    ),
  );
  const err = await api("/documents", { method: "POST" }).catch((e) => e);
  expect(err).toBeInstanceOf(ApiError);
  expect(err).toMatchObject({
    status: 422,
    message: "invalid",
    fields: { title: "required" },
  });
});
