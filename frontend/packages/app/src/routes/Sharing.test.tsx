import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Document, Sharing } from "../lib/types";

const doc: Document = {
  id: "d1",
  owner_id: "u1",
  title: "2026",
  description: "",
  state: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
  log_count: 0,
  last_log_at: null,
  role: "owner",
  is_new: false,
};

const sharing: Sharing = {
  grants: [
    {
      user_id: "u2",
      email: "bob@acme.com",
      display_name: "Bob",
      role: "viewer",
      granted_at: "2026-01-01T00:00:00Z",
    },
  ],
  invitations: [
    {
      id: "i1",
      email: "new@acme.com",
      role: "editor",
      created_at: "2026-01-02T00:00:00Z",
    },
  ],
};

const base = {
  "GET /me": me,
  "GET /documents": { owned: [], shared: [] },
  "GET /documents/d1": doc,
  "GET /documents/d1/logs": { items: [], total: 0 },
  "GET /tags": { tags: [] },
  "GET /documents/d1/sharing": sharing,
};

async function openPanel() {
  renderAt("/documents/d1");
  await userEvent.click(
    await screen.findByRole("button", { name: /^share$/i }),
  );
  const dialog = within(await screen.findByRole("dialog"));
  await dialog.findByText("Bob");
  return dialog;
}

test("owner invites, changes a role, removes, and cancels", async () => {
  const calls = mockFetch({
    ...base,
    "POST /documents/d1/sharing": { kind: "invitation" },
    "PATCH /documents/d1/grants/u2": undefined,
    "DELETE /documents/d1/grants/u2": undefined,
    "DELETE /documents/d1/invitations/i1": undefined,
  });
  const d = await openPanel();

  await userEvent.type(d.getByLabelText(/^email$/i), "carol@acme.com");
  await userEvent.selectOptions(
    d.getByLabelText(/role for new person/i),
    "editor",
  );
  await userEvent.click(d.getByRole("button", { name: /^share$/i }));
  await vi.waitFor(() =>
    expect(calls.find((c) => c.method === "POST")?.body).toEqual({
      email: "carol@acme.com",
      role: "editor",
    }),
  );
  expect(await d.findByText(/invited carol@acme\.com/i)).toBeInTheDocument();

  await userEvent.selectOptions(
    d.getByLabelText(/role for bob@acme\.com/i),
    "editor",
  );
  await vi.waitFor(() =>
    expect(calls.find((c) => c.method === "PATCH")?.body).toEqual({
      role: "editor",
    }),
  );

  await userEvent.click(
    d.getByRole("button", { name: /remove bob@acme\.com/i }),
  );
  await userEvent.click(
    d.getByRole("button", { name: /cancel invitation for new@acme\.com/i }),
  );
  await vi.waitFor(() => {
    const deleted = calls
      .filter((c) => c.method === "DELETE")
      .map((c) => c.path);
    expect(deleted).toEqual([
      "/documents/d1/grants/u2",
      "/documents/d1/invitations/i1",
    ]);
  });
});

test("transfer needs a confirmation", async () => {
  const calls = mockFetch({
    ...base,
    "POST /documents/d1/transfer": undefined,
  });
  const d = await openPanel();
  await userEvent.click(
    d.getByRole("button", { name: /make bob@acme\.com owner/i }),
  );
  expect(calls.some((c) => c.path === "/documents/d1/transfer")).toBe(false);
  await userEvent.click(
    d.getByRole("button", { name: /^transfer ownership$/i }),
  );
  await vi.waitFor(() =>
    expect(
      calls.find((c) => c.path === "/documents/d1/transfer")?.body,
    ).toEqual({ user_id: "u2" }),
  );
});

test("closing the panel drops a pending transfer confirmation", async () => {
  mockFetch(base);
  const d = await openPanel();
  await userEvent.click(
    d.getByRole("button", { name: /make bob@acme\.com owner/i }),
  );
  expect(
    d.getByRole("button", { name: /^transfer ownership$/i }),
  ).toBeInTheDocument();
  await userEvent.keyboard("{Escape}");
  await vi.waitFor(() =>
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
  );
  await userEvent.click(screen.getByRole("button", { name: /^share$/i }));
  const reopened = within(await screen.findByRole("dialog"));
  await reopened.findByText("Bob");
  expect(
    reopened.queryByRole("button", { name: /^transfer ownership$/i }),
  ).not.toBeInTheDocument();
});
