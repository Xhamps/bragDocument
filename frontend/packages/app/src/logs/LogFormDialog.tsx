import { useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import Markdown from "react-markdown";
import { XIcon } from "lucide-react";
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Badge,
  cn,
} from "@bragdoc/ui";
import type { Log } from "../lib/types";
import {
  FIELD,
  IMPACTS,
  STATUSES,
  STATUS_LABEL,
  SUGGESTED_TAGS,
} from "./constants";
import { useTags, type LogForm } from "./useLogs";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Remounts the form when it changes, so it never shows stale state. */
  formKey: string;
  initial?: Log;
  busy?: boolean;
  error?: string | null;
  /** The last save found no impact in the text (PRD-0007 FR-4). */
  noImpact?: boolean;
  onSubmit: (form: LogForm) => void;
};

export function LogFormDialog({ open, onOpenChange, formKey, ...rest }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <Form key={formKey} onCancel={() => onOpenChange(false)} {...rest} />
      </DialogContent>
    </Dialog>
  );
}

/** YYYY-MM-DD in local time, for <input type="date">. */
function dateInput(iso: string) {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function Form({
  initial,
  busy,
  error,
  noImpact,
  onSubmit,
  onCancel,
}: Omit<Props, "open" | "onOpenChange" | "formKey"> & {
  onCancel: () => void;
}) {
  const [f, setF] = useState(() => ({
    name: initial?.name ?? "",
    description: initial?.description ?? "",
    impact: initial?.impact ?? ("medium" as const),
    status: initial?.status ?? ("done" as const),
    tags: initial?.tags ?? [],
    links: initial?.links ?? [],
    date: dateInput(initial?.created_at ?? new Date().toISOString()),
  }));
  const [startDate] = useState(f.date);
  const [tagDraft, setTagDraft] = useState("");
  const [preview, setPreview] = useState(false);
  const descRef = useRef<HTMLTextAreaElement>(null);
  const known = useTags().data ?? [];
  const options = [...new Set([...known, ...SUGGESTED_TAGS])].filter(
    (t) => !f.tags.includes(t),
  );

  const withDraft = (tags: string[]) => {
    const t = tagDraft.trim().toLowerCase();
    return t && !tags.includes(t) ? [...tags, t] : tags;
  };
  const addTag = () => {
    setF({ ...f, tags: withDraft(f.tags) });
    setTagDraft("");
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit({
      name: f.name.trim(),
      description: f.description,
      impact: f.impact,
      status: f.status,
      tags: withDraft(f.tags),
      links: f.links.filter((l) => l.url.trim()),
      // Noon local time keeps the chosen calendar day in every UTC offset up to ±12 h.
      ...(f.date !== startDate && {
        created_at: new Date(`${f.date}T12:00:00`).toISOString(),
      }),
    });
  };

  const submitOnModEnter = (e: KeyboardEvent<HTMLFormElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      e.currentTarget.requestSubmit();
    }
  };

  return (
    <form
      onSubmit={submit}
      onKeyDown={submitOnModEnter}
      aria-describedby={error ? "log-form-error" : undefined}
      className="flex flex-col gap-4"
    >
      <DialogHeader>
        <DialogTitle>{initial ? "Edit log" : "New log"}</DialogTitle>
        <DialogDescription>
          Say what you did and what changed because of it. Cmd/Ctrl+Enter saves.
        </DialogDescription>
      </DialogHeader>

      {noImpact && (
        <div
          role="status"
          className="flex flex-col gap-2 rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-100"
        >
          <p>
            Saved. We couldn't find an impact in the description. What changed
            because of this work?
          </p>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="w-fit"
            onClick={() => {
              setPreview(false);
              requestAnimationFrame(() => descRef.current?.focus());
            }}
          >
            Add impact
          </Button>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-[1fr_10rem]">
        <div className="flex flex-col gap-1">
          <Label htmlFor="log-name">Name</Label>
          <Input
            id="log-name"
            required
            maxLength={120}
            value={f.name}
            onChange={(e) => setF({ ...f, name: e.target.value })}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="log-impact">Impact</Label>
          <select
            id="log-impact"
            className={FIELD}
            value={f.impact}
            onChange={(e) =>
              setF({ ...f, impact: e.target.value as typeof f.impact })
            }
          >
            {IMPACTS.map((i) => (
              <option key={i} value={i}>
                {i}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <Label htmlFor="log-description">Description</Label>
          <span className="text-xs text-muted-foreground">
            Markdown. Include the result.
          </span>
          <div className="ml-auto flex gap-1">
            <Button
              type="button"
              size="sm"
              variant={preview ? "ghost" : "secondary"}
              aria-pressed={!preview}
              onClick={() => setPreview(false)}
            >
              Write
            </Button>
            <Button
              type="button"
              size="sm"
              variant={preview ? "secondary" : "ghost"}
              aria-pressed={preview}
              onClick={() => setPreview(true)}
            >
              Preview
            </Button>
          </div>
        </div>
        {preview ? (
          <div className="min-h-32 rounded-md border p-3 text-sm [&_a]:underline [&_ol]:list-decimal [&_ol]:pl-5 [&_ul]:list-disc [&_ul]:pl-5">
            <Markdown>{f.description || "_Nothing to preview._"}</Markdown>
          </div>
        ) : (
          <textarea
            id="log-description"
            ref={descRef}
            maxLength={20000}
            className={cn(FIELD, "min-h-32")}
            value={f.description}
            onChange={(e) => setF({ ...f, description: e.target.value })}
          />
        )}
      </div>

      <div className="flex flex-col gap-1">
        <Label htmlFor="log-tags">Tags</Label>
        {f.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {f.tags.map((t) => (
              <Badge key={t} variant="secondary">
                {t}
                <button
                  type="button"
                  aria-label={`Remove tag ${t}`}
                  onClick={() =>
                    setF({ ...f, tags: f.tags.filter((x) => x !== t) })
                  }
                >
                  <XIcon aria-hidden />
                </button>
              </Badge>
            ))}
          </div>
        )}
        <Input
          id="log-tags"
          list="log-tag-options"
          placeholder="Type and press Enter"
          value={tagDraft}
          onChange={(e) => setTagDraft(e.target.value)}
          onBlur={addTag}
          onKeyDown={(e) => {
            const plainEnter = e.key === "Enter" && !e.metaKey && !e.ctrlKey;
            if (plainEnter || e.key === ",") {
              e.preventDefault();
              addTag();
            }
          }}
        />
        <datalist id="log-tag-options">
          {options.map((t) => (
            <option key={t} value={t} />
          ))}
        </datalist>
      </div>

      <fieldset className="flex flex-col gap-2">
        <legend className="text-sm font-medium">Links</legend>
        {f.links.map((l, i) => (
          <div key={i} className="flex gap-2">
            <Input
              type="url"
              aria-label={`Link ${i + 1} URL`}
              placeholder="https://"
              value={l.url}
              onChange={(e) =>
                setF({
                  ...f,
                  links: f.links.map((x, j) =>
                    j === i ? { ...x, url: e.target.value } : x,
                  ),
                })
              }
            />
            <Input
              aria-label={`Link ${i + 1} label`}
              placeholder="Label"
              maxLength={100}
              className="w-40"
              value={l.label}
              onChange={(e) =>
                setF({
                  ...f,
                  links: f.links.map((x, j) =>
                    j === i ? { ...x, label: e.target.value } : x,
                  ),
                })
              }
            />
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={`Remove link ${i + 1}`}
              onClick={() =>
                setF({ ...f, links: f.links.filter((_, j) => j !== i) })
              }
            >
              <XIcon />
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="w-fit"
          onClick={() =>
            setF({ ...f, links: [...f.links, { url: "", label: "" }] })
          }
        >
          Add link
        </Button>
      </fieldset>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <Label htmlFor="log-status">Status</Label>
          <select
            id="log-status"
            className={FIELD}
            value={f.status}
            onChange={(e) =>
              setF({ ...f, status: e.target.value as typeof f.status })
            }
          >
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {STATUS_LABEL[s]}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="log-date">Date</Label>
          <Input
            id="log-date"
            type="date"
            required
            value={f.date}
            onChange={(e) => setF({ ...f, date: e.target.value })}
          />
        </div>
      </div>

      {error && (
        <p
          id="log-form-error"
          role="alert"
          className="text-sm text-destructive"
        >
          {error}
        </p>
      )}
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          {noImpact ? "Close" : "Cancel"}
        </Button>
        <Button type="submit" disabled={busy || !f.name.trim()}>
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}
