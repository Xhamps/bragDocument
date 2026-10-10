import { useState } from "react";
import { Button, Card } from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import type { Document } from "../lib/types";
import { DocumentCard } from "../documents/DocumentCard";
import { DocumentFormDialog } from "../documents/DocumentFormDialog";
import { DeleteDocumentDialog } from "../documents/DeleteDocumentDialog";
import {
  useCreateDocument,
  useDeleteDocument,
  useDocuments,
  useUpdateDocument,
} from "../documents/useDocuments";

const ARTICLE = "https://jvns.ca/blog/brag-documents/";

export function Component() {
  const { data, isPending, error } = useDocuments();
  const create = useCreateDocument();
  const rename = useUpdateDocument();
  const archive = useUpdateDocument();
  const remove = useDeleteDocument();
  const [creating, setCreating] = useState(false);
  const [renaming, setRenaming] = useState<Document | null>(null);
  const [deleting, setDeleting] = useState<Document | null>(null);
  const [showArchived, setShowArchived] = useState(false);

  if (isPending) return <p className="text-muted-foreground">Loading…</p>;
  if (error)
    return (
      <p role="alert" className="text-destructive">
        {errorText(error)}
      </p>
    );

  const owned = data.owned.filter((d) => showArchived || d.state === "active");
  const shared = data.shared;

  const archiveError = errorText(archive.error);

  const grid = (docs: Document[], readOnly = false) => (
    <div className="grid gap-4 sm:grid-cols-2">
      {docs.map((d) => (
        <DocumentCard
          key={d.id}
          doc={d}
          readOnly={readOnly}
          onRename={setRenaming}
          onDelete={setDeleting}
          onToggleArchive={(doc) =>
            archive.mutate({
              id: doc.id,
              state: doc.state === "active" ? "archived" : "active",
            })
          }
        />
      ))}
    </div>
  );

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-4">
        <h2 className="text-xl font-semibold">Your documents</h2>
        <label className="text-muted-foreground ml-auto flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={showArchived}
            onChange={(e) => setShowArchived(e.target.checked)}
          />
          Show archived
        </label>
        {data.owned.length > 0 && (
          <Button variant="primary" onClick={() => setCreating(true)}>
            New document
          </Button>
        )}
      </div>

      {archiveError && (
        <p role="alert" className="text-destructive text-sm">
          {archiveError}
        </p>
      )}

      {data.owned.length === 0 ? (
        <Card
          title="What is a brag document?"
          description={
            <>
              A running record of the work you did and why it mattered, so
              reviews and promotions are not a memory test. Read{" "}
              <a
                className="underline"
                href={ARTICLE}
                target="_blank"
                rel="noreferrer"
              >
                jvns.ca: Get your work recognized
              </a>
              .
            </>
          }
        >
          <Button
            variant="primary"
            className="mt-4"
            onClick={() => setCreating(true)}
          >
            Create your first document
          </Button>
        </Card>
      ) : owned.length === 0 ? (
        <p className="text-muted-foreground">No active documents.</p>
      ) : (
        grid(owned)
      )}

      {shared.length > 0 && (
        <section className="flex flex-col gap-4">
          <h2 className="text-xl font-semibold">Shared with you</h2>
          {grid(shared, true)}
        </section>
      )}

      <DocumentFormDialog
        open={creating}
        formKey="new"
        onOpenChange={(o) => {
          setCreating(o);
          if (!o) create.reset();
        }}
        title="New document"
        submitLabel="Create"
        busy={create.isPending}
        error={errorText(create.error)}
        onSubmit={(form) =>
          create.mutate(form, { onSuccess: () => setCreating(false) })
        }
      />
      <DocumentFormDialog
        open={renaming !== null}
        formKey={renaming?.id ?? "none"}
        onOpenChange={(o) => {
          if (o) return;
          setRenaming(null);
          rename.reset();
        }}
        title="Rename document"
        submitLabel="Save"
        initial={
          renaming
            ? { title: renaming.title, description: renaming.description }
            : undefined
        }
        busy={rename.isPending}
        error={errorText(rename.error)}
        onSubmit={(form) =>
          renaming &&
          rename.mutate(
            { id: renaming.id, ...form },
            { onSuccess: () => setRenaming(null) },
          )
        }
      />
      <DeleteDocumentDialog
        doc={deleting}
        busy={remove.isPending}
        error={errorText(remove.error)}
        onCancel={() => {
          setDeleting(null);
          remove.reset();
        }}
        onConfirm={(doc) =>
          remove.mutate(doc.id, { onSuccess: () => setDeleting(null) })
        }
      />
    </div>
  );
}
