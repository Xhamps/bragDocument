import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Checkbox } from "#components/checkbox";
import { RadioGroup } from "#components/radio-group";
import { Toggle } from "#components/toggle";

test("Checkbox toggles via its label", async () => {
  const onCheckedChange = vi.fn();
  render(
    <Checkbox
      label="Remember me"
      description="30 days"
      onCheckedChange={onCheckedChange}
    />,
  );
  await userEvent.click(screen.getByText("Remember me"));
  expect(screen.getByRole("checkbox", { name: "Remember me" })).toBeChecked();
  expect(onCheckedChange).toHaveBeenCalledWith(true);
});

test("RadioGroup reports the chosen value", async () => {
  const onChange = vi.fn();
  render(
    <RadioGroup
      label="Trip"
      options={["Roundtrip", { value: "one", label: "One way" }]}
      defaultValue="Roundtrip"
      onChange={onChange}
    />,
  );
  await userEvent.click(screen.getByRole("radio", { name: "One way" }));
  expect(onChange).toHaveBeenCalledWith("one");
  expect(screen.getByRole("radiogroup", { name: "Trip" })).toBeInTheDocument();
});

test("RadioGroup preselects the first option by default", () => {
  render(<RadioGroup label="Stops" options={["Any", "Nonstop"]} />);
  expect(screen.getByRole("radio", { name: "Any" })).toBeChecked();
});

test("Toggle is a switch that reports changes", async () => {
  const onChange = vi.fn();
  render(<Toggle label="Notifications" onChange={onChange} />);
  await userEvent.click(screen.getByRole("switch", { name: "Notifications" }));
  expect(onChange).toHaveBeenCalledWith(true);
});

test("Checkbox without a label keeps the caller's aria-label", () => {
  render(<Checkbox aria-label="Select row" />);
  expect(
    screen.getByRole("checkbox", { name: "Select row" }),
  ).toBeInTheDocument();
});
