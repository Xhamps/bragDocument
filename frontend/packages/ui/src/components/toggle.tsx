import type * as React from "react";
import { Switch } from "radix-ui";
import { cn, focusRing } from "#lib/utils";

type ToggleProps = {
  checked?: boolean;
  defaultChecked?: boolean;
  onChange?: (checked: boolean) => void;
  label?: React.ReactNode;
  disabled?: boolean;
  className?: string;
};

function Toggle({
  checked,
  defaultChecked,
  onChange,
  label,
  disabled,
  className,
}: ToggleProps) {
  return (
    <label
      data-slot="toggle"
      className={cn(
        "group/toggle inline-flex cursor-pointer items-center gap-3 type-body text-fg-secondary",
        disabled && "cursor-default",
        className,
      )}
    >
      {label && <span>{label}</span>}
      <Switch.Root
        checked={checked}
        defaultChecked={defaultChecked}
        onCheckedChange={onChange}
        disabled={disabled}
        className={cn(
          "relative h-[30px] w-[52px] shrink-0 cursor-[inherit] rounded-pill border-container-border glass p-0 inset-shadow-ds",
          "transition-[background-color,border-color] duration-base ease-standard",
          "disabled:opacity-50 data-checked:border-button data-checked:bg-button",
          focusRing,
        )}
      >
        <Switch.Thumb
          className={cn(
            "absolute top-[3px] left-[3px] block size-[22px] rounded-pill bg-fg-tertiary shadow-sm",
            "transition-[translate,background-color,width] duration-base ease-spring group-active/toggle:w-[26px]",
            "data-checked:translate-x-[22px] data-checked:bg-button-fg data-checked:shadow-md group-active/toggle:data-checked:translate-x-[18px]",
          )}
        />
      </Switch.Root>
    </label>
  );
}

export { Toggle, type ToggleProps };
