import type * as React from "react";
import { Tooltip as T } from "radix-ui";

type TooltipProps = {
  /** The trigger: exactly one focusable element. */
  children: React.ReactNode;
  content: React.ReactNode;
  placement?: "top" | "bottom" | "left" | "right";
  /** Keyboard shortcut shown as a key cap ("⌘K"). */
  shortcut?: string;
  /** Force it open (docs, onboarding); false behaves like unset. */
  open?: boolean;
};

// DS: inverts the theme (fg-primary fill, page text) so it reads on any backdrop.
function Tooltip({
  children,
  content,
  placement = "top",
  shortcut,
  open,
}: TooltipProps) {
  return (
    <T.Provider delayDuration={300}>
      <T.Root open={open || undefined}>
        <T.Trigger asChild>{children}</T.Trigger>
        <T.Portal>
          <T.Content
            data-slot="tooltip"
            side={placement}
            sideOffset={8}
            className="z-50 inline-flex animate-in items-center gap-2 rounded-sm bg-fg-primary px-3 py-1.5 type-footnote font-medium whitespace-nowrap text-page shadow-lg fade-in-0 data-[side=bottom]:slide-in-from-top-1 data-[side=left]:slide-in-from-right-1 data-[side=right]:slide-in-from-left-1 data-[side=top]:slide-in-from-bottom-1 data-[state=closed]:animate-out data-[state=closed]:fade-out-0"
          >
            {content}
            {shortcut && (
              <kbd className="rounded-[4px] border border-current px-[5px] font-sans text-[11px] leading-4 opacity-70">
                {shortcut}
              </kbd>
            )}
            <T.Arrow className="fill-fg-primary" />
          </T.Content>
        </T.Portal>
      </T.Root>
    </T.Provider>
  );
}

export { Tooltip, type TooltipProps };
