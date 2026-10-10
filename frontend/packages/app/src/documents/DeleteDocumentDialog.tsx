import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@bragdoc/ui";
import type { Document } from "../lib/types";

type Props = {
  doc: Document | null;
  busy?: boolean;
  error?: string | null;
  onCancel: () => void;
  onConfirm: (doc: Document) => void;
};

export function DeleteDocumentDialog({
  doc,
  busy,
  error,
  onCancel,
  onConfirm,
}: Props) {
  return (
    <Dialog open={doc !== null} onOpenChange={(o) => !o && onCancel()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete "{doc?.title}"?</DialogTitle>
          <DialogDescription>
            All its logs and sharing grants are deleted too. This cannot be
            undone.
          </DialogDescription>
        </DialogHeader>
        {error && (
          <p role="alert" className="text-destructive text-sm">
            {error}
          </p>
        )}
        <DialogFooter>
          <Button variant="glass" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            variant="danger"
            disabled={busy}
            onClick={() => doc && onConfirm(doc)}
          >
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
