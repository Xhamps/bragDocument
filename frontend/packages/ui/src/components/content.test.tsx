import { render, screen } from "@testing-library/react";
import { MediaCell } from "#components/media-cell";
import { MediaCard } from "#components/media-card";
import { HeroHeader } from "#components/hero-header";
import { NotificationItem } from "#components/notification-item";

test("MediaCell draws initials without an image", () => {
  render(<MediaCell title="Eva Solain" />);
  expect(screen.getByText("ES")).toBeInTheDocument();
});

test("MediaCard with href makes the title a link", () => {
  render(<MediaCard title="Course" href="/c" />);
  expect(screen.getByRole("link", { name: "Course" })).toHaveAttribute(
    "href",
    "/c",
  );
});

test("HeroHeader uses the requested heading level", () => {
  render(<HeroHeader as="h2" lead="Power your" title="brag doc" />);
  expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent(
    "Power your brag doc",
  );
});

test("NotificationItem reads name + action + time", () => {
  render(
    <NotificationItem name="Eva" action="invited you" time="5m ago" unread />,
  );
  expect(screen.getByText("Eva")).toBeInTheDocument();
  expect(screen.getByText("5m ago")).toBeInTheDocument();
});
