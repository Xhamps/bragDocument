import { Link } from "react-router";
import { Button, HeroHeader, Icon } from "@bragdoc/ui";
import { Feature } from "../home/Feature";
import { DashboardPreview, LogPreview, SharePreview } from "../home/previews";

export function Component() {
  return (
    <main>
      <section className="relative flex min-h-dvh flex-col items-center justify-center px-4">
        <HeroHeader
          title="Brag Document"
          subtitle="Log what you did and why it mattered, from the web or Telegram. Share it with your manager and export a review-ready PDF."
          actions={
            <Button asChild variant="gradient" size="lg" chevron>
              <Link to="/sign-in">Sign in</Link>
            </Button>
          }
        />
        <Icon
          name="chevronDown"
          className="absolute bottom-8 animate-bounce text-fg-tertiary motion-reduce:animate-none"
        />
      </section>
      <Feature eyebrow="Capture" title="Log every win" preview={<LogPreview />}>
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
        A dashboard shows how often you log, which themes come up most and which
        parts of your role have gone quiet.
      </Feature>
      <Feature
        eyebrow="Share"
        title="Share and export"
        preview={<SharePreview />}
      >
        Invite your manager as a viewer or editor, and export a review-ready PDF
        when it's time to talk about your year.
      </Feature>
      <section className="flex flex-col items-center gap-4 px-4 pb-24 text-center">
        <p className="type-title-2">Ready to start?</p>
        <Button asChild variant="primary" size="lg">
          <Link to="/sign-in">Sign in to start</Link>
        </Button>
      </section>
    </main>
  );
}
