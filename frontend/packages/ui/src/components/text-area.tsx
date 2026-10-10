import * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { Field, fieldControl, type FieldFrameProps } from "#components/field";

type TextAreaProps = React.ComponentProps<"textarea"> & FieldFrameProps;

function TextArea({
  label,
  hint,
  error,
  className,
  id,
  disabled,
  ...props
}: TextAreaProps) {
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
        <textarea
          data-slot="text-area"
          rows={4}
          disabled={disabled}
          {...props}
          {...aria}
          id={fid}
          className={cn(
            fieldControl,
            focusRing,
            "block min-h-24 w-full resize-y px-4 py-3 type-body placeholder:text-fg-secondary focus:shadow-md focus:inset-shadow-ds",
          )}
        />
      )}
    </Field>
  );
}

export { TextArea, type TextAreaProps };
