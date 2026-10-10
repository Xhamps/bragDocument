import { useState, type FormEvent } from "react";
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  TextField,
} from "@bragdoc/ui";
import type { DocumentForm } from "./useDocuments";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Remounts the form when it changes (e.g. document id), so reopening on another document never shows stale state. */
  formKey: string;
  title: string;
  submitLabel: string;
  initial?: DocumentForm;
  busy?: boolean;
  error?: string | null;
  onSubmit: (form: DocumentForm) => void;
};

export function DocumentFormDialog({
  open,
  onOpenChange,
  formKey,
  ...rest
}: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* ponytail: DialogContent unmounts on close, so the form's state resets per open without an effect */}
      <DialogContent>
        <Form key={formKey} onCancel={() => onOpenChange(false)} {...rest} />
      </DialogContent>
    </Dialog>
  );
}

function Form({
  title,
  submitLabel,
  initial,
  busy,
  error,
  onSubmit,
  onCancel,
}: Omit<Props, "open" | "onOpenChange" | "formKey"> & {
  onCancel: () => void;
}) {
  const [form, setForm] = useState<DocumentForm>(
    initial ?? { title: "", description: "" },
  );

  const submit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit({ title: form.title.trim(), description: form.description });
  };

  return (
    <form onSubmit={submit} className="flex flex-col gap-4">
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription>
          A title is required; the description is optional.
        </DialogDescription>
      </DialogHeader>
      <TextField
        id="doc-title"
        label="Title"
        required
        maxLength={200}
        value={form.title}
        onChange={(e) => setForm({ ...form, title: e.target.value })}
      />
      <TextField
        id="doc-description"
        label="Description"
        maxLength={2000}
        value={form.description}
        onChange={(e) => setForm({ ...form, description: e.target.value })}
      />
      {error && (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      )}
      <DialogFooter>
        <Button type="button" variant="glass" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          variant="primary"
          type="submit"
          disabled={busy || !form.title.trim()}
        >
          {submitLabel}
        </Button>
      </DialogFooter>
    </form>
  );
}
