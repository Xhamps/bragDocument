import type * as React from "react";
import { RadioGroup as RadixRadioGroup } from "radix-ui";
import { cn, focusRing } from "#lib/utils";

type RadioOption = {
  value: string;
  label: React.ReactNode;
  disabled?: boolean;
};

type RadioGroupProps = {
  options: Array<string | RadioOption>;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
  /** column (filters) or row (Roundtrip / One way / Multi-City). */
  direction?: "column" | "row";
  name?: string;
  /** Accessible name of the group. */
  label?: string;
  disabled?: boolean;
  className?: string;
};

function RadioGroup({
  options,
  value,
  defaultValue,
  onChange,
  direction = "column",
  name,
  label,
  disabled,
  className,
}: RadioGroupProps) {
  const opts = options.map((o) =>
    typeof o === "string" ? { value: o, label: o } : o,
  );
  return (
    <RadixRadioGroup.Root
      data-slot="radio-group"
      aria-label={label}
      name={name}
      value={value}
      // DS: always preselect; the first option when no default is given.
      defaultValue={
        value === undefined ? (defaultValue ?? opts[0]?.value) : undefined
      }
      onValueChange={onChange}
      disabled={disabled}
      className={cn(
        "flex",
        direction === "row" ? "flex-row flex-wrap gap-6" : "flex-col gap-4",
        className,
      )}
    >
      {opts.map((o) => (
        <label
          key={o.value}
          className={cn(
            "group/check inline-flex cursor-pointer items-start gap-3 type-body text-fg-secondary",
            (disabled || o.disabled) && "cursor-default opacity-50",
          )}
        >
          <RadixRadioGroup.Item
            value={o.value}
            disabled={o.disabled}
            className={cn(
              "peer inline-flex size-[22px] shrink-0 cursor-[inherit] items-center justify-center rounded-pill border-[1.5px] border-fg-secondary bg-transparent p-0",
              "transition-[border-color,scale] duration-base ease-standard group-active/check:scale-[0.88] data-checked:border-button-text",
              focusRing,
            )}
          >
            <RadixRadioGroup.Indicator
              forceMount
              className="size-3 scale-0 rounded-pill bg-button-text transition-transform duration-base ease-spring data-checked:scale-100"
            />
          </RadixRadioGroup.Item>
          <span className="peer-data-checked:text-fg-primary">{o.label}</span>
        </label>
      ))}
    </RadixRadioGroup.Root>
  );
}

export { RadioGroup, type RadioGroupProps, type RadioOption };
