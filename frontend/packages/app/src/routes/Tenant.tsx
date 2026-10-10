import { useState, type FormEvent } from "react";
import { Navigate } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge, Button, Input, Label } from "@bragdoc/ui";
import { useMe } from "../auth/useMe";
import { api, ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { Invitation, Member } from "../lib/types";

export function Component() {
  const { data: me } = useMe();
  const qc = useQueryClient();
  const members = useQuery({
    queryKey: ["members"],
    queryFn: () => api<Member[]>("/tenant/members"),
    enabled: me?.role === "admin",
  });
  const invitations = useQuery({
    queryKey: ["invitations"],
    queryFn: () => api<Invitation[]>("/tenant/invitations"),
    enabled: me?.role === "admin",
  });
  const [email, setEmail] = useState("");

  const invite = useMutation({
    mutationFn: (email: string) =>
      api<Invitation>("/tenant/invitations", {
        method: "POST",
        body: JSON.stringify({ email }),
      }),
    onSuccess: () => {
      setEmail("");
      qc.invalidateQueries({ queryKey: ["invitations"] });
    },
  });
  const withdraw = useMutation({
    mutationFn: (id: string) =>
      api<void>(`/tenant/invitations/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invitations"] }),
  });
  const remove = useMutation({
    mutationFn: (id: string) =>
      api<void>(`/tenant/members/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["members"] }),
  });

  if (me && me.role !== "admin") return <Navigate to="/" replace />;
  if (!me || members.isPending || invitations.isPending)
    return <p className="text-muted-foreground">Loading…</p>;

  if (members.error || invitations.error)
    return (
      <p role="alert" className="text-destructive">
        {errorText(members.error ?? invitations.error)}
      </p>
    );

  const submit = (e: FormEvent) => {
    e.preventDefault();
    invite.mutate(email);
  };
  // ponytail: backend 409 body is generic; explain the one conflict each mutation can hit.
  const removeError =
    remove.error instanceof ApiError && remove.error.status === 409
      ? "Cannot remove: this member still owns documents."
      : errorText(remove.error);
  const inviteError =
    invite.error instanceof ApiError && invite.error.status === 409
      ? "This email is already a member or already invited."
      : errorText(invite.error);
  const withdrawError = errorText(withdraw.error);

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-3">
        <h2 className="text-xl font-semibold">{me.tenant.name}: members</h2>
        <ul className="ring-foreground/10 divide-y rounded-xl ring-1">
          {members.data.map((m) => (
            <li key={m.id} className="flex items-center gap-3 p-3 text-sm">
              <span>{m.display_name || m.email}</span>
              {m.display_name && (
                <span className="text-muted-foreground">{m.email}</span>
              )}
              <Badge variant="secondary">{m.role}</Badge>
              {m.id !== me.id && (
                <Button
                  className="ml-auto"
                  variant="tinted"
                  size="sm"
                  disabled={remove.isPending}
                  aria-label={`Remove ${m.email}`}
                  onClick={() => remove.mutate(m.id)}
                >
                  Remove
                </Button>
              )}
            </li>
          ))}
        </ul>
        {removeError && (
          <p role="alert" className="text-destructive text-sm">
            {removeError}
          </p>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-xl font-semibold">Invitations</h2>
        <p className="text-muted-foreground text-sm">
          Invited people join this tenant when they sign in with that email. No
          email is sent; share the sign-in link yourself.
        </p>
        <form onSubmit={submit} className="flex items-end gap-2">
          <div className="flex flex-1 flex-col gap-1">
            <Label htmlFor="invite-email">Invite by email</Label>
            <Input
              id="invite-email"
              type="email"
              required
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (invite.error) invite.reset();
              }}
            />
          </div>
          <Button variant="primary" type="submit" disabled={invite.isPending}>
            Invite
          </Button>
        </form>
        {inviteError && (
          <p role="alert" className="text-destructive text-sm">
            {inviteError}
          </p>
        )}
        <ul className="ring-foreground/10 divide-y rounded-xl ring-1">
          {invitations.data.length === 0 && (
            <li className="text-muted-foreground p-3 text-sm">
              No pending invitations.
            </li>
          )}
          {invitations.data.map((i) => (
            <li key={i.id} className="flex items-center gap-3 p-3 text-sm">
              <span>{i.email}</span>
              {i.document_title ? (
                <span className="text-muted-foreground ml-auto">
                  via “{i.document_title}”
                </span>
              ) : (
                <Button
                  className="ml-auto"
                  variant="tinted"
                  size="sm"
                  disabled={withdraw.isPending}
                  aria-label={`Withdraw ${i.email}`}
                  onClick={() => withdraw.mutate(i.id)}
                >
                  Withdraw
                </Button>
              )}
            </li>
          ))}
        </ul>
        {withdrawError && (
          <p role="alert" className="text-destructive text-sm">
            {withdrawError}
          </p>
        )}
      </section>
    </div>
  );
}
