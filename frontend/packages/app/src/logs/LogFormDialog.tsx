import { useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import Markdown from "react-markdown";
import { XIcon } from "lucide-react";
import {
  Button,
  IconButton,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Select,
  Tag,
  TextArea,
  TextField,
  focusRing,
} from "@bragdoc/ui";
import type { Log } from "../lib/types";
import { IMPACTS, STATUSES, STATUS_LABEL, SUGGESTED_TAGS } from "./constants";
import { ExternalLink } from "./LogRow";
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
    // Cmd/Ctrl+Enter's requestSubmit ignores the disabled Save button.
    if (busy || !f.name.trim()) return;
    onSubmit({
      name: f.name.trim(),
      description: f.description,
      impact: f.impact,
      status: f.status,
      tags: withDraft(f.tags),
      links: f.links.filter((l) => l.url.trim()),
      // Noon local time keeps the chosen calendar day for UTC offsets from -12 h to +11 h.
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
          className="flex flex-col gap-2 rounded-md border border-danger bg-glass-tint-rose p-3 type-footnote text-fg-primary"
        >
          <p>
            Saved. We couldn't find an impact in the description. What changed
            because of this work?
          </p>
          <div className="flex gap-2">
            <Button
              type="button"
              size="sm"
              variant="glass"
              onClick={() => {
                setPreview(false);
                requestAnimationFrame(() => descRef.current?.focus());
              }}
            >
              Add impact
            </Button>
            <Button type="button" size="sm" variant="tinted" onClick={onCancel}>
              Close
            </Button>
          </div>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-[1fr_10rem]">
        <TextField
          id="log-name"
          label="Name"
          required
          maxLength={120}
          value={f.name}
          onChange={(e) => setF({ ...f, name: e.target.value })}
        />
        <Select
          id="log-impact"
          label="Impact"
          options={IMPACTS}
          value={f.impact}
          onChange={(e) =>
            setF({ ...f, impact: e.target.value as typeof f.impact })
          }
        />
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <label
            htmlFor="log-description"
            className="type-footnote font-medium text-fg-secondary"
          >
            Description
          </label>
          <span className="text-xs text-fg-secondary">
            Markdown. Include the result.
          </span>
          <div className="ml-auto flex gap-1">
            <Button
              type="button"
              size="sm"
              variant={preview ? "tinted" : "glass"}
              aria-pressed={!preview}
              onClick={() => setPreview(false)}
            >
              Write
            </Button>
            <Button
              type="button"
              size="sm"
              variant={preview ? "glass" : "tinted"}
              aria-pressed={preview}
              onClick={() => setPreview(true)}
            >
              Preview
            </Button>
          </div>
        </div>
        {preview ? (
          <div className="min-h-32 rounded-md border p-3 text-sm [&_a]:underline [&_ol]:list-decimal [&_ol]:pl-5 [&_ul]:list-disc [&_ul]:pl-5">
            <Markdown
              components={{ a: ExternalLink }}
              disallowedElements={["img"]}
              unwrapDisallowed
            >
              {f.description || "_Nothing to preview._"}
            </Markdown>
          </div>
        ) : (
          <TextArea
            id="log-description"
            ref={descRef}
            maxLength={20000}
            rows={6}
            value={f.description}
            onChange={(e) => setF({ ...f, description: e.target.value })}
          />
        )}
      </div>

      <div className="flex flex-col gap-1">
        <TextField
          id="log-tags"
          label="Tags"
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
        {f.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {f.tags.map((t) => (
              <Tag key={t}>
                {t}
                <button
                  type="button"
                  aria-label={`Remove tag ${t}`}
                  className={`${focusRing} rounded-sm`}
                  onClick={() =>
                    setF({ ...f, tags: f.tags.filter((x) => x !== t) })
                  }
                >
                  <XIcon aria-hidden className="size-3" />
                </button>
              </Tag>
            ))}
          </div>
        )}
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
            <TextField
              type="url"
              className="flex-1"
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
            <TextField
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
            <IconButton
              type="button"
              variant="tinted"
              size="sm"
              icon="close"
              label={`Remove link ${i + 1}`}
              onClick={() =>
                setF({ ...f, links: f.links.filter((_, j) => j !== i) })
              }
            />
          </div>
        ))}
        <Button
          type="button"
          variant="glass"
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
        <Select
          id="log-status"
          label="Status"
          options={STATUSES.map((s) => ({ value: s, label: STATUS_LABEL[s] }))}
          value={f.status}
          onChange={(e) =>
            setF({ ...f, status: e.target.value as typeof f.status })
          }
        />
        <TextField
          id="log-date"
          label="Date"
          type="date"
          required
          value={f.date}
          onChange={(e) => setF({ ...f, date: e.target.value })}
        />
      </div>

      {error && (
        <p id="log-form-error" role="alert" className="text-sm text-danger">
          {error}
        </p>
      )}
      <DialogFooter>
        <Button type="button" variant="glass" onClick={onCancel}>
          {noImpact ? "Close" : "Cancel"}
        </Button>
        <Button
          variant="primary"
          type="submit"
          disabled={busy || !f.name.trim()}
        >
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}
