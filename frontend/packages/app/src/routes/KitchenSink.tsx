import * as React from "react";
import {
  Button,
  ButtonGroup,
  Card,
  Checkbox,
  DataTable,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
  FieldTile,
  HeroHeader,
  Icon,
  IconButton,
  MediaCard,
  MediaCell,
  MenuList,
  NotificationItem,
  PageHeader,
  Pagination,
  RadioGroup,
  SearchField,
  SegmentedControl,
  Select,
  Slider,
  Stepper,
  Tag,
  TextArea,
  TextField,
  Toggle,
  Tooltip,
  TopBar,
  UserMenu,
  type ButtonProps,
  type CardProps,
  type DataTableColumn,
  type IconButtonProps,
  type MenuEntry,
  type TagProps,
  type TopBarItem,
} from "@bragdoc/ui";

// Data mirrors ds/preview/<Name>.html so the two can be compared side by side.

const BUTTONS: Array<ButtonProps & { text: string }> = [
  { text: "Learn more", chevron: true },
  { text: "Primary color", variant: "primary", chevron: true },
  { text: "Ghost color", variant: "ghost" },
  { text: "See all", variant: "tinted" },
  { text: "Danger", variant: "danger" },
  {
    text: "Primary inactive",
    variant: "primary",
    disabled: true,
    chevron: true,
  },
  { text: "Download", trailingIcon: "download" },
  { text: "Buy now", trailingIcon: "card" },
  { text: "Add card", trailingIcon: "plusCircle" },
  { text: "Email address", icon: "mail" },
  { text: "Start course", variant: "ghost", trailingIcon: "arrowRight" },
  { text: "Mark all as read", size: "sm" },
  { text: "Subscribe", pill: true },
  { text: "Start a free trial", pill: true, variant: "ghost" },
  { text: "Search flights", variant: "primary", size: "lg", glow: true },
  { text: "Get Started", variant: "gradient", size: "lg", chevron: true },
  { text: "Get Started", variant: "gradient", chevron: true },
  { text: "Ghost inactive", variant: "ghost", disabled: true },
];

const ICON_BUTTONS: IconButtonProps[] = [
  { icon: "moon", label: "Dark mode", pressed: true },
  { icon: "sun", label: "Light mode", pressed: false },
  { icon: "eye", label: "Show layer" },
  { icon: "bell", label: "Notifications", badge: 3 },
  { icon: "home", label: "Home", badge: true },
  { icon: "sliders", label: "Filters", variant: "tinted", shape: "square" },
  { icon: "close", label: "Close", size: "sm" },
  { icon: "play", label: "Play sm", size: "sm" },
  { icon: "play", label: "Play md", size: "md" },
  { icon: "play", label: "Play lg", size: "lg" },
  { icon: "play", label: "Play xl", size: "xl" },
  { icon: "play", label: "Play xl primary", size: "xl", variant: "primary" },
  {
    icon: "plus",
    label: "New item",
    shape: "diamond",
    size: "lg",
    variant: "primary",
    glow: true,
    tooltipPlacement: "right",
  },
  { icon: "share", label: "Share", shape: "square", variant: "ghost" },
  { icon: "download", label: "Download", shape: "square", shortcut: "⌘S" },
  { icon: "plus", label: "Add", variant: "primary", disabled: true },
];

const TAGS: Array<TagProps & { text: string }> = [
  { text: "Active", tone: "success" },
  { text: "In review", tone: "accent" },
  { text: "Overdue", tone: "danger" },
  { text: "Draft" },
  { text: "Design", tone: "purple" },
  { text: "Research", tone: "teal" },
  { text: "Trial term", variant: "outline", tone: "violet" },
  { text: "XP + 10", variant: "outline" },
  { text: "UI/UX Design", variant: "solid", tone: "accent" },
  { text: "Danger solid", variant: "solid", tone: "danger" },
];

const CARDS: Array<{
  tint: NonNullable<CardProps["tint"]>;
  title: string;
  body: string;
  action?: ButtonProps & { text: string };
}> = [
  {
    tint: "sheen",
    title: "UI Templates",
    body: "Introducing a collection of fully designed and functional components.",
    action: { text: "Browse" },
  },
  {
    tint: "accent",
    title: "Pro Plan",
    body: "Everything in Starter, plus AI summaries of your wins.",
    action: { text: "Upgrade", variant: "primary" },
  },
  {
    tint: "violet",
    title: "Mastering modular design systems",
    body: "Build scalable, cohesive systems for your team.",
    action: { text: "Start", trailingIcon: "arrowRight" },
  },
  {
    tint: "rose",
    title: "Limited offer",
    body: "Unlock full access to premium features at 50% off.",
    action: { text: "Claim" },
  },
  {
    tint: "cool",
    title: "Charging",
    body: "30 min remaining · 65% of the set charge limit.",
  },
  {
    tint: "none",
    title: "Confirmation",
    body: "Are you sure you want to change your profile information?",
    action: { text: "Cancel" },
  },
];

const ELEVATIONS = ["sm", "md", "lg", "xl"] as const;

const PIN =
  "M12 21s-7-6.2-7-11a7 7 0 0 1 14 0c0 4.8-7 11-7 11zM12 8a2 2 0 1 0 0 4 2 2 0 0 0 0-4z";
const CAL = "M4 6h16v14H4zM4 10h16M8 3v4M16 3v4";

const MENU: MenuEntry[] = [
  "Workspace",
  { value: "home", label: "Dashboard", icon: "home", href: "#dashboard" },
  { value: "docs", label: "Brag docs", icon: "book", badge: 3, href: "#docs" },
  {
    value: "components",
    label: "Components",
    icon: "grid",
    children: [
      { value: "all", label: "All components", href: "#all" },
      { value: "buttons", label: "Buttons", href: "#buttons" },
      { value: "menus", label: "Menus", href: "#menus" },
      { value: "cards", label: "Cards", href: "#cards" },
    ],
  },
  {
    value: "reports",
    label: "Reports",
    icon: "chart",
    children: [
      { value: "quarterly", label: "Quarterly review", href: "#q" },
      { value: "impact", label: "Impact summary", href: "#impact" },
    ],
  },
  "separator",
  { value: "settings", label: "Settings", icon: "settings", href: "#settings" },
  {
    value: "help",
    label: "Help center",
    icon: "help",
    href: "https://example.com/help",
    external: true,
  },
];

const INLINE_MENU: MenuEntry[] = [
  {
    value: "r",
    label: "Reports",
    icon: "chart",
    children: [
      { value: "quarterly", label: "Quarterly review" },
      { value: "impact", label: "Impact summary" },
    ],
  },
  { value: "s", label: "Settings", icon: "settings" },
];

const TOPBAR_ITEMS: TopBarItem[] = [
  { value: "flights", label: "Flights", href: "#" },
  { value: "stays", label: "Stays", href: "#" },
  { value: "cars", label: "Cars", href: "#" },
  {
    value: "packages",
    label: "Packages",
    children: [
      {
        value: "weekend",
        label: "Weekend getaways",
        description: "Two nights, flight and hotel",
        icon: "sun",
        href: "#",
      },
      {
        value: "family",
        label: "Family trips",
        description: "Rooms for four or more",
        icon: "home",
        href: "#",
      },
      {
        value: "business",
        label: "Business travel",
        description: "Flexible fares, lounge access",
        icon: "card",
        href: "#",
      },
    ],
  },
];

const NOTIFICATIONS = [
  {
    name: "Hola Spine",
    action: "prepared a report",
    time: "2m ago",
    unread: true,
  },
  {
    name: "Eva Solain",
    action: "invited you to a chat",
    time: "5m ago",
    unread: true,
  },
  { name: "Pierre Ford", action: "invited you to a meeting", time: "15m ago" },
];

const CELLS = [
  { title: "Meng To", subtitle: "Design Lead" },
  {
    title: "Riley Anderson",
    subtitle: "UI/UX Designer",
    image: "gradient-red-3",
    shape: "rounded",
  },
  {
    title: "Design and Prototype an App",
    subtitle: "Designcode",
    meta: "14k views · 1 month ago",
    image: "gradient-blue-1",
    badge: "4:30",
    shape: "thumb",
  },
  { title: "Eva Solain", subtitle: "Staff Engineer", size: "lg" },
] as const;

type Person = {
  id: number;
  name: string;
  role: string;
  team: string[];
  status: string;
  wins: number;
  updated: string;
};

const PEOPLE: Person[] = [
  {
    id: 1,
    name: "Hola Spine",
    role: "Product Designer",
    team: ["Design"],
    status: "Active",
    wins: 14,
    updated: "2026-10-08",
  },
  {
    id: 2,
    name: "Eva Solain",
    role: "Staff Engineer",
    team: ["Platform", "AI"],
    status: "In review",
    wins: 22,
    updated: "2026-10-06",
  },
  {
    id: 3,
    name: "Pierre Ford",
    role: "Eng Manager",
    team: ["Platform"],
    status: "Active",
    wins: 9,
    updated: "2026-09-30",
  },
  {
    id: 4,
    name: "Steve Ater",
    role: "Data Scientist",
    team: ["Research"],
    status: "Overdue",
    wins: 5,
    updated: "2026-09-12",
  },
  {
    id: 5,
    name: "Meng To",
    role: "Design Lead",
    team: ["Design", "Research"],
    status: "Draft",
    wins: 31,
    updated: "2026-10-09",
  },
];

const TEAM_TONE: Record<string, TagProps["tone"]> = {
  Design: "purple",
  Platform: "accent",
  AI: "violet",
  Research: "teal",
};
const STATUS_TONE: Record<string, TagProps["tone"]> = {
  Active: "success",
  "In review": "accent",
  Overdue: "danger",
  Draft: "neutral",
};

const PEOPLE_COLUMNS: DataTableColumn<Person>[] = [
  { key: "name", header: "Name", sortable: true, primary: true },
  { key: "role", header: "Role", sortable: true },
  {
    key: "team",
    header: "Team",
    sortable: true,
    tags: (t) => ({ tone: TEAM_TONE[t] }),
    sortValue: (r) => r.team[0] ?? "",
  },
  {
    key: "status",
    header: "Status",
    sortable: true,
    tags: (t) => ({ tone: STATUS_TONE[t] }),
  },
  { key: "wins", header: "Wins", sortable: true, align: "right" },
  {
    key: "updated",
    header: "Updated",
    sortable: true,
    render: (r) =>
      new Date(`${r.updated}T12:00:00`).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
      }),
  },
];

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-4">
      <h2 className="type-title-2">{title}</h2>
      {children}
    </section>
  );
}

const Row = ({ children }: { children: React.ReactNode }) => (
  <div className="flex flex-wrap items-center gap-3">{children}</div>
);

export function Component() {
  const [radius, setRadius] = React.useState("10");
  const [trip, setTrip] = React.useState("Roundtrip");
  const [lock, setLock] = React.useState(true);
  const [budget, setBudget] = React.useState(6000);
  const [bags, setBags] = React.useState(1);
  const [range, setRange] = React.useState("Week");
  const [menu, setMenu] = React.useState("buttons");
  const [page, setPage] = React.useState(1);
  const [compact, setCompact] = React.useState(false);

  return (
    <div className="flex flex-col gap-12">
      <Section title="Buttons">
        <Row>
          {BUTTONS.map(({ text, ...p }, i) => (
            <Button key={i} {...p}>
              {text}
            </Button>
          ))}
        </Row>
        <Row>
          {ICON_BUTTONS.map((p) => (
            <IconButton key={p.label} {...p} />
          ))}
        </Row>
        <Row>
          <ButtonGroup label="Pager" size="sm">
            <IconButton icon="arrowLeft" label="Previous" disabled />
            <IconButton icon="arrowRight" label="Next" />
          </ButtonGroup>
          <ButtonGroup label="Channel">
            <Button>Join</Button>
            <Button
              icon="bell"
              trailingIcon={<Icon name="chevronDown" size={16} />}
            >
              Subscribed
            </Button>
          </ButtonGroup>
          <ButtonGroup label="Channel primary">
            <Button>Join</Button>
            <Button variant="primary" icon="bell">
              Subscribed
            </Button>
          </ButtonGroup>
          <ButtonGroup label="Copy">
            <Button icon="download">Export</Button>
            <IconButton icon="chevronDown" label="More export options" />
          </ButtonGroup>
        </Row>
        <Row>
          <Tooltip content="Search" shortcut="⌘K">
            <IconButton icon="search" label="Search" tooltip={false} />
          </Tooltip>
          <Tooltip content="Copied to clipboard" placement="bottom">
            <Button>Copy link</Button>
          </Tooltip>
          <Tooltip content="Next card" placement="right">
            <IconButton icon="arrowRight" label="Next card" tooltip={false} />
          </Tooltip>
          <Tooltip content="Hover me" placement="left">
            <Button variant="ghost">Hover for tooltip</Button>
          </Tooltip>
        </Row>
      </Section>

      <Section title="Forms">
        <div className="grid gap-6 sm:grid-cols-3">
          <TextField
            label="Email"
            type="email"
            icon="mail"
            placeholder="Email address"
            hint="We send the receipt here."
          />
          <TextField
            label="Card number"
            defaultValue="4242 42"
            error="Enter all 16 digits."
          />
          <TextField label="Promo code" placeholder="Optional" disabled />
        </div>
        <TextArea
          label="Reply to comment"
          placeholder="Write a reply"
          hint="Markdown is supported."
        />
        <Row>
          <Select
            aria-label="Corner radius"
            prefix="Corner Radius:"
            options={["6", "10", "16", "24"]}
            value={radius}
            onChange={(e) => setRadius(e.target.value)}
          />
          <Select
            aria-label="Sort by"
            placeholder="Sort by"
            defaultValue=""
            options={["Price", "Duration", "Departure"]}
          />
          <div className="w-56">
            <Select
              label="Currency"
              options={["USD", "CAD", "BRL"]}
              defaultValue="USD"
            />
          </div>
        </Row>
        <div className="grid gap-6 sm:grid-cols-2">
          <div className="flex flex-col gap-4">
            <Checkbox label="Seat choice included" defaultChecked />
            <Checkbox label="No cancel fee" />
            <Checkbox
              label="No change fee"
              description="Change dates up to 24h before departure."
            />
            <Checkbox label="Lounge access" disabled />
          </div>
          <div className="flex flex-col gap-6">
            <RadioGroup
              label="Trip type"
              direction="row"
              options={["Roundtrip", "One way", "Multi-City"]}
              value={trip}
              onChange={setTrip}
            />
            <RadioGroup
              label="Stops"
              options={[
                "Any number of stops",
                "Nonstop only",
                "1 stop or fewer",
                "2 stops or fewer",
              ]}
              defaultValue="Nonstop only"
            />
          </div>
        </div>
        <Row>
          <Toggle label="Lock" checked={lock} onChange={setLock} />
          <Toggle label="Select all airlines" />
          <Toggle label="Disabled" disabled />
        </Row>
        <div className="grid items-center gap-6 sm:grid-cols-2">
          <Slider
            label={`Up to CA$${budget}`}
            min={0}
            max={10000}
            step={100}
            value={budget}
            onChange={setBudget}
            format={(v) => `CA$ ${v}`}
            onClear={() => setBudget(0)}
          />
          <SearchField />
        </div>
        <Row>
          <span className="text-fg-secondary">Carry-on bag included</span>
          <Stepper
            label="Carry-on bags"
            value={bags}
            onChange={setBags}
            max={3}
          />
          <Stepper label="Travellers" defaultValue={0} />
        </Row>
        <Card>
          <div className="grid gap-4 sm:grid-cols-2">
            <FieldTile label="From" value="Montreal, Canada" icon={PIN} />
            <FieldTile label="To" value="Tokyo, Japan" icon={PIN} />
            <FieldTile label="Depart" value="Dec 15, 2023" icon={CAL} />
            <FieldTile label="Return" placeholder="Add date" icon={CAL} />
          </div>
        </Card>
        <Row>
          <SegmentedControl
            label="Style"
            options={["Glass", "Outline", "Flat"]}
          />
          <SegmentedControl
            label="Range"
            tone="accent"
            options={["Day", "Week", "Month"]}
            value={range}
            onChange={setRange}
          />
          <SegmentedControl
            label="Size"
            size="sm"
            options={["sm", "md", "lg"]}
          />
        </Row>
      </Section>

      <Section title="Tags">
        <Row>
          {TAGS.map(({ text, ...p }) => (
            <Tag key={text} {...p}>
              {text}
            </Tag>
          ))}
        </Row>
      </Section>

      <Section title="Cards">
        <div className="grid gap-4 sm:grid-cols-3">
          {CARDS.map(({ tint, title, body, action }) => (
            <Card key={tint} tint={tint} title={title} description={body}>
              <div className="mt-4 flex items-center justify-between">
                <Tag variant="outline">tint: {tint}</Tag>
                {action && (
                  <Button size="sm" {...action}>
                    {action.text}
                  </Button>
                )}
              </div>
            </Card>
          ))}
        </div>
        <div className="grid gap-4 sm:grid-cols-4">
          {ELEVATIONS.map((e, i) => (
            <Card
              key={e}
              tint="none"
              compact
              elevation={e}
              animate
              delay={i * 60}
              title={`elevation ${e}`}
            />
          ))}
          <Card
            interactive
            tint="accent"
            title="Interactive card"
            description="Lifts on hover with a sheen sweep."
            aria-label="Interactive card"
            onClick={() => {}}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-3">
          <MediaCard
            image="gradient-learn"
            imageAlt="Violet gradient"
            tag={
              <Tag variant="solid" tone="accent">
                UI/UX Design
              </Tag>
            }
            title="Designing a Travel App"
            description="Learn how to design a captivating travel app from concept to user-centric experience."
            progress={42}
            author={{ name: "Riley Anderson", role: "UI/UX Designer" }}
            href="#"
            animate
          />
          <MediaCard
            image="gradient-red-3"
            badge="12:30"
            eyebrow="Spline Academy · 9.2k views"
            title="Create 3D Site with Spline and React"
            tint="rose"
            mediaAction={
              <IconButton
                icon="play"
                label="Play preview"
                size="sm"
                tooltip={false}
              />
            }
            actions={
              <>
                <Button size="sm" variant="primary" icon="play">
                  Watch
                </Button>
                <Button size="sm">Save</Button>
              </>
            }
            animate
            delay={60}
          />
          <MediaCard
            layout="overlay"
            image="gradient-blue-1"
            imageAlt="Deep blue gradient"
            tag={
              <Tag variant="outline" tone="violet">
                Trial term
              </Tag>
            }
            title="Mastering Modular Design Systems"
            description="Build scalable, cohesive systems that streamline UI development."
            actions={
              <Button size="sm" trailingIcon="arrowRight">
                Start course
              </Button>
            }
            onClick={() => {}}
            animate
            delay={120}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <MediaCard
            layout="horizontal"
            image="gradient-light-blue-1"
            title="Lesson 4 · Formatting Text"
            eyebrow="Web Development"
            description="Headings, emphasis and lists in HTML."
            progress={80}
            tint="accent"
            href="#"
          />
          <MediaCard
            layout="horizontal"
            image="gradient-yellow-1"
            title="Smart cards"
            eyebrow="All your cards in one place"
            description="Make your finances work for you with the power of AI."
            actions={
              <Button size="sm" trailingIcon="plusCircle">
                Add card
              </Button>
            }
          />
        </div>
      </Section>

      <Section title="Navigation">
        <div className="flex flex-wrap items-start gap-6">
          <div className="w-64">
            <MenuList
              label="Main"
              items={MENU}
              value={menu}
              onSelect={setMenu}
            />
          </div>
          <div className="w-60">
            <MenuList
              label="Inline groups"
              submenu="inline"
              defaultValue="quarterly"
              items={INLINE_MENU}
            />
          </div>
        </div>
        <TopBar
          brand={{ name: "BragDoc", href: "#" }}
          items={TOPBAR_ITEMS}
          defaultValue="flights"
          search={{ placeholder: "Search", "aria-label": "Search trips" }}
          actions={
            <>
              <IconButton
                icon="bell"
                label="Notifications"
                badge
                variant="tinted"
                tooltipPlacement="bottom"
              />
              <IconButton
                icon="moon"
                label="Dark mode"
                variant="tinted"
                tooltipPlacement="bottom"
              />
            </>
          }
          user={{
            user: {
              name: "Riley Anderson",
              email: "riley@bragdoc.app",
              status: "online",
            },
          }}
        />
        <TopBar
          variant="floating"
          brand={{ name: "BragDoc UI", href: "#" }}
          defaultValue="components"
          items={[
            { value: "components", label: "Components", href: "#" },
            { value: "pricing", label: "Pricing", href: "#" },
            { value: "changelog", label: "Changelog", href: "#" },
          ]}
          actions={<Button variant="tinted">Log in</Button>}
          cta={
            <Button pill variant="primary">
              Buy now
            </Button>
          }
        />
        <Row>
          <UserMenu
            placement="bottom-start"
            user={{
              name: "Riley Anderson",
              role: "UI/UX Designer",
              email: "riley@bragdoc.app",
              status: "online",
            }}
          />
          <UserMenu
            user={{
              name: "Hola Spine",
              role: "Product Designer",
              email: "hola@bragdoc.app",
            }}
            items={[
              { label: "Profile", icon: "user" },
              { label: "Settings", icon: "settings", shortcut: "⌘," },
              { label: "Billing", icon: "card" },
              "separator",
              { label: "Sign out", icon: "signOut", tone: "danger" },
            ]}
          />
          <UserMenu
            variant="avatar"
            user={{
              name: "Eva Solain",
              email: "eva@bragdoc.app",
              status: "busy",
            }}
          />
        </Row>
        <PageHeader
          as="h2"
          breadcrumbs={[
            { label: "Workspace", href: "#" },
            { label: "Courses", href: "#" },
            { label: "Web development" },
          ]}
          eyebrow={
            <>
              <Tag variant="outline" tone="violet">
                Trial term
              </Tag>
              <Tag tone="accent">Level 6</Tag>
            </>
          }
          title="Web development"
          gradient="accent"
          subtitle="Getting started with HTML: the basic web technology, HTML code, headers and formatting text."
          meta={
            <>
              <span>12 lessons</span>
              <span>·</span>
              <span>Updated Oct 9</span>
              <span>·</span>
              <span>34% complete</span>
            </>
          }
          actions={
            <>
              <Button>Preview</Button>
              <Button variant="primary" chevron>
                Continue
              </Button>
            </>
          }
          tabs={
            <SegmentedControl
              label="Section"
              tone="accent"
              options={["Lessons", "Resources", "Discussion"]}
            />
          }
        />
      </Section>

      <Section title="Content">
        <HeroHeader
          as="h2"
          eyebrow="Announcing Series B"
          onEyebrowClick={() => {}}
          lead="Power your"
          title="brag doc with AI"
          gradient="accent"
          subtitle="Capture every win as it happens, and turn a year of work into a review-ready story in one click."
          actions={
            <>
              <Button chevron>Learn more</Button>
              <Button variant="primary">Start for free</Button>
            </>
          }
        />
        <div className="grid gap-4 sm:grid-cols-2">
          <Card>
            <HeroHeader
              as="h3"
              size="title-1"
              align="left"
              gradient="primary"
              title="Customize everything."
              subtitle="Layouts, styles, patterns, breakpoints, icons."
            />
          </Card>
          <Card>
            <HeroHeader
              as="h3"
              size="title-1"
              align="left"
              gradient="secondary"
              title="Mastering modular design systems"
              eyebrow={
                <Tag variant="outline" tone="violet">
                  Trial term
                </Tag>
              }
            />
          </Card>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Card compact title="Notifications">
            <div className="mt-3 flex flex-col gap-2">
              {NOTIFICATIONS.map((n, i) => (
                <NotificationItem key={n.name} {...n} animate delay={i * 60} />
              ))}
            </div>
          </Card>
          <Card compact title="Media cells">
            <div className="mt-3 flex flex-col gap-4">
              {CELLS.map((c) => (
                <MediaCell key={c.title} {...c} />
              ))}
            </div>
          </Card>
        </div>
      </Section>

      <Section title="DataTable & Pagination">
        <Toggle
          label="Compact density"
          checked={compact}
          onChange={setCompact}
        />
        <DataTable
          title="Brag documents"
          caption="Brag documents by person"
          rowLabel={(r) => r.name}
          density={compact ? "compact" : "comfortable"}
          toolbar={
            <>
              <Button size="sm" icon="download">
                Export
              </Button>
              <Button size="sm" variant="primary" icon="plus">
                New entry
              </Button>
            </>
          }
          selectable
          defaultSelected={[2]}
          defaultSort={{ key: "updated", dir: "desc" }}
          bulkActions={(_keys, clear) => (
            <>
              <Button size="sm" icon="download">
                Export
              </Button>
              <Button size="sm" icon="trash" onClick={clear}>
                Delete
              </Button>
            </>
          )}
          columns={PEOPLE_COLUMNS}
          rows={PEOPLE}
          rowActions={(r) => (
            <>
              <IconButton
                icon="edit"
                label={`Edit ${r.name}`}
                variant="tinted"
                size="sm"
                tooltip="Edit"
              />
              <IconButton
                icon="trash"
                label={`Delete ${r.name}`}
                variant="tinted"
                size="sm"
                tooltip="Delete"
              />
            </>
          )}
          footer={
            <Pagination
              pageCount={10}
              page={page}
              onChange={setPage}
              total={48}
              pageSize={5}
            />
          }
        />
        <DataTable
          title="Empty"
          caption="Empty table"
          columns={PEOPLE_COLUMNS}
          rows={[]}
          empty="No brag documents yet."
        />
        <Pagination pageCount={25} defaultPage={12} total={248} pageSize={10} />
        <Pagination
          pageCount={6}
          defaultPage={2}
          pageSize={10}
          pageSizeOptions={[10, 25, 50]}
        />
        <Pagination pageCount={4} defaultPage={1} variant="simple" />
      </Section>

      <Section title="Dialog & DropdownMenu">
        <Row>
          <Dialog>
            <DialogTrigger asChild>
              <Button>Open dialog</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Dialog title</DialogTitle>
                <DialogDescription>Dialog description.</DialogDescription>
              </DialogHeader>
              <TextField label="Name" placeholder="Your name" />
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="ghost">Cancel</Button>
                </DialogClose>
                <DialogClose asChild>
                  <Button variant="primary">Save</Button>
                </DialogClose>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button chevron>Actions</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuLabel>Document</DropdownMenuLabel>
              <DropdownMenuItem>
                Rename
                <DropdownMenuShortcut>⌘R</DropdownMenuShortcut>
              </DropdownMenuItem>
              <DropdownMenuItem>Archive</DropdownMenuItem>
              <DropdownMenuCheckboxItem
                checked={compact}
                onCheckedChange={(v) => setCompact(v === true)}
              >
                Compact table
              </DropdownMenuCheckboxItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="danger">Delete</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </Row>
      </Section>
    </div>
  );
}
