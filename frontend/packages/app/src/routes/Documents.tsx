import { useState } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@bragdoc/ui";
import { ApiError } from "../lib/api";
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

function errorText(e: unknown) {
  if (e instanceof ApiError)
    return e.fields ? Object.values(e.fields).join(", ") : e.message;
  return e instanceof Error ? e.message : null;
}

export function Component() {
  const { data, isPending, error } = useDocuments();
  const create = useCreateDocument();
  const update = useUpdateDocument();
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

  const grid = (docs: Document[]) => (
    <div className="grid gap-4 sm:grid-cols-2">
      {docs.map((d) => (
        <DocumentCard
          key={d.id}
          doc={d}
          onRename={setRenaming}
          onDelete={setDeleting}
          onToggleArchive={(doc) =>
            update.mutate({
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
        <label className="ml-auto flex items-center gap-2 text-sm text-muted-foreground">
          <input
            type="checkbox"
            checked={showArchived}
            onChange={(e) => setShowArchived(e.target.checked)}
          />
          Show archived
        </label>
        {data.owned.length > 0 && (
          <Button onClick={() => setCreating(true)}>New document</Button>
        )}
      </div>

      {data.owned.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>What is a brag document?</CardTitle>
            <CardDescription>
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
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => setCreating(true)}>
              Create your first document
            </Button>
          </CardContent>
        </Card>
      ) : (
        grid(owned)
      )}

      {shared.length > 0 && (
        <section className="flex flex-col gap-4">
          <h2 className="text-xl font-semibold">Shared with you</h2>
          {grid(shared)}
        </section>
      )}

      <DocumentFormDialog
        open={creating}
        onOpenChange={setCreating}
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
        onOpenChange={(o) => !o && setRenaming(null)}
        title="Rename document"
        submitLabel="Save"
        initial={
          renaming
            ? { title: renaming.title, description: renaming.description }
            : undefined
        }
        busy={update.isPending}
        error={errorText(update.error)}
        onSubmit={(form) =>
          renaming &&
          update.mutate(
            { id: renaming.id, ...form },
            { onSuccess: () => setRenaming(null) },
          )
        }
      />
      <DeleteDocumentDialog
        doc={deleting}
        busy={remove.isPending}
        onCancel={() => setDeleting(null)}
        onConfirm={(doc) =>
          remove.mutate(doc.id, { onSuccess: () => setDeleting(null) })
        }
      />
    </div>
  );
}
