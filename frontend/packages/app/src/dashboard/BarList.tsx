import { useState } from "react";
import { Link, useNavigate } from "react-router";
import {
  Bar,
  BarChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Button } from "@bragdoc/ui";
import type { Bucket } from "../lib/types";

type Props = {
  title: string;
  buckets: Bucket[];
  hrefFor: (b: Bucket) => string;
  label?: (key: string) => string;
  /** columns: one bar per x value (months); rows: horizontal bars (tags, status, impact). */
  layout: "columns" | "rows";
  color?: string;
};

/** A bar chart whose bars link to the filtered logs (FR-3), with a table of links as its text alternative (NFR-2). */
export function BarList({
  title,
  buckets,
  hrefFor,
  label = (k) => k,
  layout,
  color = "var(--chart-1)",
}: Props) {
  const [table, setTable] = useState(false);
  const navigate = useNavigate();
  const data = buckets.map((b) => ({ ...b, name: label(b.key) }));
  const rows = layout === "rows";
  const empty = buckets.every((b) => b.count === 0);
  const tick = { fill: "var(--muted-foreground)", fontSize: 12 };
  return (
    <section
      aria-label={title}
      className="ring-foreground/10 flex flex-col gap-2 rounded-xl p-4 ring-1"
    >
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">{title}</h3>
        {!empty && (
          <Button variant="tinted" size="sm" onClick={() => setTable(!table)}>
            {table ? "Show as chart" : "Show as table"}
          </Button>
        )}
      </div>
      {empty ? (
        <p className="text-muted-foreground text-sm">No logs in this period.</p>
      ) : table ? (
        <table className="text-sm">
          <thead>
            <tr className="text-muted-foreground">
              <th className="text-left font-normal">{title}</th>
              <th className="text-right font-normal">Logs</th>
            </tr>
          </thead>
          <tbody>
            {data.map((b) => (
              <tr key={b.key}>
                <td>
                  <Link className="hover:underline" to={hrefFor(b)}>
                    {b.name}
                  </Link>
                </td>
                <td className="text-right tabular-nums">{b.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <div style={{ height: rows ? Math.max(120, data.length * 28) : 220 }}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart
              data={data}
              layout={rows ? "vertical" : "horizontal"}
              accessibilityLayer
            >
              <XAxis
                type={rows ? "number" : "category"}
                dataKey={rows ? undefined : "name"}
                hide={rows}
                allowDecimals={false}
                tick={tick}
                tickLine={false}
                axisLine={false}
              />
              <YAxis
                type={rows ? "category" : "number"}
                dataKey={rows ? "name" : undefined}
                width={rows ? 128 : 32}
                allowDecimals={false}
                tick={tick}
                tickLine={false}
                axisLine={false}
              />
              <Tooltip
                cursor={{ fill: "var(--muted)" }}
                contentStyle={{
                  background: "var(--popover)",
                  color: "var(--popover-foreground)",
                  border: "1px solid var(--border)",
                }}
                itemStyle={{ color: "var(--popover-foreground)" }}
              />
              <Bar
                dataKey="count"
                name="Logs"
                fill={color}
                radius={4}
                cursor="pointer"
                onClick={(_, i) => navigate(hrefFor(buckets[i]))}
              />
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}
    </section>
  );
}
