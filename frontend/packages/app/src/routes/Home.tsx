import { Link } from "react-router";
import { Button, Tag } from "@bragdoc/ui";
import { DotWave } from "../home/DotWave";
import { Feature } from "../home/Feature";
import { DashboardPreview, LogPreview, SharePreview } from "../home/previews";

export function Component() {
  return (
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
      <div id="features">
        <Feature
          eyebrow="Capture"
          title="Log every win"
          preview={<LogPreview />}
        >
          Write it down while it's fresh, from the web or by messaging the
          Telegram bot. Each log keeps its impact, status, tags and links, so
          nothing is lost by review season.
        </Feature>
        <Feature
          eyebrow="Reflect"
          title="See your impact"
          preview={<DashboardPreview />}
          flip
        >
          A dashboard shows how often you log, which themes come up most and
          which parts of your role have gone quiet.
        </Feature>
        <Feature
          eyebrow="Share"
          title="Share and export"
          preview={<SharePreview />}
        >
          Invite your manager as a viewer or editor, and export a review-ready
          PDF when it's time to talk about your year.
        </Feature>
      </div>
      <section className="flex flex-col items-center gap-4 px-4 pb-24 text-center">
        <p className="type-title-2">Ready to start?</p>
        <Button asChild variant="primary" size="lg">
          <Link to="/sign-in">Sign in to start</Link>
        </Button>
      </section>
    </main>
  );
}
