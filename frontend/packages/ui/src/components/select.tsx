import * as React from "react";
import { cn } from "#lib/utils";
import { Icon } from "#components/icon";
import {
  Field,
  fieldControl,
  fieldFocusWithin,
  type FieldFrameProps,
} from "#components/field";

type SelectOption = { value: string; label: React.ReactNode };

type SelectProps = Omit<React.ComponentProps<"select">, "children" | "prefix"> &
  FieldFrameProps & {
    options: ReadonlyArray<string | SelectOption>;
    /** Inline lead-in inside the pill ("Corner Radius:"). */
    prefix?: React.ReactNode;
    /** Adds a disabled empty first option ("Sort by"); shows only with `value=""` or `defaultValue=""`. */
    placeholder?: string;
  };

function Select({
  label,
  hint,
  error,
  options,
  prefix,
  placeholder,
  className,
  id,
  disabled,
  "aria-invalid": invalid,
  "aria-describedby": describedBy,
  ...props
}: SelectProps) {
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
      aria-invalid={invalid}
      aria-describedby={describedBy}
    >
      {(aria) => (
        <div
          data-slot="select"
          className={cn(
            fieldControl,
            fieldFocusWithin,
            "relative inline-flex h-11 items-center gap-1 rounded-pill pr-4 pl-5",
          )}
        >
          {prefix && (
            <span
              aria-hidden="true"
              className="type-body font-medium whitespace-nowrap"
            >
              {prefix}
            </span>
          )}
          <select
            disabled={disabled}
            {...props}
            {...aria}
            id={fid}
            className="min-w-0 flex-1 cursor-pointer appearance-none border-0 bg-transparent pr-7 type-body font-medium text-fg-primary outline-hidden [&>option]:bg-surface [&>option]:text-fg-primary"
          >
            {placeholder && (
              <option value="" disabled>
                {placeholder}
              </option>
            )}
            {options.map((o) => {
              const opt = typeof o === "string" ? { value: o, label: o } : o;
              return (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              );
            })}
          </select>
          <Icon
            name="chevronDown"
            size={20}
            className="pointer-events-none absolute right-4"
          />
        </div>
      )}
    </Field>
  );
}

export { Select, type SelectOption, type SelectProps };
