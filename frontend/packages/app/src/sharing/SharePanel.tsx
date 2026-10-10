import { useState, type FormEvent } from "react";
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Select,
  Tag,
  TextField,
} from "@bragdoc/ui";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { Document, GrantRole } from "../lib/types";
import {
  useCancelInvitation,
  useChangeRole,
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

const roleOptions = [
  { value: "viewer", label: "Viewer" },
  { value: "editor", label: "Editor" },
];

export function SharePanel({ doc, open, onOpenChange }: Props) {
  const sharing = useSharing(doc.id, open);
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
  // Reset before each row action so the error shown is the latest action's.
  const resetActions = () => {
    changeRole.reset();
    revoke.reset();
    cancel.reset();
    transfer.reset();
  };
  // Closing starts the next open fresh.
  const setOpen = (o: boolean) => {
    if (!o) {
      setEmail("");
      setRole("viewer");
      setNotice(null);
      setTransferTo(null);
      share.reset();
      resetActions();
    }
    onOpenChange(o);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share “{doc.title}”</DialogTitle>
          <DialogDescription>
            Viewers read; editors also add and edit logs. Only the owner changes
            sharing.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="flex items-end gap-2">
          <TextField
            id="share-email"
            label="Email"
            className="flex-1"
            type="email"
            required
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
              if (share.error) share.reset();
            }}
          />
          <Select
            aria-label="Role for new person"
            options={roleOptions}
            value={role}
            onChange={(e) => setRole(e.target.value as GrantRole)}
          />
          <Button variant="primary" type="submit" disabled={share.isPending}>
            Share
          </Button>
        </form>
        {shareError && (
          <p role="alert" className="text-sm text-danger">
            {shareError}
          </p>
        )}
        {notice && (
          <p role="status" className="text-sm text-fg-secondary">
            {notice}
          </p>
        )}

        {sharing.error ? (
          <p role="alert" className="text-sm text-danger">
            {errorText(sharing.error)}
          </p>
        ) : !sharing.data ? (
          <p className="text-sm text-fg-secondary">Loading…</p>
        ) : (
          <ul className="divide-y rounded-md border">
            {sharing.data.grants.length + sharing.data.invitations.length ===
              0 && (
              <li className="p-3 text-sm text-fg-secondary">
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
                <Select
                  aria-label={`Role for ${g.email}`}
                  options={roleOptions}
                  value={g.role}
                  disabled={changeRole.isPending}
                  onChange={(e) => {
                    resetActions();
                    changeRole.mutate({
                      userId: g.user_id,
                      role: e.target.value as GrantRole,
                    });
                  }}
                />
                <Button
                  variant="tinted"
                  size="sm"
                  aria-label={`Make ${g.email} owner`}
                  onClick={() => setTransferTo(g.user_id)}
                >
                  Make owner
                </Button>
                <Button
                  variant="tinted"
                  size="sm"
                  aria-label={`Remove ${g.email}`}
                  disabled={revoke.isPending}
                  onClick={() => {
                    resetActions();
                    revoke.mutate(g.user_id);
                  }}
                >
                  Remove
                </Button>
              </li>
            ))}
            {sharing.data.invitations.map((i) => (
              <li key={i.id} className="flex items-center gap-2 p-3 text-sm">
                <span className="flex-1 truncate">{i.email}</span>
                <Tag>Pending · {i.role}</Tag>
                <Button
                  variant="tinted"
                  size="sm"
                  aria-label={`Cancel invitation for ${i.email}`}
                  disabled={cancel.isPending}
                  onClick={() => {
                    resetActions();
                    cancel.mutate(i.id);
                  }}
                >
                  Cancel
                </Button>
              </li>
            ))}
          </ul>
        )}
        {actionError && (
          <p role="alert" className="text-sm text-danger">
            {actionError}
          </p>
        )}

        {target && (
          <div
            role="alertdialog"
            aria-label="Confirm ownership transfer"
            aria-describedby="transfer-confirm-text"
            className="flex flex-col gap-2 rounded-md border border-danger p-3 text-sm"
          >
            <p id="transfer-confirm-text">
              Make {target.display_name || target.email} the owner? You become
              an editor and can no longer change sharing.
            </p>
            <div className="flex justify-end gap-2">
              <Button
                variant="glass"
                size="sm"
                onClick={() => setTransferTo(null)}
              >
                Keep ownership
              </Button>
              <Button
                variant="danger"
                size="sm"
                disabled={transfer.isPending}
                onClick={() => {
                  resetActions();
                  transfer.mutate(target.user_id, {
                    onSuccess: () => setOpen(false),
                  });
                }}
              >
                Transfer ownership
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
