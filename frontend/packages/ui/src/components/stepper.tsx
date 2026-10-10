import * as React from "react";
import { cn } from "#lib/utils";
import { IconButton } from "#components/icon-button";

type StepperProps = {
  min?: number;
  max?: number;
  value?: number;
  defaultValue?: number;
  onChange?: (value: number) => void;
  /** Accessible name of what is counted ("Carry-on bags"). */
  label?: string;
  disabled?: boolean;
  className?: string;
};

const step = "size-8 [&>svg]:size-[18px]";

// DS order: blue plus, value, glass minus.
function Stepper({
  min = 0,
  max = 99,
  value,
  defaultValue,
  onChange,
  label = "Quantity",
  disabled,
  className,
}: StepperProps) {
  const [inner, setInner] = React.useState(defaultValue ?? min);
  const current = value ?? inner;

  function set(n: number) {
    const v = Math.min(max, Math.max(min, n));
    if (value === undefined) setInner(v);
    onChange?.(v);
  }

  return (
    <div
      role="group"
      aria-label={label}
      data-slot="stepper"
      className={cn("inline-flex items-center gap-3", className)}
    >
      <IconButton
        icon="plus"
        label={`Increase ${label}`}
        tooltip={false}
        variant="primary"
        size="sm"
        glow
        disabled={disabled || current >= max}
        onClick={() => set(current + 1)}
        className={cn(
          step,
          "border-button disabled:border-button-inactive disabled:bg-transparent disabled:text-button-inactive",
        )}
      />
      {/* Stable live region; only the inner span remounts to replay the tick. */}
      <output
        aria-live="polite"
        className="inline-block min-w-5 text-center type-title-3 text-fg-primary tabular-nums"
      >
        <span key={current} className="inline-block animate-tick">
          {current}
        </span>
      </output>
      <IconButton
        icon="minus"
        label={`Decrease ${label}`}
        tooltip={false}
        size="sm"
        disabled={disabled || current <= min}
        onClick={() => set(current - 1)}
        className={cn(step, "shadow-none")}
      />
    </div>
  );
}

export { Stepper, type StepperProps };
