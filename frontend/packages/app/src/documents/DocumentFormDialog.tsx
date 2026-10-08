import { useState, type FormEvent } from "react";
import {
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
} from "@bragdoc/ui";
import type { DocumentForm } from "./useDocuments";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  submitLabel: string;
  initial?: DocumentForm;
  busy?: boolean;
  error?: string | null;
  onSubmit: (form: DocumentForm) => void;
};

export function DocumentFormDialog({ open, onOpenChange, ...rest }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* ponytail: DialogContent unmounts on close, so the form's state resets per open without an effect */}
      <DialogContent>
        <Form onCancel={() => onOpenChange(false)} {...rest} />
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
}: Omit<Props, "open" | "onOpenChange"> & { onCancel: () => void }) {
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
      </DialogHeader>
      <div className="flex flex-col gap-1">
        <Label htmlFor="doc-title">Title</Label>
        <Input
          id="doc-title"
          required
          maxLength={200}
          value={form.title}
          onChange={(e) => setForm({ ...form, title: e.target.value })}
        />
      </div>
      <div className="flex flex-col gap-1">
        <Label htmlFor="doc-description">Description</Label>
        <Input
          id="doc-description"
          maxLength={2000}
          value={form.description}
          onChange={(e) => setForm({ ...form, description: e.target.value })}
        />
      </div>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={busy || !form.title.trim()}>
          {submitLabel}
        </Button>
      </DialogFooter>
    </form>
  );
}
