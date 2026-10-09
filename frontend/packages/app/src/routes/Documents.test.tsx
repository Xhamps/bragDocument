import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Document } from "../lib/types";

const doc = (over: Partial<Document>): Document => ({
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
  ...over,
});

test("empty state explains brag documents and creates the first one", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [], shared: [] },
    "POST /documents": (init?: RequestInit) =>
      doc({ title: JSON.parse(String(init?.body)).title }),
  });
  renderAt("/");
  expect(
    await screen.findByText(/what is a brag document/i),
  ).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /jvns\.ca/i })).toHaveAttribute(
    "href",
    "https://jvns.ca/blog/brag-documents/",
  );
  await userEvent.click(
    screen.getByRole("button", { name: /create your first document/i }),
  );
  await userEvent.type(await screen.findByLabelText(/title/i), "2026");
  await userEvent.click(screen.getByRole("button", { name: /^create$/i }));
  expect(calls.find((c) => c.method === "POST")?.body).toEqual({
    title: "2026",
    description: "",
  });
});

test("lists owned documents, hides archived behind a toggle, shared section hidden when empty", async () => {
  mockFetch({
    "GET /me": me,
    "GET /documents": {
      owned: [
        doc({ id: "d1", title: "2026" }),
        doc({ id: "d2", title: "Old", state: "archived" }),
      ],
      shared: [],
    },
  });
  renderAt("/");
  expect(await screen.findByText("2026")).toBeInTheDocument();
  expect(screen.queryByText("Old")).not.toBeInTheDocument();
  expect(
    screen.queryByRole("heading", { name: /shared with you/i }),
  ).not.toBeInTheDocument();
  await userEvent.click(
    screen.getByRole("checkbox", { name: /show archived/i }),
  );
  expect(await screen.findByText("Old")).toBeInTheDocument();
  expect(screen.getByText("Archived")).toBeInTheDocument();
});

test("delete asks for confirmation and then calls the API", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [doc({})], shared: [] },
    "DELETE /documents/d1": undefined,
  });
  renderAt("/");
  const card = (await screen.findByText("2026")).closest("[data-slot=card]")!;
  await userEvent.click(
    within(card as HTMLElement).getByRole("button", { name: /actions/i }),
  );
  await userEvent.click(
    await screen.findByRole("menuitem", { name: /delete/i }),
  );
  expect(await screen.findByRole("dialog")).toHaveTextContent(/delete "2026"/i);
  expect(calls.some((c) => c.method === "DELETE")).toBe(false);
  await userEvent.click(screen.getByRole("button", { name: /^delete$/i }));
  await vi.waitFor(() =>
    expect(
      calls.some((c) => c.method === "DELETE" && c.path === "/documents/d1"),
    ).toBe(true),
  );
});

test("delete error keeps the dialog open and shows the message", async () => {
  mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [doc({})], shared: [] },
    "DELETE /documents/d1": { status: 403, body: { message: "not allowed" } },
  });
  renderAt("/");
  const card = (await screen.findByText("2026")).closest("[data-slot=card]")!;
  await userEvent.click(
    within(card as HTMLElement).getByRole("button", { name: /actions/i }),
  );
  await userEvent.click(
    await screen.findByRole("menuitem", { name: /delete/i }),
  );
  await userEvent.click(
    await screen.findByRole("button", { name: /^delete$/i }),
  );
  expect(await screen.findByRole("alert")).toHaveTextContent("not allowed");
  expect(screen.getByRole("dialog")).toBeInTheDocument();
});

test("rename prefills the dialog and PATCHes the new title", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [doc({})], shared: [] },
    "PATCH /documents/d1": doc({ title: "2027" }),
  });
  renderAt("/");
  const card = (await screen.findByText("2026")).closest("[data-slot=card]")!;
  await userEvent.click(
    within(card as HTMLElement).getByRole("button", { name: /actions/i }),
  );
  await userEvent.click(
    await screen.findByRole("menuitem", { name: /rename/i }),
  );
  const title = await screen.findByLabelText(/title/i);
  expect(title).toHaveValue("2026");
  await userEvent.clear(title);
  await userEvent.type(title, "2027");
  await userEvent.click(screen.getByRole("button", { name: /^save$/i }));
  await vi.waitFor(() =>
    expect(calls.find((c) => c.method === "PATCH")).toMatchObject({
      path: "/documents/d1",
      body: { title: "2027", description: "" },
    }),
  );
});

test("shared documents show owner, role, and New, without the actions menu", async () => {
  mockFetch({
    "GET /me": me,
    "GET /documents": {
      owned: [doc({ id: "d1", title: "Mine" })],
      shared: [
        doc({
          id: "d2",
          title: "Bob 2026",
          owner_id: "u2",
          role: "viewer",
          owner_name: "Bob",
          is_new: true,
        }),
      ],
    },
  });
  renderAt("/");
  const heading = await screen.findByRole("heading", {
    name: /shared with you/i,
  });
  const s = within(heading.closest("section")!);
  expect(s.getByText("Bob 2026")).toBeInTheDocument();
  expect(s.getByText("Shared by Bob")).toBeInTheDocument();
  expect(s.getByText("viewer")).toBeInTheDocument();
  expect(s.getByText("New")).toBeInTheDocument();
  expect(s.queryByRole("button", { name: /actions/i })).not.toBeInTheDocument();
});
