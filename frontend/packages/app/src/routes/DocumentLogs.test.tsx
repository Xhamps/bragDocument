import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Document, Log } from "../lib/types";

const doc = (over: Partial<Document> = {}): Document => ({
  id: "d1",
  owner_id: "u1",
  title: "2026",
  description: "",
  state: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
  log_count: 1,
  last_log_at: "2026-03-01T12:00:00Z",
  ...over,
});

const log = (over: Partial<Log> = {}): Log => ({
  id: "l1",
  document_id: "d1",
  name: "Moved billing jobs",
  description: "**Result:** zero failures",
  impact: "high",
  impact_statement: "Zero failed runs",
  status: "done",
  is_example: false,
  tags: ["project"],
  links: [{ url: "https://github.com/x/pull/1", label: "PR" }],
  created_at: "2026-03-01T12:00:00Z",
  created_by: "u1",
  updated_at: "2026-03-01T12:00:00Z",
  updated_by: "u1",
  ...over,
});

const routes = (logs: Log[] = [log()], d = doc()) => ({
  "GET /me": me,
  "GET /documents": { owned: [d], shared: [] },
  "GET /documents/d1/logs": { items: logs, total: logs.length },
  "GET /tags": { tags: ["project"] },
});

type Calls = ReturnType<typeof mockFetch>;
const lastListQuery = (calls: Calls) =>
  calls.filter((c) => c.path === "/documents/d1/logs").at(-1)?.search;

test("shows statement, expands to Markdown and safe links", async () => {
  mockFetch(routes());
  renderAt("/documents/d1");
  expect(await screen.findByText("Moved billing jobs")).toBeInTheDocument();
  expect(screen.getByText("Zero failed runs")).toBeInTheDocument();
  expect(screen.getByText("1 log")).toBeInTheDocument();
  await userEvent.click(
    screen.getByRole("button", { name: /moved billing jobs/i }),
  );
  expect(screen.getByText("Result:").tagName).toBe("STRONG");
  const pr = screen.getByRole("link", { name: "PR" });
  expect(pr).toHaveAttribute("rel", "noopener noreferrer");
  expect(pr).toHaveAttribute("target", "_blank");
});

test("warns when a log has no impact", async () => {
  mockFetch(routes([log({ impact_statement: "" })]));
  renderAt("/documents/d1");
  expect(await screen.findByText(/no impact stated/i)).toBeInTheDocument();
});

test("filters live in the URL and the request; chips remove them", async () => {
  const calls = mockFetch(routes());
  renderAt("/documents/d1");
  await screen.findByText("Moved billing jobs");
  await userEvent.click(screen.getByRole("button", { name: "Done" }));
  await userEvent.click(screen.getByRole("button", { name: "critical" }));
  await waitFor(() =>
    expect(lastListQuery(calls)).toBe("?status=done&impact=critical"),
  );
  await userEvent.type(screen.getByRole("searchbox"), "mig");
  await waitFor(() =>
    expect(lastListQuery(calls)).toBe("?status=done&impact=critical&q=mig"),
  );
  await userEvent.click(
    screen.getByRole("button", { name: "Remove filter Status: done" }),
  );
  await waitFor(() =>
    expect(lastListQuery(calls)).toBe("?impact=critical&q=mig"),
  );
  await userEvent.click(screen.getByRole("button", { name: "Clear all" }));
  await waitFor(() => expect(lastListQuery(calls)).toBe(""));
  expect(screen.getByRole("searchbox")).toHaveValue("");
});

test("Cmd+Enter creates; the dialog stays open when no impact was found", async () => {
  const calls = mockFetch({
    ...routes([]),
    "POST /documents/d1/logs": (init?: RequestInit) =>
      log({
        id: "l2",
        name: JSON.parse(String(init?.body)).name,
        impact_statement: "",
      }),
  });
  renderAt("/documents/d1");
  expect(await screen.findByText(/no logs yet/i)).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "New log" }));
  await userEvent.type(screen.getByLabelText("Name"), "Wrote docs");
  await userEvent.type(screen.getByLabelText("Tags"), "Docs{Enter}");
  await userEvent.keyboard("{Meta>}{Enter}{/Meta}");
  expect(
    await screen.findByText(/couldn't find an impact/i),
  ).toBeInTheDocument();
  expect(screen.getByRole("dialog")).toBeInTheDocument();
  expect(calls.find((c) => c.method === "POST")?.body).toEqual({
    name: "Wrote docs",
    description: "",
    impact: "medium",
    status: "done",
    tags: ["docs"],
    links: [],
  });
});

test("no-impact warning only follows a save that re-extracted", async () => {
  const calls = mockFetch({
    ...routes([]),
    "POST /documents/d1/logs": log({ id: "l2", impact_statement: "" }),
    "PATCH /documents/d1/logs/l2": log({
      id: "l2",
      impact_statement: "",
      status: "in_progress",
    }),
  });
  renderAt("/documents/d1");
  await userEvent.click(await screen.findByRole("button", { name: "New log" }));
  await userEvent.type(screen.getByLabelText("Name"), "Moved billing jobs");
  await userEvent.click(screen.getByRole("button", { name: "Save" }));
  expect(
    await screen.findByText(/couldn't find an impact/i),
  ).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Edit log" })).toBeInTheDocument();
  await userEvent.selectOptions(screen.getByLabelText("Status"), "in_progress");
  await userEvent.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() =>
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
  );
  expect(
    calls.some(
      (c) => c.method === "PATCH" && c.path === "/documents/d1/logs/l2",
    ),
  ).toBe(true);
});

test("Cmd+Enter while saving does not submit twice", async () => {
  const calls = mockFetch({
    ...routes([]),
    "POST /documents/d1/logs": () => new Promise(() => {}),
  });
  renderAt("/documents/d1");
  await userEvent.click(await screen.findByRole("button", { name: "New log" }));
  await userEvent.type(screen.getByLabelText("Name"), "Wrote docs");
  await userEvent.keyboard("{Meta>}{Enter}{/Meta}");
  await userEvent.keyboard("{Meta>}{Enter}{/Meta}");
  await waitFor(() =>
    expect(calls.some((c) => c.method === "POST")).toBe(true),
  );
  expect(calls.filter((c) => c.method === "POST")).toHaveLength(1);
});

test("a page past the end offers page 1", async () => {
  mockFetch({
    ...routes(),
    "GET /documents/d1/logs": { items: [], total: 3 },
  });
  renderAt("/documents/d1?page=9");
  expect(
    await screen.findByRole("button", { name: "Go to page 1" }),
  ).toBeInTheDocument();
  expect(screen.queryByText(/no logs yet/i)).not.toBeInTheDocument();
});

test("removes example logs in one click", async () => {
  const calls = mockFetch({
    ...routes([log({ is_example: true })]),
    "DELETE /documents/d1/example-logs": { status: 204 },
  });
  renderAt("/documents/d1");
  expect(await screen.findByText("Example")).toBeInTheDocument();
  await userEvent.click(
    screen.getByRole("button", { name: "Remove examples" }),
  );
  await waitFor(() =>
    expect(
      calls.some(
        (c) => c.method === "DELETE" && c.path === "/documents/d1/example-logs",
      ),
    ).toBe(true),
  );
});

test("archived documents are read-only", async () => {
  mockFetch(routes([log()], doc({ state: "archived" })));
  renderAt("/documents/d1");
  await screen.findByText("Moved billing jobs");
  expect(
    screen.queryByRole("button", { name: "New log" }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "Actions" }),
  ).not.toBeInTheDocument();
});
