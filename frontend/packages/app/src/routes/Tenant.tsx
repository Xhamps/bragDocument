import { useState, type FormEvent } from "react";
import { Navigate } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Tag, TextField } from "@bragdoc/ui";
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
    return <p className="text-fg-secondary">Loading…</p>;

  if (members.error || invitations.error)
    return (
      <p role="alert" className="text-danger">
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
        <h2 className="type-title-2">{me.tenant.name}: members</h2>
        <ul className="divide-y rounded-lg glass shadow-glass">
          {members.data.map((m) => (
            <li key={m.id} className="flex items-center gap-3 p-3 text-sm">
              <span>{m.display_name || m.email}</span>
              {m.display_name && (
                <span className="text-fg-secondary">{m.email}</span>
              )}
              <Tag tone={m.role === "admin" ? "success" : "neutral"}>
                {m.role}
              </Tag>
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
          <p role="alert" className="text-sm text-danger">
            {removeError}
          </p>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="type-title-2">Invitations</h2>
        <p className="text-sm text-fg-secondary">
          Invited people join this tenant when they sign in with that email. No
          email is sent; share the sign-in link yourself.
        </p>
        <form onSubmit={submit} className="flex items-end gap-2">
          <TextField
            id="invite-email"
            label="Invite by email"
            className="flex-1"
            type="email"
            required
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
              if (invite.error) invite.reset();
            }}
          />
          <Button variant="primary" type="submit" disabled={invite.isPending}>
            Invite
          </Button>
        </form>
        {inviteError && (
          <p role="alert" className="text-sm text-danger">
            {inviteError}
          </p>
        )}
        <ul className="divide-y rounded-lg glass shadow-glass">
          {invitations.data.length === 0 && (
            <li className="p-3 text-sm text-fg-secondary">
              No pending invitations.
            </li>
          )}
          {invitations.data.map((i) => (
            <li key={i.id} className="flex items-center gap-3 p-3 text-sm">
              <span>{i.email}</span>
              {i.document_title ? (
                <span className="ml-auto text-fg-secondary">
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
          <p role="alert" className="text-sm text-danger">
            {withdrawError}
          </p>
        )}
      </section>
    </div>
  );
}
