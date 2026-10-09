import { useState, type FormEvent } from "react";
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
} from "@bragdoc/ui";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { Document, GrantRole } from "../lib/types";
import { describeAudit } from "./audit";
import {
  useCancelInvitation,
  useChangeRole,
  useDocumentAudit,
  useRevoke,
  useShare,
  useSharing,
  useTransfer,
} from "./useSharing";

type Props = {
  doc: Document;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

// ponytail: native select; add a Select to @bragdoc/ui if more screens need one.
const selectClass = "h-8 rounded-md border bg-transparent px-2 text-sm";

function RoleOptions() {
  return (
    <>
      <option value="viewer">Viewer</option>
      <option value="editor">Editor</option>
    </>
  );
}

export function SharePanel({ doc, open, onOpenChange }: Props) {
  const sharing = useSharing(doc.id, open);
  const [showHistory, setShowHistory] = useState(false);
  const history = useDocumentAudit(doc.id, open && showHistory);
  const share = useShare(doc.id);
  const changeRole = useChangeRole(doc.id);
  const revoke = useRevoke(doc.id);
  const cancel = useCancelInvitation(doc.id);
  const transfer = useTransfer(doc.id);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<GrantRole>("viewer");
  const [notice, setNotice] = useState<string | null>(null);
  const [transferTo, setTransferTo] = useState<string | null>(null);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setNotice(null);
    share.mutate(
      { email, role },
      {
        onSuccess: (r) => {
          setEmail("");
          setNotice(
            r.kind === "grant"
              ? `Shared with ${email}.`
              : `Invited ${email}. They get access when they sign in.`,
          );
        },
      },
    );
  };
  const shareError =
    share.error instanceof ApiError && share.error.status === 409
      ? "Already shared with or invited."
      : errorText(share.error);
  const actionError = errorText(
    changeRole.error ?? revoke.error ?? cancel.error ?? transfer.error,
  );
  const target = sharing.data?.grants.find((g) => g.user_id === transferTo);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share “{doc.title}”</DialogTitle>
          <DialogDescription>
            Viewers read; editors also add and edit logs. Only the owner changes
            sharing.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="flex items-end gap-2">
          <div className="flex flex-1 flex-col gap-1">
            <Label htmlFor="share-email">Email</Label>
            <Input
              id="share-email"
              type="email"
              required
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (share.error) share.reset();
              }}
            />
          </div>
          <select
            aria-label="Role for new person"
            className={selectClass}
            value={role}
            onChange={(e) => setRole(e.target.value as GrantRole)}
          >
            <RoleOptions />
          </select>
          <Button type="submit" disabled={share.isPending}>
            Share
          </Button>
        </form>
        {shareError && (
          <p role="alert" className="text-sm text-destructive">
            {shareError}
          </p>
        )}
        {notice && (
          <p role="status" className="text-sm text-muted-foreground">
            {notice}
          </p>
        )}

        {sharing.error ? (
          <p role="alert" className="text-sm text-destructive">
            {errorText(sharing.error)}
          </p>
        ) : !sharing.data ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : (
          <ul className="divide-y rounded-md border">
            {sharing.data.grants.length + sharing.data.invitations.length ===
              0 && (
              <li className="p-3 text-sm text-muted-foreground">
                Only you have access.
              </li>
            )}
            {sharing.data.grants.map((g) => (
              <li
                key={g.user_id}
                className="flex items-center gap-2 p-3 text-sm"
              >
                <span className="flex-1 truncate" title={g.email}>
                  {g.display_name || g.email}
                </span>
                <select
                  aria-label={`Role for ${g.email}`}
                  className={selectClass}
                  value={g.role}
                  disabled={changeRole.isPending}
                  onChange={(e) =>
                    changeRole.mutate({
                      userId: g.user_id,
                      role: e.target.value as GrantRole,
                    })
                  }
                >
                  <RoleOptions />
                </select>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Make ${g.email} owner`}
                  onClick={() => setTransferTo(g.user_id)}
                >
                  Make owner
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Remove ${g.email}`}
                  disabled={revoke.isPending}
                  onClick={() => revoke.mutate(g.user_id)}
                >
                  Remove
                </Button>
              </li>
            ))}
            {sharing.data.invitations.map((i) => (
              <li key={i.id} className="flex items-center gap-2 p-3 text-sm">
                <span className="flex-1 truncate">{i.email}</span>
                <Badge variant="secondary">Pending · {i.role}</Badge>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Cancel invitation for ${i.email}`}
                  disabled={cancel.isPending}
                  onClick={() => cancel.mutate(i.id)}
                >
                  Cancel
                </Button>
              </li>
            ))}
          </ul>
        )}
        {actionError && (
          <p role="alert" className="text-sm text-destructive">
            {actionError}
          </p>
        )}

        {target && (
          <div className="flex flex-col gap-2 rounded-md border border-destructive/50 p-3 text-sm">
            <p>
              Make {target.display_name || target.email} the owner? You become
              an editor and can no longer change sharing.
            </p>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setTransferTo(null)}
              >
                Keep ownership
              </Button>
              <Button
                variant="destructive"
                size="sm"
                disabled={transfer.isPending}
                onClick={() =>
                  transfer.mutate(target.user_id, {
                    onSuccess: () => {
                      setTransferTo(null);
                      onOpenChange(false);
                    },
                  })
                }
              >
                Transfer ownership
              </Button>
            </div>
          </div>
        )}

        <details
          className="text-sm"
          onToggle={(e) => setShowHistory(e.currentTarget.open)}
        >
          <summary className="cursor-pointer text-muted-foreground">
            History
          </summary>
          <ul className="mt-2 flex flex-col gap-1">
            {history.data?.length === 0 && (
              <li className="text-muted-foreground">No sharing changes yet.</li>
            )}
            {history.data?.map((a) => (
              <li key={a.id}>
                <time className="text-muted-foreground">
                  {new Date(a.at).toLocaleDateString()}
                </time>{" "}
                {describeAudit(a)}
              </li>
            ))}
          </ul>
        </details>
      </DialogContent>
    </Dialog>
  );
}
