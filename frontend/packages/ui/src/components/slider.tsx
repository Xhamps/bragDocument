import * as React from "react";
import { cn, focusRing } from "#lib/utils";

type SliderProps = {
  min?: number;
  max?: number;
  step?: number;
  value?: number;
  defaultValue?: number;
  onChange?: (value: number) => void;
  /** Formats the value bubble and aria-valuetext ("CA$ 6000"). */
  format?: (value: number) => string;
  showValue?: boolean;
  label?: React.ReactNode;
  /** Shows a "Clear" link in the header. */
  onClear?: () => void;
  disabled?: boolean;
  id?: string;
  className?: string;
};

// Native range input, as in the DS; the filled track is a gradient at --pct.
function Slider({
  min = 0,
  max = 100,
  step = 1,
  value,
  defaultValue,
  onChange,
  format = String,
  showValue = true,
  label,
  onClear,
  disabled,
  id,
  className,
}: SliderProps) {
  const [inner, setInner] = React.useState(defaultValue ?? min);
  const current = value ?? inner;
  const pct = max > min ? ((current - min) / (max - min)) * 100 : 0;
  const autoId = React.useId();
  const fid = id ?? autoId;

  return (
    <div
      data-slot="slider"
      className={cn(
        "flex min-w-[220px] flex-col gap-2",
        disabled && "pointer-events-none opacity-50",
        className,
      )}
    >
      {(label || onClear) && (
        <div className="flex items-baseline justify-between type-body text-fg-primary">
          {label ? <label htmlFor={fid}>{label}</label> : <span />}
          {onClear && (
            <button
              type="button"
              onClick={onClear}
              className={cn(
                "cursor-pointer rounded-sm border-0 bg-transparent p-0 type-body text-fg-primary hover:text-button-text",
                focusRing,
              )}
            >
              Clear
            </button>
          )}
        </div>
      )}
      <div
        className="group/slider relative pt-10"
        style={{ "--pct": `${pct}%` } as React.CSSProperties}
      >
        {showValue && (
          <span
            aria-hidden="true"
            className={cn(
              "absolute top-0 left-(--pct) -translate-x-1/2 rounded-pill bg-button px-3 py-[3px] text-[13px] leading-[18px] font-semibold whitespace-nowrap text-button-fg shadow-md",
              "transition-[translate,scale] duration-base ease-spring group-focus-within/slider:-translate-y-[3px] group-focus-within/slider:scale-105 group-hover/slider:-translate-y-[3px] group-hover/slider:scale-105",
              "after:absolute after:-bottom-[5px] after:left-1/2 after:-z-1 after:-ml-[5px] after:size-2.5 after:rotate-45 after:rounded-[2px] after:bg-button",
            )}
          >
            {format(current)}
          </span>
        )}
        <input
          id={fid}
          type="range"
          min={min}
          max={max}
          step={step}
          value={current}
          disabled={disabled}
          aria-valuetext={format(current)}
          onChange={(e) => {
            const v = Number(e.target.value);
            if (value === undefined) setInner(v);
            onChange?.(v);
          }}
          className={cn(
            "m-0 block h-2 w-full cursor-pointer appearance-none rounded-pill bg-[linear-gradient(to_right,var(--button)_var(--pct),var(--chart-muted)_var(--pct))] inset-shadow-ds",
            "[&::-webkit-slider-thumb]:size-[22px] [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-pill [&::-webkit-slider-thumb]:border-3 [&::-webkit-slider-thumb]:border-solid [&::-webkit-slider-thumb]:border-button-fg [&::-webkit-slider-thumb]:bg-button [&::-webkit-slider-thumb]:shadow-button",
            "[&::-webkit-slider-thumb]:transition-[scale,box-shadow] [&::-webkit-slider-thumb]:duration-base [&::-webkit-slider-thumb]:ease-spring hover:[&::-webkit-slider-thumb]:scale-115 hover:[&::-webkit-slider-thumb]:shadow-md active:[&::-webkit-slider-thumb]:scale-125 active:[&::-webkit-slider-thumb]:shadow-glow",
            "[&::-moz-range-thumb]:size-4 [&::-moz-range-thumb]:rounded-pill [&::-moz-range-thumb]:border-3 [&::-moz-range-thumb]:border-solid [&::-moz-range-thumb]:border-button-fg [&::-moz-range-thumb]:bg-button [&::-moz-range-thumb]:shadow-button",
            focusRing,
          )}
        />
      </div>
    </div>
  );
}

export { Slider, type SliderProps };
