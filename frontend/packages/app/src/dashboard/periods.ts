export type Preset = "30d" | "quarter" | "12m" | "year";
export type Range = { from: string; to: string };

export const PRESETS: { value: Preset | "custom"; label: string }[] = [
  { value: "30d", label: "Last 30 days" },
  { value: "quarter", label: "Last quarter" },
  { value: "12m", label: "Last 12 months" },
  { value: "year", label: "This year" },
  { value: "custom", label: "Custom range" },
];

const iso = (d: Date) => d.toISOString().slice(0, 10);
const utc = (y: number, m: number, d: number) => new Date(Date.UTC(y, m, d));

/** Inclusive YYYY-MM-DD range in UTC; the API's dates are UTC (PRD-0005 FR-2). */
export function presetRange(p: Preset, today = new Date()): Range {
  const y = today.getUTCFullYear();
  const m = today.getUTCMonth();
  const d = today.getUTCDate();
  switch (p) {
    case "30d":
      return { from: iso(utc(y, m, d - 29)), to: iso(utc(y, m, d)) };
    case "12m":
      return { from: iso(utc(y - 1, m, d + 1)), to: iso(utc(y, m, d)) };
    case "year":
      return { from: `${y}-01-01`, to: `${y}-12-31` };
    case "quarter": {
      const q = Math.floor(m / 3) * 3; // first month of this quarter
      return { from: iso(utc(y, q - 3, 1)), to: iso(utc(y, q, 0)) };
    }
  }
}

/** The days of month "YYYY-MM" that fall inside [from, to]. */
export function monthRange(key: string, from: string, to: string): Range {
  const [y, m] = key.split("-").map(Number);
  const start = iso(utc(y, m - 1, 1));
  const end = iso(utc(y, m, 0));
  return { from: start > from ? start : from, to: end < to ? end : to };
}

/** The logs list filtered to a dashboard slice; examples hidden like the dashboard (FR-3, FR-6). */
export function logsHref(docId: string, pairs: [string, string][]) {
  const p = new URLSearchParams(pairs);
  p.set("examples", "false");
  return `/documents/${docId}?${p}`;
}
