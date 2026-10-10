import { Link, useParams, useSearchParams } from "react-router";
import { Select, TextField, focusRing } from "@bragdoc/ui";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { LogStatus } from "../lib/types";
import { DocumentPage } from "../documents/DocumentPage";
import { useDocument } from "../documents/useDocuments";
import { STATUS_LABEL } from "../logs/constants";
import { BarList } from "../dashboard/BarList";
import {
  logsHref,
  monthRange,
  PRESETS,
  presetRange,
  type Preset,
  type Range,
} from "../dashboard/periods";
import { useDashboard } from "../dashboard/useDashboard";
import { ExportButton } from "../exports/ExportButton";

const monthLabel = (key: string) =>
  new Date(`${key}-01T00:00:00Z`).toLocaleDateString("en-US", {
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  });

export function Component() {
  const { id = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const raw = params.get("period");
  const period = PRESETS.some((p) => p.value === raw)
    ? (raw as Preset | "custom")
    : "12m";
  let range: Range;
  if (period === "custom") {
    // A half-open custom range: no `to` means today, no `from` means 12 months before `to`.
    const to = params.get("to") ?? presetRange("12m").to;
    const end = new Date(`${to}T00:00:00Z`);
    const from =
      params.get("from") ??
      presetRange("12m", isNaN(+end) ? undefined : end).from;
    range = { from, to };
  } else range = presetRange(period);
  const doc = useDocument(id);
  const dash = useDashboard(id, range);

  if (doc.isPending) return <p className="text-fg-secondary">Loading…</p>;
  if (!doc.data)
    return (
      <p role="alert" className="text-danger">
        {doc.error instanceof ApiError && doc.error.status === 404
          ? "Document not found."
          : errorText(doc.error)}
      </p>
    );

  const inPeriod: [string, string][] = [
    ["from", range.from],
    ["to", range.to],
  ];
  const href = (pairs: [string, string][]) =>
    logsHref(id, [...pairs, ...inPeriod]);
  const d = dash.data;

  const tile = (label: string, value: number | undefined, to: string) => (
    <Link
      to={to}
      className={`${focusRing} flex flex-col gap-1 rounded-lg glass p-4 shadow-glass hover:border-fg-tertiary`}
    >
      <span className="text-sm text-fg-secondary">{label}</span>
      <span className="text-2xl font-semibold tabular-nums">
        {value ?? "–"}
      </span>
    </Link>
  );

  return (
    <div className="flex flex-col gap-6">
      <DocumentPage
        doc={doc.data}
        current="dashboard"
        actions={
          <>
            <Select
              id="period"
              aria-label="Period"
              prefix="Period:"
              options={PRESETS}
              value={period}
              onChange={(e) => {
                const v = e.target.value;
                setParams(
                  v === "custom"
                    ? { period: v, from: range.from, to: range.to }
                    : { period: v },
                );
              }}
            />
            {period === "custom" && (
              <>
                <TextField
                  aria-label="From"
                  type="date"
                  value={range.from}
                  max={range.to}
                  onChange={(e) =>
                    e.target.value &&
                    setParams({ period, from: e.target.value, to: range.to })
                  }
                />
                <TextField
                  aria-label="To"
                  type="date"
                  value={range.to}
                  min={range.from}
                  onChange={(e) =>
                    e.target.value &&
                    setParams({ period, from: range.from, to: e.target.value })
                  }
                />
              </>
            )}
            {/* The active range, preset or custom: the report covers exactly what's shown. */}
            <ExportButton
              docId={id}
              role={doc.data.role}
              params={new URLSearchParams(range)}
            />
          </>
        }
      >
        {dash.error && (
          <p role="alert" className="text-danger">
            {errorText(dash.error)}
          </p>
        )}

        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          {tile("Total logs", d?.total, logsHref(id, []))}
          {tile("In period", d?.in_period, href([]))}
          {tile(
            "High or critical",
            d?.high_impact,
            href([
              ["impact", "high"],
              ["impact", "critical"],
            ]),
          )}
          {tile(
            "In progress",
            d?.in_progress,
            href([["status", "in_progress"]]),
          )}
        </div>

        {d && (
          <>
            <BarList
              title="Logs per month"
              layout="columns"
              buckets={d.months}
              label={monthLabel}
              hrefFor={(b) => {
                const r = monthRange(b.key, range.from, range.to);
                return logsHref(id, [
                  ["from", r.from],
                  ["to", r.to],
                ]);
              }}
            />
            <div className="grid gap-4 lg:grid-cols-3">
              <div className="flex flex-col gap-4 lg:col-span-2">
                <BarList
                  title="Top tags"
                  layout="rows"
                  color="var(--chart-2)"
                  buckets={d.tags}
                  hrefFor={(b) => href([["tag", b.key]])}
                />
                <BarList
                  title="Status"
                  layout="rows"
                  color="var(--chart-3)"
                  buckets={d.statuses}
                  label={(k) => STATUS_LABEL[k as LogStatus] ?? k}
                  hrefFor={(b) => href([["status", b.key]])}
                />
                <BarList
                  title="Impact"
                  layout="rows"
                  color="var(--chart-4)"
                  buckets={d.impacts}
                  hrefFor={(b) => href([["impact", b.key]])}
                />
              </div>
              <section
                aria-label="Coverage of the article's sections"
                className="flex flex-col gap-2 rounded-lg glass p-4 shadow-glass"
              >
                <h3 className="text-sm font-medium">Coverage</h3>
                <p className="text-xs text-fg-secondary">
                  The brag document article's sections. Zero means nothing
                  logged there this period.
                </p>
                <ul className="flex flex-col gap-1 text-sm">
                  {d.coverage.map((b) => (
                    <li key={b.key}>
                      <Link
                        to={href([["tag", b.key]])}
                        data-zero={b.count === 0 ? "true" : undefined}
                        className={`flex justify-between hover:underline ${b.count === 0 ? "font-medium text-danger" : ""}`}
                      >
                        <span>{b.key}</span>
                        <span className="tabular-nums">{b.count}</span>
                      </Link>
                    </li>
                  ))}
                </ul>
              </section>
            </div>
          </>
        )}
      </DocumentPage>
    </div>
  );
}
