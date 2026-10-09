import { Link, useParams, useSearchParams } from "react-router";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { LogStatus } from "../lib/types";
import { DocumentTabs } from "../documents/DocumentTabs";
import { useDocument } from "../documents/useDocuments";
import { FIELD, STATUS_LABEL } from "../logs/constants";
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

const monthLabel = (key: string) =>
  new Date(`${key}-01T00:00:00Z`).toLocaleDateString("en-US", {
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  });

export function Component() {
  const { id = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const period = (params.get("period") ?? "12m") as Preset | "custom";
  const fallback = presetRange("12m");
  const range: Range =
    period === "custom"
      ? {
          from: params.get("from") ?? fallback.from,
          to: params.get("to") ?? fallback.to,
        }
      : presetRange(period);
  const doc = useDocument(id);
  const dash = useDashboard(id, range);

  if (doc.isPending) return <p className="text-muted-foreground">Loading…</p>;
  if (!doc.data)
    return (
      <p role="alert" className="text-destructive">
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
      className="flex flex-col gap-1 rounded-xl p-4 ring-1 ring-foreground/10 hover:bg-muted"
    >
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="text-2xl font-semibold tabular-nums">
        {value ?? "–"}
      </span>
    </Link>
  );

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-4">
        <Link to="/" className="text-sm text-muted-foreground hover:underline">
          ← Documents
        </Link>
        <h2 className="text-xl font-semibold">{doc.data.title}</h2>
        <DocumentTabs id={id} current="dashboard" />
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <label htmlFor="period" className="text-sm text-muted-foreground">
            Period
          </label>
          <select
            id="period"
            className={FIELD}
            value={period}
            onChange={(e) => {
              const v = e.target.value;
              setParams(
                v === "custom"
                  ? { period: v, from: range.from, to: range.to }
                  : { period: v },
              );
            }}
          >
            {PRESETS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
          {period === "custom" && (
            <>
              <input
                aria-label="From"
                type="date"
                className={FIELD}
                value={range.from}
                onChange={(e) =>
                  e.target.value &&
                  setParams({ period, from: e.target.value, to: range.to })
                }
              />
              <input
                aria-label="To"
                type="date"
                className={FIELD}
                value={range.to}
                onChange={(e) =>
                  e.target.value &&
                  setParams({ period, from: range.from, to: e.target.value })
                }
              />
            </>
          )}
        </div>
      </div>

      {dash.error && (
        <p role="alert" className="text-destructive">
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
        {tile("In progress", d?.in_progress, href([["status", "in_progress"]]))}
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
                label={(k) => STATUS_LABEL[k as LogStatus]}
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
              className="flex flex-col gap-2 rounded-xl p-4 ring-1 ring-foreground/10"
            >
              <h3 className="text-sm font-medium">Coverage</h3>
              <p className="text-xs text-muted-foreground">
                The brag document article's sections. Zero means nothing logged
                there this period.
              </p>
              <ul className="flex flex-col gap-1 text-sm">
                {d.coverage.map((b) => (
                  <li key={b.key}>
                    <Link
                      to={href([["tag", b.key]])}
                      data-zero={b.count === 0 ? "true" : undefined}
                      className={`flex justify-between hover:underline ${b.count === 0 ? "font-medium text-destructive" : ""}`}
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
    </div>
  );
}
