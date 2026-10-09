import { logsHref, monthRange, presetRange } from "./periods";

const today = new Date(Date.UTC(2026, 9, 9, 15)); // 2026-10-09

test("presets are inclusive UTC ranges", () => {
  expect(presetRange("30d", today)).toEqual({
    from: "2026-09-10",
    to: "2026-10-09",
  });
  expect(presetRange("12m", today)).toEqual({
    from: "2025-10-10",
    to: "2026-10-09",
  });
  expect(presetRange("year", today)).toEqual({
    from: "2026-01-01",
    to: "2026-12-31",
  });
  expect(presetRange("quarter", today)).toEqual({
    from: "2026-07-01",
    to: "2026-09-30",
  });
  expect(presetRange("quarter", new Date(Date.UTC(2026, 1, 3)))).toEqual({
    from: "2025-10-01",
    to: "2025-12-31",
  });
});

test("a month bar narrows to the month, clipped to the period", () => {
  expect(monthRange("2026-03", "2026-01-01", "2026-12-31")).toEqual({
    from: "2026-03-01",
    to: "2026-03-31",
  });
  expect(monthRange("2026-03", "2026-03-15", "2026-12-31")).toEqual({
    from: "2026-03-15",
    to: "2026-03-31",
  });
  expect(monthRange("2024-02", "2024-01-01", "2024-02-10")).toEqual({
    from: "2024-02-01",
    to: "2024-02-10",
  });
});

test("links hide examples so counts match", () => {
  expect(
    logsHref("d1", [
      ["tag", "project"],
      ["from", "2026-01-01"],
    ]),
  ).toBe("/documents/d1?tag=project&from=2026-01-01&examples=false");
});
