import { useEffect, useEffectEvent, useState } from "react";
import { XIcon } from "lucide-react";
import { Badge, Button, Input } from "@bragdoc/ui";
import type { LogStatus } from "../lib/types";
import { FIELD, IMPACTS, SORTS, STATUSES, STATUS_LABEL } from "./constants";

type SetFilter = (key: string, values: string[]) => void;

/** Every control writes to the URL through onChange; the URL is the only state (PRD-0002 FR-6). */
export function LogFilters({
  params,
  onChange,
}: {
  params: URLSearchParams;
  onChange: SetFilter;
}) {
  const toggle = (key: string, value: string) => {
    const cur = params.getAll(key);
    onChange(
      key,
      cur.includes(value) ? cur.filter((v) => v !== value) : [...cur, value],
    );
  };
  const single = (key: string) => (value: string) =>
    onChange(key, value.trim() ? [value.trim()] : []);
  // Blur fires on every tab-past; only an actual change may reset page/push history.
  const commitDomain = (value: string) => {
    if (value.trim() !== (params.get("domain") ?? "")) single("domain")(value);
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        <SearchInput value={params.get("q") ?? ""} onCommit={single("q")} />
        <Input
          aria-label="Filter by tag"
          placeholder="Tag"
          className="w-36"
          onKeyDown={(e) => {
            const v = e.currentTarget.value.trim().toLowerCase();
            if (e.key === "Enter" && v) {
              if (!params.getAll("tag").includes(v))
                onChange("tag", [...params.getAll("tag"), v]);
              e.currentTarget.value = "";
            }
          }}
        />
        <Input
          key={params.get("domain") ?? ""} // remount when a chip clears it
          aria-label="Filter by link domain"
          placeholder="Domain, e.g. github.com"
          className="w-52"
          defaultValue={params.get("domain") ?? ""}
          onBlur={(e) => commitDomain(e.target.value)}
          onKeyDown={(e) =>
            e.key === "Enter" && commitDomain(e.currentTarget.value)
          }
        />
        <Input
          type="date"
          aria-label="From"
          className="w-40"
          value={params.get("from") ?? ""}
          onChange={(e) => single("from")(e.target.value)}
        />
        <Input
          type="date"
          aria-label="To"
          className="w-40"
          value={params.get("to") ?? ""}
          onChange={(e) => single("to")(e.target.value)}
        />
        <select
          aria-label="Sort"
          className={FIELD}
          value={params.get("sort") ?? "-created_at"}
          onChange={(e) =>
            onChange(
              "sort",
              e.target.value === "-created_at" ? [] : [e.target.value],
            )
          }
        >
          {SORTS.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-wrap items-center gap-4">
        <Toggles
          legend="Status"
          values={STATUSES}
          label={(s) => STATUS_LABEL[s as LogStatus]}
          active={params.getAll("status")}
          onToggle={(v) => toggle("status", v)}
        />
        <Toggles
          legend="Impact"
          values={IMPACTS}
          label={(i) => i}
          active={params.getAll("impact")}
          onToggle={(v) => toggle("impact", v)}
        />
      </div>
    </div>
  );
}

function Toggles({
  legend,
  values,
  label,
  active,
  onToggle,
}: {
  legend: string;
  values: string[];
  label: (v: string) => string;
  active: string[];
  onToggle: (v: string) => void;
}) {
  return (
    <fieldset className="flex items-center gap-1">
      <legend className="sr-only">{legend}</legend>
      <span aria-hidden className="mr-1 text-sm text-muted-foreground">
        {legend}
      </span>
      {values.map((v) => {
        const on = active.includes(v);
        return (
          <Button
            key={v}
            type="button"
            size="sm"
            variant={on ? "default" : "outline"}
            aria-pressed={on}
            onClick={() => onToggle(v)}
          >
            {label(v)}
          </Button>
        );
      })}
    </fieldset>
  );
}

/** Debounced search. Local text follows the URL when the URL changes elsewhere (chip, back button). */
function SearchInput({
  value,
  onCommit,
}: {
  value: string;
  onCommit: (v: string) => void;
}) {
  const [text, setText] = useState(value);
  const [seen, setSeen] = useState(value);
  if (value !== seen) {
    setSeen(value);
    if (value !== text.trim()) setText(value); // keep a trailing space being typed
  }
  const commit = useEffectEvent(onCommit);
  useEffect(() => {
    if (text.trim() === value) return;
    const t = setTimeout(() => commit(text), 300);
    return () => clearTimeout(t);
  }, [text, value]);
  return (
    <Input
      type="search"
      aria-label="Search logs"
      placeholder="Search name and description"
      className="w-64"
      value={text}
      onChange={(e) => setText(e.target.value)}
    />
  );
}

const LABELS: Record<string, string> = {
  q: "Text",
  tag: "Tag",
  status: "Status",
  impact: "Impact",
  from: "From",
  to: "To",
  domain: "Domain",
};

/** Removable chips for every active filter (PRD-0002 §9). */
export function ActiveFilters({
  params,
  onRemove,
  onClear,
}: {
  params: URLSearchParams;
  onRemove: (key: string, value: string) => void;
  onClear: () => void;
}) {
  const chips = [...params.entries()].filter(([k]) => Object.hasOwn(LABELS, k));
  if (chips.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-2">
      {chips.map(([k, v]) => (
        <Badge key={k + v} variant="secondary">
          {LABELS[k]}: {k === "status" ? STATUS_LABEL[v as LogStatus] : v}
          <button
            type="button"
            aria-label={`Remove filter ${LABELS[k]}: ${v}`}
            onClick={() => onRemove(k, v)}
          >
            <XIcon aria-hidden />
          </button>
        </Badge>
      ))}
      <Button type="button" variant="link" size="sm" onClick={onClear}>
        Clear all
      </Button>
    </div>
  );
}
