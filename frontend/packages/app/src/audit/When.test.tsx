import { render, screen } from "@testing-library/react";
import { afterEach, vi } from "vitest";
import { When } from "./When";

afterEach(() => vi.useRealTimers());

const NOW = new Date("2026-10-09T12:00:00Z").getTime();
const ago = (s: number) => new Date(NOW - s * 1000).toISOString();
const fmt = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

test.each([
  [3540, fmt.format(-59, "minute")],
  [3570, fmt.format(-1, "hour")],
  [84500, fmt.format(-23, "hour")],
  [84600, fmt.format(-1, "day")],
])("%i s ago reads %s", (s, want) => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW);
  render(<When at={ago(s)} />);
  expect(screen.getByText(want)).toBeInTheDocument();
});
