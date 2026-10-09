import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Dashboard, Document } from "../lib/types";

const doc: Document = {
  id: "d1",
  owner_id: "u2",
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

const dash: Dashboard = {
  from: "2026-01-01",
  to: "2026-03-31",
  total: 9,
  in_period: 4,
  high_impact: 2,
  in_progress: 1,
  months: [
    { key: "2026-01", count: 1 },
    { key: "2026-02", count: 0 },
    { key: "2026-03", count: 3 },
  ],
  tags: [{ key: "project", count: 3 }],
  statuses: [
    { key: "idea", count: 0 },
    { key: "in_progress", count: 1 },
    { key: "done", count: 3 },
    { key: "dropped", count: 0 },
  ],
  impacts: [
    { key: "low", count: 1 },
    { key: "medium", count: 1 },
    { key: "high", count: 1 },
    { key: "critical", count: 1 },
  ],
  coverage: [
    { key: "project", count: 3 },
    { key: "mentorship", count: 0 },
  ],
};

const url =
  "/documents/d1/dashboard?period=custom&from=2026-01-01&to=2026-03-31";
const routes = {
  "GET /me": me,
  "GET /documents/d1": doc,
  "GET /documents/d1/dashboard": dash,
};
const q = "from=2026-01-01&to=2026-03-31&examples=false";

test("tiles show the numbers and link to the list", async () => {
  const calls = mockFetch(routes);
  renderAt(url);
  const total = await screen.findByRole("link", { name: /total logs\s*9/i });
  expect(total).toHaveAttribute("href", "/documents/d1?examples=false");
  expect(screen.getByRole("link", { name: /in period\s*4/i })).toHaveAttribute(
    "href",
    `/documents/d1?${q}`,
  );
  expect(
    screen.getByRole("link", { name: /high or critical\s*2/i }),
  ).toHaveAttribute("href", `/documents/d1?impact=high&impact=critical&${q}`);
  expect(
    screen.getByRole("link", { name: /in progress\s*1/i }),
  ).toHaveAttribute("href", `/documents/d1?status=in_progress&${q}`);
  expect(calls.find((c) => c.path === "/documents/d1/dashboard")?.search).toBe(
    "?from=2026-01-01&to=2026-03-31",
  );
});

test("table views are real links to each slice", async () => {
  mockFetch(routes);
  renderAt(url);
  const months = await screen.findByRole("region", { name: "Logs per month" });
  await userEvent.click(
    within(months).getByRole("button", { name: /show as table/i }),
  );
  expect(
    within(months).getByRole("link", { name: "Mar 2026" }),
  ).toHaveAttribute(
    "href",
    "/documents/d1?from=2026-03-01&to=2026-03-31&examples=false",
  );

  const tags = screen.getByRole("region", { name: "Top tags" });
  await userEvent.click(
    within(tags).getByRole("button", { name: /show as table/i }),
  );
  expect(within(tags).getByRole("link", { name: "project" })).toHaveAttribute(
    "href",
    `/documents/d1?tag=project&${q}`,
  );
});

test("coverage highlights tags never used", async () => {
  mockFetch(routes);
  renderAt(url);
  const cov = await screen.findByRole("region", { name: /coverage/i });
  expect(
    within(cov).getByRole("link", { name: /mentorship\s*0/i }),
  ).toHaveAttribute("data-zero", "true");
  expect(
    within(cov).getByRole("link", { name: /project\s*3/i }),
  ).not.toHaveAttribute("data-zero");
});

test("period select drives the URL and the request", async () => {
  const calls = mockFetch(routes);
  const { router } = renderAt(url);
  await screen.findByRole("link", { name: /total logs/i });
  await userEvent.selectOptions(screen.getByLabelText(/period/i), "year");
  await vi.waitFor(() =>
    expect(router.state.location.search).toBe("?period=year"),
  );
  const y = new Date().getUTCFullYear();
  await vi.waitFor(() =>
    expect(calls.at(-1)?.search).toBe(`?from=${y}-01-01&to=${y}-12-31`),
  );
  expect(screen.queryByLabelText(/^from$/i)).not.toBeInTheDocument();
});

test("viewers get the dashboard; tabs switch views", async () => {
  mockFetch(routes);
  renderAt(url);
  expect(
    await screen.findByRole("link", { name: "Dashboard" }),
  ).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", { name: "Logs" })).toHaveAttribute(
    "href",
    "/documents/d1",
  );
});
