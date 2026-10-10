import * as React from "react";
import { cn } from "#lib/utils";
import { renderIcon, type IconProp } from "#components/icon";
import {
  Field,
  fieldControl,
  fieldFocusWithin,
  type FieldFrameProps,
} from "#components/field";

type TextFieldProps = Omit<React.ComponentProps<"input">, "children"> &
  FieldFrameProps & {
    /** Leading 20px icon: a node, or a built-in name ("mail", "search"). */
    icon?: IconProp;
    /** Trailing node, e.g. a unit or an icon button. */
    trailing?: React.ReactNode;
  };

function TextField({
  label,
  hint,
  error,
  icon,
  trailing,
  className,
  id,
  disabled,
  ...props
}: TextFieldProps) {
  const autoId = React.useId();
  const fid = id ?? autoId;
  return (
    <Field
      id={fid}
      label={label}
      hint={hint}
      error={error}
      disabled={disabled}
      className={className}
    >
      {(aria) => (
        <div
          data-slot="text-field"
          className={cn(
            fieldControl,
            fieldFocusWithin,
            "flex h-11 items-center gap-2 px-4",
          )}
        >
          {icon && (
            <span
              aria-hidden="true"
              className="inline-flex shrink-0 text-fg-primary [&_svg]:size-5"
            >
              {renderIcon(icon)}
            </span>
          )}
          <input
            type="text"
            disabled={disabled}
            {...props}
            {...aria}
            id={fid}
            className="h-full min-w-0 flex-1 border-0 bg-transparent type-body text-fg-primary outline-hidden placeholder:text-fg-secondary"
          />
          {trailing}
        </div>
      )}
    </Field>
  );
}

export { TextField, type TextFieldProps };
