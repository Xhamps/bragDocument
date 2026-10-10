import type { ReactNode } from "react";
import { Link } from "react-router";
import { Card, TextField, focusRing } from "@bragdoc/ui";
import { Logo } from "../layout/Logo";

export function AuthCard({
  title,
  description,
  onSubmit,
  children,
  footer,
}: {
  title: string;
  description: string;
  onSubmit: () => void;
  children: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <main className="mx-auto flex min-h-screen max-w-sm flex-col justify-center gap-6 p-4">
      <Link
        to="/"
        className={`${focusRing} inline-flex items-center gap-2 self-center rounded-sm text-[17px] font-semibold text-fg-primary no-underline`}
      >
        <Logo className="h-8" />
        Brag Document
      </Link>
      {/* Not Card's title prop: that renders an h3, and this page needs its h1. */}
      <Card className="flex w-full flex-col gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="type-title-2">{title}</h1>
          <p className="type-body text-fg-secondary">{description}</p>
        </div>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit();
          }}
          className="flex flex-col gap-3"
        >
          {children}
        </form>
        {footer && (
          <div className="flex flex-col gap-1 text-sm text-fg-secondary">
            {footer}
          </div>
        )}
      </Card>
    </main>
  );
}

export function Field({
  id,
  label,
  type,
  value,
  onChange,
  required,
}: {
  id: string;
  label: string;
  type: "email" | "password";
  value: string;
  onChange: (v: string) => void;
  required?: boolean;
}) {
  return (
    <TextField
      id={id}
      label={label}
      type={type}
      required={required}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}
