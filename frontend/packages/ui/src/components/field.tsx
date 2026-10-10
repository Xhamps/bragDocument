import type * as React from "react";
import { cn } from "#lib/utils";

type FieldFrameProps = {
  /** Visible label above the control. */
  label?: React.ReactNode;
  /** Helper line under the control. */
  hint?: React.ReactNode;
  /** Error message; replaces the hint, sets the danger border and aria-invalid. */
  error?: React.ReactNode;
};

type FieldAria = Pick<
  React.AriaAttributes,
  "aria-invalid" | "aria-describedby"
>;

// Shared label / hint / error frame for TextField, TextArea and Select.
function Field({
  id,
  label,
  hint,
  error,
  disabled,
  className,
  children,
  "aria-invalid": invalid,
  "aria-describedby": describedBy,
}: FieldFrameProps &
  FieldAria & {
    id: string;
    disabled?: boolean;
    className?: string;
    children: (aria: FieldAria) => React.ReactNode;
  }) {
  const msgId = `${id}-msg`;
  return (
    <div
      data-slot="field"
      data-invalid={error ? "" : undefined}
      className={cn(
        "group/field flex min-w-0 flex-col gap-1.5",
        disabled && "pointer-events-none opacity-50",
        className,
      )}
    >
      {label && (
        <label
          htmlFor={id}
          className="type-footnote font-medium text-fg-secondary"
        >
          {label}
        </label>
      )}
      {children({
        // The frame's error wins; otherwise the caller's aria-invalid stands.
        "aria-invalid": error ? true : invalid,
        "aria-describedby":
          [describedBy, error || hint ? msgId : undefined]
            .filter(Boolean)
            .join(" ") || undefined,
      })}
      {error ? (
        <div id={msgId} role="alert" className="type-footnote text-danger">
          {error}
        </div>
      ) : hint ? (
        <div id={msgId} className="type-footnote text-fg-secondary">
          {hint}
        </div>
      ) : null}
    </div>
  );
}

// Glass box shared by the controls (the DS .bd-glass includes shadow-glass).
const fieldControl =
  "glass border-container-border text-fg-primary shadow-glass rounded-md transition-[border-color,box-shadow] duration-base ease-standard group-data-invalid/field:border-danger";

// Focus ring + pressed-in shadow for wrappers around the real control.
const fieldFocusWithin =
  "focus-within:outline-2 focus-within:outline-solid focus-within:outline-offset-2 focus-within:outline-focus-ring focus-within:inset-shadow-ds focus-within:shadow-md";

export { Field, fieldControl, fieldFocusWithin, type FieldFrameProps };
