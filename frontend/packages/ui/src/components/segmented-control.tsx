import * as React from "react";
import { cn } from "#lib/utils";
import { useIndicator } from "#lib/use-indicator";

type SegmentedOption = { value: string; label: React.ReactNode };

type SegmentedControlProps = {
  options: Array<string | SegmentedOption>;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
  /** neutral = raised glass pill; accent = selected label in `button-text` with a sliding underline. */
  tone?: "neutral" | "accent";
  size?: "sm" | "md";
  /** Accessible name of the group. */
  label?: string;
  name?: string;
  className?: string;
};

// Native radios (sr-only) give arrow keys, roving tab stop and checked state.
function SegmentedControl({
  options,
  value,
  defaultValue,
  onChange,
  tone = "neutral",
  size = "md",
  label,
  name,
  className,
}: SegmentedControlProps) {
  const opts = options.map((o) =>
    typeof o === "string" ? { value: o, label: o } : o,
  );
  const [inner, setInner] = React.useState(defaultValue ?? opts[0]?.value);
  const current = value ?? inner;
  const autoName = React.useId();
  const { ref, style, animated } = useIndicator([current, opts.length, size]);

  function pick(v: string) {
    if (value === undefined) setInner(v);
    onChange?.(v);
  }

  return (
    <div
      ref={ref}
      role="radiogroup"
      aria-label={label}
      data-slot="segmented-control"
      className={cn(
        "relative isolate inline-flex gap-0.5 rounded-pill glass p-[3px] shadow-glass inset-shadow-ds",
        className,
      )}
    >
      <span
        aria-hidden="true"
        style={style}
        className={cn(
          "pointer-events-none absolute top-0 left-0 z-0 box-border rounded-pill will-change-[transform,width]",
          tone === "neutral"
            ? "glass shadow-md"
            : "after:absolute after:inset-x-1/4 after:bottom-[3px] after:h-0.5 after:rounded-[2px] after:bg-button-text",
          animated &&
            "[transition:transform_var(--duration-slow)_var(--ease-spring),width_var(--duration-slow)_var(--ease-spring),height_var(--duration-slow)_var(--ease-spring),opacity_var(--duration-fast)_var(--ease-standard)]",
        )}
      />
      {opts.map((o) => {
        const on = o.value === current;
        return (
          <label
            key={o.value}
            data-on={on}
            className={cn(
              "relative z-1 inline-flex cursor-pointer items-center rounded-pill border border-transparent type-callout text-fg-secondary transition-colors duration-fast ease-standard select-none hover:text-fg-primary",
              "has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-focus-ring has-focus-visible:outline-solid",
              size === "sm"
                ? "h-6 px-3 type-footnote font-medium"
                : "h-[30px] px-4",
              on &&
                (tone === "accent" ? "text-button-text" : "text-fg-primary"),
            )}
          >
            <input
              type="radio"
              className="sr-only"
              name={name ?? autoName}
              value={o.value}
              checked={on}
              onChange={() => pick(o.value)}
            />
            {o.label}
          </label>
        );
      })}
    </div>
  );
}

export { SegmentedControl, type SegmentedControlProps, type SegmentedOption };
