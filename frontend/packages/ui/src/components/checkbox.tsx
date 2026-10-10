import * as React from "react";
import { Checkbox as RadixCheckbox } from "radix-ui";
import { cn, focusRing } from "#lib/utils";
import { Icon } from "#components/icon";

type CheckboxProps = Omit<
  React.ComponentProps<typeof RadixCheckbox.Root>,
  "children"
> & {
  label?: React.ReactNode;
  /** Footnote line under the label. */
  description?: React.ReactNode;
  /** Alternative to `label`. */
  children?: React.ReactNode;
};

function Checkbox({
  label,
  description,
  children,
  className,
  disabled,
  ...props
}: CheckboxProps) {
  const id = React.useId();
  const text = label ?? children;
  return (
    <label
      data-slot="checkbox"
      className={cn(
        "group/check inline-flex cursor-pointer items-start gap-3 type-body text-fg-secondary",
        disabled && "cursor-default opacity-50",
        className,
      )}
    >
      <RadixCheckbox.Root
        disabled={disabled}
        aria-labelledby={text != null ? `${id}-label` : undefined}
        aria-describedby={description ? `${id}-desc` : undefined}
        {...props}
        className={cn(
          "peer inline-flex size-[22px] shrink-0 cursor-[inherit] items-center justify-center rounded-pill border-[1.5px] border-fg-secondary bg-transparent p-0 text-transparent",
          "transition-[background-color,border-color,box-shadow,scale] duration-base ease-standard group-active/check:scale-[0.88]",
          "data-checked:border-button data-checked:bg-button data-checked:text-button-fg data-checked:shadow-sm",
          "data-[state=indeterminate]:border-button data-[state=indeterminate]:bg-button data-[state=indeterminate]:text-button-fg",
          focusRing,
        )}
      >
        <RadixCheckbox.Indicator
          forceMount
          className="flex scale-[0.4] opacity-0 transition-[scale,opacity] duration-base ease-spring data-[state=indeterminate]:scale-100 data-[state=indeterminate]:opacity-100 data-checked:scale-100 data-checked:opacity-100"
        >
          <Icon name="check" size={14} />
        </RadixCheckbox.Indicator>
      </RadixCheckbox.Root>
      <span className="flex flex-col peer-data-checked:text-fg-primary">
        {text != null && <span id={`${id}-label`}>{text}</span>}
        {description && (
          <span id={`${id}-desc`} className="type-footnote text-fg-secondary">
            {description}
          </span>
        )}
      </span>
    </label>
  );
}

export { Checkbox, type CheckboxProps };
