import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Document, ExportJob } from "../lib/types";

const doc: Document = {
  id: "d1",
  owner_id: "u1",
  title: "2026",
  description: "",
  state: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
  log_count: 4,
  last_log_at: null,
  role: "viewer",
  owner_name: "Bob",
  is_new: false,
};
const job = (over: Partial<ExportJob> = {}): ExportJob => ({
  id: "j1",
  status: "queued",
  progress: 0,
  error: "",
  created_at: "2026-10-09T10:00:00Z",
  expires_at: "2026-10-10T10:00:00Z",
  downloadable: false,
  ...over,
});
const base = {
  "GET /me": me,
  "GET /documents/d1": doc,
  "GET /documents/d1/logs": { items: [], total: 0 },
  "GET /tags": { tags: ["project", "misc"] },
  "GET /documents/d1/report-settings": {
    goals_this_year: "ship",
    goals_next_year: "",
    section_map: { project: "Projects" },
  },
  "GET /documents/d1/exports": { items: [] },
};

test("export carries the list's filters, goals, and mapping", async () => {
  const calls = mockFetch({
    ...base,
    "POST /documents/d1/exports": { status: 202, body: job() },
  });
  renderAt("/documents/d1?impact=high&status=done&page=2");
  await userEvent.click(
    await screen.findByRole("button", { name: /export pdf/i }),
  );
  const dialog = await screen.findByRole("dialog");
  expect(
    await within(dialog).findByLabelText(/goals for this year/i),
  ).toHaveValue("ship");
  expect(within(dialog).getByLabelText("Section for misc")).toHaveValue(
    "Other",
  );
  await userEvent.selectOptions(
    within(dialog).getByLabelText("Section for misc"),
    "Company building",
  );
  await userEvent.click(
    within(dialog).getByRole("button", { name: /generate/i }),
  );
  const post = calls.find((c) => c.method === "POST");
  expect(post?.body).toMatchObject({
    query: "impact=high&status=done",
    goals_this_year: "ship",
    section_map: { project: "Projects", misc: "Company building" },
  });
});

test("too many logs shows the narrowing message", async () => {
  mockFetch({
    ...base,
    "POST /documents/d1/exports": {
      status: 422,
      body: {
        message: "validation failed",
        fields: {
          filters: "2400 logs match; narrow the filters to at most 2000",
        },
      },
    },
  });
  renderAt("/documents/d1");
  await userEvent.click(
    await screen.findByRole("button", { name: /export pdf/i }),
  );
  await userEvent.click(
    await within(await screen.findByRole("dialog")).findByRole("button", {
      name: /generate/i,
    }),
  );
  expect(await screen.findByText(/narrow the filters/i)).toBeInTheDocument();
});

test("polls to done and offers the download outside the dialog", async () => {
  let n = 0;
  mockFetch({
    ...base,
    "POST /documents/d1/exports": { status: 202, body: job() },
    "GET /documents/d1/exports/j1": () =>
      ++n < 2
        ? job({ status: "running", progress: 40 })
        : job({ status: "done", progress: 100, downloadable: true }),
  });
  renderAt("/documents/d1");
  await userEvent.click(
    await screen.findByRole("button", { name: /export pdf/i }),
  );
  await userEvent.click(
    await within(await screen.findByRole("dialog")).findByRole("button", {
      name: /generate/i,
    }),
  );
  await userEvent.keyboard("{Escape}");
  const status = await screen.findByRole("status", {}, { timeout: 3000 });
  expect(
    await within(status).findByRole(
      "button",
      { name: /download/i },
      { timeout: 3000 },
    ),
  ).toBeInTheDocument();
});

test("dashboard export carries the period", async () => {
  const calls = mockFetch({
    ...base,
    "GET /documents/d1/dashboard": {
      from: "2026-01-01",
      to: "2026-03-31",
      total: 0,
      in_period: 0,
      high_impact: 0,
      in_progress: 0,
      months: [],
      tags: [],
      statuses: [],
      impacts: [],
      coverage: [],
    },
    "POST /documents/d1/exports": { status: 202, body: job() },
  });
  renderAt(
    "/documents/d1/dashboard?period=custom&from=2026-01-01&to=2026-03-31",
  );
  await userEvent.click(
    await screen.findByRole("button", { name: /export pdf/i }),
  );
  await userEvent.click(
    await within(await screen.findByRole("dialog")).findByRole("button", {
      name: /generate/i,
    }),
  );
  expect(calls.find((c) => c.method === "POST")?.body).toMatchObject({
    query: "from=2026-01-01&to=2026-03-31",
  });
});
