import { Link } from "react-router";
import { Button, HeroHeader, Icon } from "@bragdoc/ui";

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
    </main>
  );
}
