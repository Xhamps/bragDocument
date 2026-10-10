import type { ReactNode } from "react";
import { Card, TextField } from "@bragdoc/ui";

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
    <main className="mx-auto flex min-h-screen max-w-sm items-center p-4">
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
