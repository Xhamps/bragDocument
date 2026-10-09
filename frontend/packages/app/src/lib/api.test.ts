// mocks first: it registers the supabase mock before ./api loads the client
import { mockFetch, supabaseMock } from "../test/mocks";
import { api, ApiError, download } from "./api";

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

test("download fetches with the bearer token and clicks a link", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response("pdf")),
  );
  // jsdom has no object URLs
  const { createObjectURL, revokeObjectURL } = URL;
  URL.createObjectURL = vi.fn(() => "blob:x");
  URL.revokeObjectURL = vi.fn();
  const click = vi
    .spyOn(HTMLAnchorElement.prototype, "click")
    .mockImplementation(function (this: HTMLAnchorElement) {
      expect(this.isConnected).toBe(true);
    });
  try {
    await download("/documents/d1/exports/j1/file", "r.pdf");
    const [url, init] = vi.mocked(fetch).mock.calls[0];
    expect(String(url)).toMatch(/\/documents\/d1\/exports\/j1\/file$/);
    expect(new Headers(init?.headers).get("Authorization")).toBe("Bearer tok");
    expect(click).toHaveBeenCalled();
    expect(document.querySelector("a[download]")).toBeNull();
    await vi.waitFor(() =>
      expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:x"),
    );
  } finally {
    URL.createObjectURL = createObjectURL;
    URL.revokeObjectURL = revokeObjectURL;
    click.mockRestore();
  }
});
