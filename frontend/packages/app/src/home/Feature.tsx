import type { ReactNode } from "react";
import { cn } from "@bragdoc/ui";

type Props = {
  eyebrow: string;
  title: string;
  children: ReactNode;
  preview: ReactNode;
  flip?: boolean;
};

export function Feature({ eyebrow, title, children, preview, flip }: Props) {
  return (
    <section className="mx-auto grid max-w-5xl items-center gap-10 px-4 py-24 md:grid-cols-2">
      <div className={cn(flip && "md:order-2")}>
        <p className="type-caption text-button-text">{eyebrow}</p>
        <h2 className="mt-2 type-title-1">{title}</h2>
        <p className="mt-4 type-body text-fg-secondary">{children}</p>
      </div>
      {/* Decorative: a picture of the product, not part of the page's content. */}
      <div aria-hidden inert>
        {preview}
      </div>
    </section>
  );
}
