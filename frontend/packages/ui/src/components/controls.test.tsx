import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SegmentedControl } from "#components/segmented-control";
import { SearchField } from "#components/search-field";
import { Slider } from "#components/slider";
import { Stepper } from "#components/stepper";
import { FieldTile } from "#components/field-tile";

test("SegmentedControl selects an option", async () => {
  const onChange = vi.fn();
  render(
    <SegmentedControl
      label="Period"
      options={["Day", "Week"]}
      defaultValue="Day"
      onChange={onChange}
    />,
  );
  await userEvent.click(screen.getByRole("radio", { name: "Week" }));
  expect(onChange).toHaveBeenCalledWith("Week");
  expect(screen.getByRole("radio", { name: "Week" })).toBeChecked();
  expect(
    screen.getByRole("radiogroup", { name: "Period" }),
  ).toBeInTheDocument();
});

test("SegmentedControl arrow keys move the selection", async () => {
  render(<SegmentedControl options={["Day", "Week", "Month"]} />);
  await userEvent.click(screen.getByRole("radio", { name: "Day" }));
  await userEvent.keyboard("{ArrowLeft}");
  expect(screen.getByRole("radio", { name: "Month" })).toBeChecked();
});

test("SearchField is a labelled searchbox", () => {
  render(<SearchField label="Search logs" />);
  expect(
    screen.getByRole("searchbox", { name: "Search logs" }),
  ).toBeInTheDocument();
});

test("Stepper clamps to min and max", async () => {
  const onChange = vi.fn();
  render(
    <Stepper
      label="Bags"
      min={0}
      max={1}
      defaultValue={1}
      onChange={onChange}
    />,
  );
  const inc = screen.getByRole("button", { name: /increase/i });
  expect(inc).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: /decrease/i }));
  expect(onChange).toHaveBeenLastCalledWith(0);
  expect(screen.getByRole("button", { name: /decrease/i })).toBeDisabled();
});

test("Slider exposes a formatted value", () => {
  render(<Slider label="Budget" defaultValue={50} format={(v) => `$${v}`} />);
  const slider = screen.getByRole("slider", { name: "Budget" });
  expect(slider).toHaveAttribute("aria-valuetext", "$50");
});

test("Slider Clear link calls onClear", async () => {
  const onClear = vi.fn();
  render(<Slider label="Budget" onClear={onClear} />);
  await userEvent.click(screen.getByRole("button", { name: "Clear" }));
  expect(onClear).toHaveBeenCalled();
});

test("FieldTile shows label and placeholder", () => {
  render(<FieldTile label="From" placeholder="Choose" />);
  expect(
    screen.getByRole("button", { name: /From.*Choose/ }),
  ).toBeInTheDocument();
});

test("Stepper keeps one live region across changes", async () => {
  render(<Stepper label="Bags" defaultValue={1} />);
  const out = screen.getByRole("status");
  await userEvent.click(screen.getByRole("button", { name: /increase/i }));
  expect(screen.getByRole("status")).toBe(out);
  expect(out).toHaveTextContent("2");
});

test("Slider without a visible label takes aria-label", () => {
  render(<Slider aria-label="Volume" defaultValue={500} max={100} />);
  expect(screen.getByRole("slider", { name: "Volume" })).toBeInTheDocument();
});
