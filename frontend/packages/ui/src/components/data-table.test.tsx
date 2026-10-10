import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DataTable } from "#components/data-table";

const rows = [
  { id: 1, name: "Beta", n: 2 },
  { id: 2, name: "alpha", n: 10 },
];
const columns = [
  { key: "name", header: "Name", sortable: true, primary: true },
  { key: "n", header: "Count", sortable: true, align: "right" as const },
];
const names = () =>
  screen
    .getAllByRole("row")
    .slice(1)
    .map((r) => within(r).getAllByRole("cell")[0].textContent);

test("sorts asc, desc, then clears", async () => {
  render(<DataTable columns={columns} rows={rows} />);
  const sortBtn = screen.getByRole("button", { name: /Name/ });
  await userEvent.click(sortBtn);
  expect(names()).toEqual(["alpha", "Beta"]);
  expect(screen.getByRole("columnheader", { name: /Name/ })).toHaveAttribute(
    "aria-sort",
    "ascending",
  );
  await userEvent.click(sortBtn);
  expect(names()).toEqual(["Beta", "alpha"]);
  await userEvent.click(sortBtn);
  expect(screen.getByRole("columnheader", { name: /Name/ })).toHaveAttribute(
    "aria-sort",
    "none",
  );
});

test("selection shows the count and bulk actions; clear resets", async () => {
  const bulk = vi.fn();
  render(
    <DataTable
      columns={columns}
      rows={rows}
      selectable
      rowLabel={(r) => r.name}
      bulkActions={(keys) => (
        <button onClick={() => bulk(keys)}>Archive</button>
      )}
    />,
  );
  await userEvent.click(
    screen.getByRole("checkbox", { name: "Select row Beta" }),
  );
  expect(screen.getByText("1 selected")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Archive" }));
  expect(bulk).toHaveBeenCalledWith([1]);
  await userEvent.click(
    screen.getByRole("checkbox", { name: "Select all rows" }),
  );
  expect(screen.getByText("2 selected")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Clear" }));
  expect(screen.queryByText(/selected/)).toBeNull();
});

test("empty state spans the table", () => {
  render(<DataTable columns={columns} rows={[]} empty="Nothing yet" />);
  expect(screen.getByRole("cell", { name: "Nothing yet" })).toHaveAttribute(
    "colspan",
    "2",
  );
});

test("selecting one of two rows makes the header checkbox indeterminate", async () => {
  render(
    <DataTable
      columns={columns}
      rows={rows}
      selectable
      rowLabel={(r) => r.name}
    />,
  );
  const all = screen.getByRole<HTMLInputElement>("checkbox", {
    name: "Select all rows",
  });
  expect(all.indeterminate).toBe(false);
  await userEvent.click(
    screen.getByRole("checkbox", { name: "Select row Beta" }),
  );
  expect(all.indeterminate).toBe(true);
});
