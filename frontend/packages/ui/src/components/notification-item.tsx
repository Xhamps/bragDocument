import type * as React from "react";
import { cn } from "#lib/utils";
import { avatarClass, initials } from "#components/media-cell";

type NotificationItemProps = Omit<React.ComponentProps<"div">, "children"> & {
  name: string;
  action: React.ReactNode;
  time: string;
  /** Photo URL; initials are drawn when absent. */
  avatar?: string;
  unread?: boolean;
  /** Fade-and-rise entrance; stagger rows with delay in ms. */
  animate?: boolean;
  delay?: number;
};

function NotificationItem({
  name,
  action,
  time,
  avatar,
  unread,
  animate,
  delay,
  className,
  style,
  ...props
}: NotificationItemProps) {
  return (
    <div
      data-slot="notification-item"
      style={
        animate && delay
          ? ({ "--delay": `${delay}ms`, ...style } as React.CSSProperties)
          : style
      }
      {...props}
      className={cn(
        "flex items-start gap-3 rounded-md glass p-3 shadow-glass transition-[translate,box-shadow,background-color] duration-base ease-standard hover:translate-x-0.5 hover:shadow-md motion-reduce:hover:translate-x-0",
        animate && "animate-enter",
        className,
      )}
    >
      {avatar ? (
        <img className={avatarClass} src={avatar} alt="" />
      ) : (
        <span className={avatarClass} aria-hidden="true">
          {initials(name)}
        </span>
      )}
      <div className="min-w-0 flex-1">
        <div className="type-callout text-fg-primary">{name}</div>
        <div className="type-callout font-normal text-fg-secondary">
          {action}
        </div>
      </div>
      <div className="type-footnote leading-5 whitespace-nowrap text-fg-secondary">
        {time}
      </div>
      {unread && (
        <span
          role="img"
          aria-label="Unread"
          className="mt-1.5 size-2 flex-none animate-pulse-dot rounded-pill bg-button-text"
        />
      )}
    </div>
  );
}

export { NotificationItem, type NotificationItemProps };
