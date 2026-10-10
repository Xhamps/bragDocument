import { act, renderHook } from "@testing-library/react";
import { useTheme } from "#lib/theme";

beforeEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

test("leaves data-theme unset when nothing is stored", () => {
  renderHook(() => useTheme());
  expect(document.documentElement.dataset.theme).toBeUndefined();
});

test("toggle persists the choice and sets data-theme", () => {
  const { result } = renderHook(() => useTheme());
  act(() => result.current.toggle());
  expect(localStorage.getItem("theme")).toBe("dark");
  expect(document.documentElement.dataset.theme).toBe("dark");
  act(() => result.current.toggle());
  expect(document.documentElement.dataset.theme).toBe("light");
});

test("restores a stored theme", () => {
  localStorage.setItem("theme", "dark");
  const { result } = renderHook(() => useTheme());
  expect(result.current.theme).toBe("dark");
});
