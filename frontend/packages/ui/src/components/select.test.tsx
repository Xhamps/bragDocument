import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Select } from "#components/select";

test("renders string and object options and reports changes", async () => {
  const onChange = vi.fn();
  render(
    <Select
      aria-label="Role"
      options={["viewer", { value: "editor", label: "Editor" }]}
      defaultValue="viewer"
      onChange={onChange}
    />,
  );
  await userEvent.selectOptions(
    screen.getByRole("combobox", { name: "Role" }),
    "editor",
  );
  expect(onChange).toHaveBeenCalled();
  expect(screen.getByRole("option", { name: "Editor" })).toBeInTheDocument();
});

test("placeholder is a disabled empty option", () => {
  render(
    <Select
      aria-label="X"
      options={["a"]}
      placeholder="Pick one"
      defaultValue=""
    />,
  );
  expect(screen.getByRole("option", { name: "Pick one" })).toBeDisabled();
});
