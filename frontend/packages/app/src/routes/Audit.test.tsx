import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";

const entry = {
  id: 2,
  at: "2026-10-09T12:00:00Z",
  source: "telegram",
  action: "log.created",
  actor: { id: "u2", name: "Bob", email: "bob@acme.com" },
  document: { id: "d1", title: "2026" },
  target: { type: "log", id: "l1", name: "Migrated billing" },
  role: "",
  changed_fields: [],
};
const filters = {
  actors: [{ id: "u2", name: "Bob", email: "bob@acme.com" }],
  documents: [{ id: "d1", title: "2026" }],
};
const docs = { owned: [], shared: [] };
const base = {
  "GET /me": me,
  "GET /documents": docs,
  "GET /audit/filters": filters,
};
const auditCalls = (calls: { path: string; search: string }[]) =>
  calls.filter((c) => c.path === "/audit");

test("lists entries; clicking a user or document filters and keeps it in the URL", async () => {
  const calls = mockFetch({
    ...base,
    "GET /audit": { entries: [entry], next_before: null },
  });
  const { router } = renderAt("/audit");
  expect(
    await screen.findByText(/created log “Migrated billing”/),
  ).toBeInTheDocument();
  expect(screen.getByText("telegram")).toBeInTheDocument();
  expect(screen.getByRole("columnheader", { name: "Source" })).toBeVisible();

  await userEvent.click(screen.getByRole("button", { name: "Bob" }));
  await vi.waitFor(() =>
    expect(auditCalls(calls).at(-1)?.search).toContain("actor=u2"),
  );
  expect(router.state.location.search).toContain("actor=u2");
  expect(screen.getByLabelText("User")).toHaveValue("u2");

  await userEvent.click(screen.getByRole("button", { name: "2026" }));
  await vi.waitFor(() =>
    expect(auditCalls(calls).at(-1)?.search).toContain("document=d1"),
  );
});

test("filters read from the URL and selects update it", async () => {
  const calls = mockFetch({
    ...base,
    "GET /audit": { entries: [entry], next_before: null },
  });
  const { router } = renderAt("/audit?action=log.created");
  await screen.findByText(/created log/);
  expect(auditCalls(calls)[0].search).toContain("action=log.created");
  expect(screen.getByLabelText("Action")).toHaveValue("log.created");

  await userEvent.selectOptions(screen.getByLabelText("Action"), "");
  await vi.waitFor(() =>
    expect(router.state.location.search).not.toContain("action="),
  );
});

test("filtered empty state offers Clear filters", async () => {
  const calls = mockFetch({
    ...base,
    "GET /audit": { entries: [], next_before: null },
  });
  renderAt("/audit?document=d1");
  expect(
    await screen.findByText("No entries match these filters"),
  ).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: /clear filters/i }));
  expect(await screen.findByText("No activity yet.")).toBeInTheDocument();
  expect(auditCalls(calls).at(-1)?.search).not.toContain("document=");
});

test("Load more fetches the next page by cursor", async () => {
  const calls = mockFetch({
    ...base,
    "GET /audit": { entries: [entry], next_before: 2 },
  });
  renderAt("/audit");
  await userEvent.click(
    await screen.findByRole("button", { name: /load more/i }),
  );
  await vi.waitFor(() =>
    expect(auditCalls(calls).at(-1)?.search).toContain("before=2"),
  );
});

test("a failed load shows the error with Retry", async () => {
  const calls = mockFetch({
    ...base,
    "GET /audit": { status: 500, body: { message: "boom" } },
  });
  renderAt("/audit");
  expect(await screen.findByRole("alert")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Retry" }));
  await vi.waitFor(() => expect(auditCalls(calls)).toHaveLength(2));
});

test("nav shows Audit log to admins", async () => {
  mockFetch({ "GET /me": me, "GET /documents": docs });
  renderAt("/");
  expect(
    await screen.findByRole("link", { name: "Audit log" }),
  ).toHaveAttribute("href", "/audit");
});

test("nav shows Audit log to members who own a document", async () => {
  mockFetch({
    "GET /me": { ...me, role: "member" },
    "GET /documents": {
      owned: [{ id: "d1", title: "2026", role: "owner" }],
      shared: [],
    },
  });
  renderAt("/");
  expect(
    await screen.findByRole("link", { name: "Audit log" }),
  ).toBeInTheDocument();
});

test("nav hides Audit log from members who own nothing", async () => {
  mockFetch({ "GET /me": { ...me, role: "member" }, "GET /documents": docs });
  renderAt("/");
  await screen.findByText("a@acme.com");
  expect(
    screen.queryByRole("link", { name: "Audit log" }),
  ).not.toBeInTheDocument();
});
