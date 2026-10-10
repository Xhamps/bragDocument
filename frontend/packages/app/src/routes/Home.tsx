import { Link } from "react-router";
import type { ReactNode } from "react";
import { Button, Card, cn, Icon, Tag, TopBar } from "@bragdoc/ui";
import { DotWave } from "../home/DotWave";
import {
  AuditPreview,
  DashboardPreview,
  LogPreview,
  SharePreview,
  TelegramPreview,
} from "../home/previews";

// Faint grid lines behind each preview panel.
const gridLine =
  "color-mix(in srgb, var(--container-divider) 35%, transparent) 0 1px, transparent 1px 24px";

function BentoCard({
  icon,
  title,
  text,
  className,
  children,
}: {
  icon: string;
  title: string;
  text: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <Card
      title={
        <span className="flex items-center gap-2">
          <Icon name={icon} className="text-button-text" />
          {title}
        </span>
      }
      description={text}
      className={cn("flex flex-col", className)}
    >
      {/* Decorative: a picture of the product, not part of the page's content. */}
      <div
        aria-hidden
        inert
        className="mt-6 flex-1 rounded-md border border-divider p-4"
        style={{
          backgroundImage: `repeating-linear-gradient(0deg, ${gridLine}), repeating-linear-gradient(90deg, ${gridLine})`,
        }}
      >
        {children}
      </div>
    </Card>
  );
}

export function Component() {
  return (
    <>
      <TopBar
        variant="floating"
        sticky
        className="fixed inset-x-0 top-0"
        brand={{ name: "Brag Document", href: "/" }}
        renderLink={(_item, { href, ...props }) => (
          <Link to={href!} {...props} />
        )}
        cta={
          <Button asChild variant="primary" size="sm" chevron>
            <Link to="/sign-in">Sign in</Link>
          </Button>
        }
      />
      <main>
        <section className="relative isolate flex min-h-dvh items-center overflow-hidden px-4 py-24">
          <DotWave className="pointer-events-none absolute inset-0 -z-10 size-full" />
          <div className="mx-auto grid w-full max-w-6xl items-center gap-12 md:grid-cols-2">
            <div className="flex flex-col items-start gap-6">
              <Tag tone="accent">Your year, in writing</Tag>
              <h1 className="type-display">
                <span className="block">You did the work.</span>{" "}
                <span className="block text-gradient-accent">
                  We keep the receipts.
                </span>
              </h1>
              <p className="max-w-md type-body text-fg-secondary">
                Log what you did and why it mattered, from the web or Telegram.
                Share it with your manager and export a review-ready PDF.
              </p>
              <div className="flex flex-wrap gap-3">
                <Button asChild variant="gradient" size="lg" chevron>
                  <Link to="/sign-in">Sign in</Link>
                </Button>
                <Button asChild size="lg">
                  <a href="#features">See how it works</a>
                </Button>
              </div>
            </div>
            {/* Decorative: a picture of the product, not part of the page's content. */}
            <div aria-hidden inert className="rounded-xl glass shadow-glass">
              <div className="flex items-center gap-2 border-b border-divider px-4 py-3">
                <span className="size-3 rounded-full bg-fg-tertiary/40" />
                <span className="size-3 rounded-full bg-fg-tertiary/40" />
                <span className="size-3 rounded-full bg-fg-tertiary/40" />
                <span className="ml-3 type-footnote text-fg-tertiary">
                  brag document · job 1
                </span>
              </div>
              <div className="p-6">
                <LogPreview />
              </div>
            </div>
          </div>
        </section>
        <section id="features" className="mx-auto max-w-6xl px-4 py-24">
          <div className="mx-auto flex max-w-2xl flex-col items-center gap-3 text-center">
            <p className="type-caption text-button-text">
              Everything review season needs
            </p>
            <h2 className="type-title-1">
              From a quick note to a review-ready story
            </h2>
            <p className="type-body text-fg-secondary">
              Capture wins as they happen, see the shape of your year, and hand
              your manager something worth reading.
            </p>
          </div>
          <div className="mt-12 grid gap-4 md:grid-cols-3">
            <BentoCard
              icon="edit"
              title="Log every win"
              text="Write it down while it's fresh, from the web or Telegram. Each log keeps its impact, status, tags and links."
              className="md:col-span-2"
            >
              <LogPreview />
            </BentoCard>
            <BentoCard
              icon="chart"
              title="See your impact"
              text="How often you log, which themes come up most, and what's gone quiet."
              className="md:row-span-3"
            >
              <DashboardPreview />
            </BentoCard>
            <BentoCard
              icon="share"
              title="Share and export"
              text="Invite your manager as a viewer or editor, and export a review-ready PDF."
              className="md:col-span-2"
            >
              <SharePreview />
            </BentoCard>
            <BentoCard
              icon="send"
              title="Telegram bot"
              text="Message the bot and it lands in your document."
            >
              <TelegramPreview />
            </BentoCard>
            <BentoCard
              icon="history"
              title="Audit trail"
              text="Every edit and share is recorded."
            >
              <AuditPreview />
            </BentoCard>
          </div>
        </section>
        <section className="flex flex-col items-center gap-4 px-4 pb-24 text-center">
          <p className="type-title-2">Ready to start?</p>
          <Button asChild variant="primary" size="lg">
            <Link to="/sign-in">Sign in to start</Link>
          </Button>
        </section>
      </main>
    </>
  );
}
