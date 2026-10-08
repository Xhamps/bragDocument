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
