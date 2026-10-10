import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { pageList } from "#lib/page-list";
import { Pagination } from "#components/pagination";

test("pageList windows around the current page", () => {
  expect(pageList(1, 3, 1)).toEqual([1, 2, 3]);
  expect(pageList(6, 12, 1)).toEqual([1, "gap-l", 5, 6, 7, "gap-r", 12]);
  expect(pageList(1, 12, 1)).toEqual([1, 2, 3, 4, "gap-r", 12]);
  expect(pageList(12, 12, 1)).toEqual([1, "gap-l", 9, 10, 11, 12]);
});

test("Pagination moves pages and marks the current one", async () => {
  const onChange = vi.fn();
  render(
    <Pagination
      pageCount={5}
      defaultPage={2}
      onChange={onChange}
      total={48}
      pageSize={10}
    />,
  );
  expect(screen.getByRole("button", { name: "Page 2" })).toHaveAttribute(
    "aria-current",
    "page",
  );
  await userEvent.click(screen.getByRole("button", { name: "Next page" }));
  expect(onChange).toHaveBeenCalledWith(3);
  expect(screen.getByText("21–30 of 48")).toBeInTheDocument();
});
