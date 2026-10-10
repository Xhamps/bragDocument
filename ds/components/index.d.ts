import type { ReactNode, ButtonHTMLAttributes, InputHTMLAttributes, TextareaHTMLAttributes, SelectHTMLAttributes } from "react";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /** glass = translucent default; primary = Color button on `button`; ghost = outlined in `button`; tinted = text-only in `button`; gradient = the single Get Started CTA (`gradient-red-3`). */
  variant?: "glass" | "primary" | "ghost" | "tinted" | "gradient";
  size?: "sm" | "md" | "lg";
  /** Trailing chevron: the action navigates. */
  chevron?: boolean;
  /** Adds `shadow-glow`. At most one per view. */
  glow?: boolean;
  /** Leading 16px icon: a node, or a built-in name ("download", "card", "plusCircle", "arrowRight"…). */
  icon?: ReactNode | string;
  /** Trailing 16px icon, same forms as `icon`. */
  trailingIcon?: ReactNode | string;
  /** Fully rounded corners (radius-pill). */
  pill?: boolean;
  /** Stretches to its container's width. */
  fullWidth?: boolean;
  children?: ReactNode;
}
export interface SegmentedOption { value: string; label: ReactNode }
export interface SegmentedControlProps {
  options: Array<string | SegmentedOption>;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
  /** neutral = raised glass pill (Glass/Outline/Flat); accent = selected label in `button` (Day/Week/Month). */
  tone?: "neutral" | "accent";
  size?: "sm" | "md";
  /** Accessible name of the group. */
  label?: string;
}
export interface SearchFieldProps extends InputHTMLAttributes<HTMLInputElement> { label?: string }
export interface CardProps {
  /** Transparent gradient over the glass: sheen (default, white light catch), accent (blue → violet), violet (corner glow), rose (warm base), cool (teal → blue), none (flat glass). */
  tint?: "sheen" | "accent" | "violet" | "rose" | "cool" | "none";
  /** Hover lift (translateY -4px, shadow-lg) and a sheen sweep; set automatically when onClick is given. */
  interactive?: boolean;
  onClick?: () => void;
  /** Override the resting shadow with a step of the elevation scale. */
  elevation?: "sm" | "md" | "lg" | "xl";
  /** Fade-and-rise entrance (bd-enter); stagger with delay in ms. */
  animate?: boolean;
  delay?: number;
  title?: ReactNode;
  description?: ReactNode;
  compact?: boolean;
  as?: string;
  style?: object;
  children?: ReactNode;
}
export interface NotificationItemProps {
  name: string;
  action: ReactNode;
  time: string;
  /** Photo URL; initials are drawn when absent. */
  avatar?: string;
  unread?: boolean;
  /** Fade-and-rise entrance; stagger rows with delay in ms. */
  animate?: boolean;
  delay?: number;
}
export interface MenuItem {
  label: string;
  value?: string;
  /** A node, or a built-in icon name ("grid", "book", "chart", "settings", "help", "user", "card"…). */
  icon?: ReactNode | string;
  trailing?: ReactNode;
  /** Count pill on the right, e.g. unread items. */
  badge?: ReactNode;
  /** Renders the row as a link; selecting still updates the highlight. */
  href?: string;
  /** Opens in a new tab and shows an external-link icon. */
  external?: boolean;
  /** Nested items: the row becomes an expandable group with a rotating caret. */
  children?: Array<MenuItem | "separator" | string>;
  /** Start this group expanded (groups holding the current item open automatically). */
  defaultOpen?: boolean;
  onSelect?: (value: string) => void;
}
export interface MenuListProps {
  /** Rows, "separator", or a plain string for an uppercase section heading. */
  items: Array<MenuItem | "separator" | string>;
  value?: string;
  defaultValue?: string;
  onSelect?: (value: string, item: MenuItem) => void;
  /** false drops the glass container (inside a sidebar that is already glass). */
  glass?: boolean;
  /** How groups with children open: flyout = a tooltip-style panel beside the row, on hover, focus, click or ArrowRight (default); inline = an accordion inside the list. */
  submenu?: "flyout" | "inline";
  label?: string;
}
export interface ToggleProps {
  checked?: boolean;
  defaultChecked?: boolean;
  onChange?: (checked: boolean) => void;
  label?: ReactNode;
  disabled?: boolean;
}
export declare function Button(props: ButtonProps): JSX.Element;
export declare function SegmentedControl(props: SegmentedControlProps): JSX.Element;
export declare function SearchField(props: SearchFieldProps): JSX.Element;
export declare function Card(props: CardProps): JSX.Element;
export declare function NotificationItem(props: NotificationItemProps): JSX.Element;
export declare function MenuList(props: MenuListProps): JSX.Element;
export declare function Toggle(props: ToggleProps): JSX.Element;

interface FieldFrame {
  /** Visible label above the control. */
  label?: ReactNode;
  /** Helper line under the control. */
  hint?: ReactNode;
  /** Error message; replaces the hint, sets the `danger` border and aria-invalid. */
  error?: ReactNode;
}
export interface TextFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "children">, FieldFrame {
  /** Leading 20px icon: a node, or a built-in name ("mail", "search"). */
  icon?: ReactNode | string;
  /** Trailing node, e.g. a unit or an icon button. */
  trailing?: ReactNode;
}
export interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement>, FieldFrame {}
export interface SelectOption { value: string; label: ReactNode }
export interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, "children" | "prefix">, FieldFrame {
  options: Array<string | SelectOption>;
  /** Inline lead-in inside the pill ("Corner Radius:"). */
  prefix?: ReactNode;
  placeholder?: string;
}
export interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type"> {
  label?: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
}
export interface RadioOption { value: string; label: ReactNode; disabled?: boolean }
export interface RadioGroupProps {
  options: Array<string | RadioOption>;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
  /** column (filters) or row (Roundtrip / One way / Multi-City). */
  direction?: "column" | "row";
  name?: string;
  label?: string;
  disabled?: boolean;
}
export interface SliderProps {
  min?: number; max?: number; step?: number;
  value?: number; defaultValue?: number;
  onChange?: (value: number) => void;
  /** Formats the value bubble and aria-valuetext ("CA$ 6000"). */
  format?: (value: number) => string;
  showValue?: boolean;
  label?: ReactNode;
  /** Shows a "Clear" link in the header. */
  onClear?: () => void;
  disabled?: boolean;
}
export interface StepperProps {
  min?: number; max?: number;
  value?: number; defaultValue?: number;
  onChange?: (value: number) => void;
  /** Accessible name of what is counted ("Carry-on bags"). */
  label?: string;
  disabled?: boolean;
}
export interface FieldTileProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: ReactNode;
  value?: ReactNode;
  placeholder?: ReactNode;
  /** 24px icon: a node or a built-in name. */
  icon?: ReactNode | string;
}
export declare function TextField(props: TextFieldProps): JSX.Element;
export declare function TextArea(props: TextAreaProps): JSX.Element;
export declare function Select(props: SelectProps): JSX.Element;
export declare function Checkbox(props: CheckboxProps): JSX.Element;
export declare function RadioGroup(props: RadioGroupProps): JSX.Element;
export declare function Slider(props: SliderProps): JSX.Element;
export declare function Stepper(props: StepperProps): JSX.Element;
export declare function FieldTile(props: FieldTileProps): JSX.Element;

export interface TooltipProps {
  /** The trigger: exactly one focusable element. */
  children: ReactNode;
  content: ReactNode;
  placement?: "top" | "bottom" | "left" | "right";
  /** Keyboard shortcut shown as a key cap ("⌘K"). */
  shortcut?: string;
  /** Force it open (docs, onboarding). */
  open?: boolean;
}
export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /** A node, or a built-in name: arrowLeft, arrowRight, plus, minus, close, download, card, moon, sun, play, home, bell, sliders, eye, share, search, chevron, check, mail. */
  icon: ReactNode | string;
  /** Required accessible name; also the default tooltip. */
  label: string;
  variant?: "glass" | "primary" | "ghost" | "tinted" | "gradient";
  size?: "sm" | "md" | "lg" | "xl";
  /** circle (default), square (radius-md), or diamond (the glowing + button). */
  shape?: "circle" | "square" | "diamond";
  /** Tooltip text; false hides it. Defaults to `label`. */
  tooltip?: ReactNode | false;
  tooltipPlacement?: "top" | "bottom" | "left" | "right";
  shortcut?: string;
  glow?: boolean;
  /** Toggle state (moon / sun, eye): sets aria-pressed and the selected ring. */
  pressed?: boolean;
  /** true = dot; a number or short text = count. */
  badge?: boolean | number | string;
}
export interface ButtonGroupProps {
  /** Buttons or IconButtons joined into one pill. */
  children: ReactNode;
  label?: string;
  size?: "sm" | "md";
}
export declare function Tooltip(props: TooltipProps): JSX.Element;
export declare function IconButton(props: IconButtonProps): JSX.Element;
export declare function ButtonGroup(props: ButtonGroupProps): JSX.Element;

export interface TagProps {
  children: ReactNode;
  /** dot = neutral pill with a coloured dot (status); outline = coloured border (Trial term, XP + 10); solid = uppercase caption fill (UI/UX DESIGN). */
  variant?: "dot" | "outline" | "solid";
  tone?: "neutral" | "accent" | "success" | "danger" | "purple" | "teal" | "violet";
}
export interface DataTableColumn<Row = any> {
  key: string;
  header: ReactNode;
  sortable?: boolean;
  /** Value used for sorting when it differs from row[key]. */
  sortValue?: (row: Row) => string | number;
  render?: (row: Row) => ReactNode;
  /** Render row[key] (a string or string[]) as Tags; a function maps each value to Tag props. */
  tags?: boolean | ((value: string, row: Row) => Partial<TagProps>);
  align?: "left" | "right" | "center";
  /** The row's name column: fg-primary, medium weight. */
  primary?: boolean;
  /** Render row[key] as a MediaCell title with an image and subtitle; each field is a row key or a function. */
  media?: { image?: string | ((row: Row) => string); subtitle?: string | ((row: Row) => ReactNode); meta?: string | ((row: Row) => ReactNode); badge?: string | ((row: Row) => ReactNode); shape?: MediaCellProps["shape"]; size?: MediaCellProps["size"] };
  /** Two-line cell: row[key] above a fg-secondary subtitle (a row key or a function), e.g. "$120" over "per year". */
  subtitle?: string | ((row: Row) => ReactNode);
  /** Let this column's text wrap (descriptions). */
  wrap?: boolean;
  maxWidth?: number | string;
  width?: number | string;
}
export interface DataTableSort { key: string; dir: "asc" | "desc" }
export interface DataTableProps<Row = any> {
  columns: DataTableColumn<Row>[];
  rows: Row[];
  /** Field name (default "id") or function giving each row a unique key. */
  rowKey?: string | ((row: Row) => string | number);
  /** Short row name for checkbox labels ("Select row Hola Spine"). */
  rowLabel?: (row: Row) => string;
  title?: ReactNode;
  /** Right side of the header bar when nothing is selected (search, filters, Add). */
  toolbar?: ReactNode;
  caption?: string;
  selectable?: boolean;
  selected?: Array<string | number>;
  defaultSelected?: Array<string | number>;
  onSelectionChange?: (keys: Array<string | number>) => void;
  /** Buttons shown in the bar while rows are selected; receives the keys and a clear() callback. */
  bulkActions?: (keys: Array<string | number>, clear: () => void) => ReactNode;
  /** Per-row buttons in the last column (IconButtons, sm). */
  rowActions?: (row: Row) => ReactNode;
  sort?: DataTableSort | null;
  defaultSort?: DataTableSort;
  onSortChange?: (sort: DataTableSort | null) => void;
  /** Leave sorting to the caller (server-side); the table only reports onSortChange. */
  manualSort?: boolean;
  /** Under the rows, usually a Pagination. */
  footer?: ReactNode;
  empty?: ReactNode;
  density?: "comfortable" | "compact" | "spacious";
}
export interface PaginationProps {
  pageCount: number;
  page?: number;
  defaultPage?: number;
  onChange?: (page: number) => void;
  /** Pages shown either side of the current one (default 1). */
  siblingCount?: number;
  /** numbered (default) or simple ("‹ 3 of 12 ›"). */
  variant?: "numbered" | "simple";
  /** With pageSize, shows "11–20 of 248". */
  total?: number;
  pageSize?: number;
  /** Adds a "Rows per page" Select. */
  pageSizeOptions?: number[];
  onPageSizeChange?: (size: number) => void;
  label?: string;
}
export declare function Tag(props: TagProps): JSX.Element;
export declare function DataTable<Row = any>(props: DataTableProps<Row>): JSX.Element;
export declare function Pagination(props: PaginationProps): JSX.Element;

export interface MediaCellProps {
  title: ReactNode;
  subtitle?: ReactNode;
  /** Third, smaller line in fg-tertiary ("14k views · 1 month ago"). */
  meta?: ReactNode;
  /** An image URL, or a gradient token name ("gradient-blue-1") for an artwork placeholder. Omit for initials. */
  image?: string;
  imageAlt?: string;
  /** circle = person, rounded = app/brand icon, thumb = 16:9 video or course thumbnail. */
  shape?: "circle" | "rounded" | "thumb";
  size?: "md" | "lg";
  /** Overlay on the image's corner, e.g. a duration "4:30". */
  badge?: ReactNode;
}
export declare function MediaCell(props: MediaCellProps): JSX.Element;

export interface HeroHeaderProps {
  /** The headline; it carries the gradient. */
  title: ReactNode;
  /** Optional words before the title, set in plain fg-primary ("Power your" + gradient "finances with AI"). */
  lead?: ReactNode;
  subtitle?: ReactNode;
  /** A short announcement pill above the title (string = glass sm button), or any node such as a Tag. */
  eyebrow?: ReactNode;
  onEyebrowClick?: () => void;
  /** Buttons under the subtitle. */
  actions?: ReactNode;
  /** primary = tg-primary (ink); secondary = tg-secondary (indigo); accent = button-text → chart-2 → chart-3; none = solid fg-primary. */
  gradient?: "primary" | "secondary" | "accent" | "none";
  size?: "display" | "title-1" | "title-2";
  align?: "center" | "left";
  /** Heading element, default h1. */
  as?: "h1" | "h2" | "h3";
}
export declare function HeroHeader(props: HeroHeaderProps): JSX.Element;

export interface MediaCardProps {
  title: ReactNode;
  /** Image URL, or a gradient token name ("gradient-blue-1") as an artwork placeholder. */
  image?: string;
  imageAlt?: string;
  /** vertical = image on top (default); horizontal = image left; overlay = text over the image behind a dark scrim. */
  layout?: "vertical" | "horizontal" | "overlay";
  /** CSS aspect-ratio of the image (vertical) or the whole card (overlay); default 16 / 9, overlay 4 / 5. */
  aspect?: string;
  /** A Tag (or several) above the title: "UI/UX DESIGN". */
  tag?: ReactNode;
  eyebrow?: ReactNode;
  description?: ReactNode;
  /** 0–100: a thin progress bar under the description. */
  progress?: number;
  progressLabel?: string;
  author?: { name: string; role?: string; avatar?: string };
  /** Overlay on the image's bottom-right corner, e.g. a duration "4:30". */
  badge?: ReactNode;
  /** A small IconButton on the image's top-right corner (save, play). */
  mediaAction?: ReactNode;
  actions?: ReactNode;
  /** Makes the whole card a link (the title gets the hit area). */
  href?: string;
  onClick?: () => void;
  interactive?: boolean;
  tint?: CardProps["tint"];
  titleAs?: "h2" | "h3" | "h4";
  animate?: boolean;
  delay?: number;
  style?: object;
}
export declare function MediaCard(props: MediaCardProps): JSX.Element;

export interface UserMenuItem { label: string; icon?: ReactNode | string; href?: string; onSelect?: () => void; tone?: "danger"; shortcut?: string }
export interface UserMenuProps {
  user: { name: string; email?: string; role?: string; avatar?: string; status?: "online" | "busy" | "away" };
  /** Menu rows; defaults to Account, Billing, separator, Sign out. */
  items?: Array<UserMenuItem | "separator">;
  onSelect?: (label: string, item: UserMenuItem) => void;
  /** pill = glass button with avatar, name, role and caret (default); avatar = the avatar alone (headers). */
  variant?: "pill" | "avatar";
  /** Where the menu opens relative to the button. */
  placement?: "bottom-end" | "bottom-start" | "top-start" | "top-end";
  /** Start open (documentation, onboarding). */
  defaultOpen?: boolean;
}
export declare function UserMenu(props: UserMenuProps): JSX.Element;

export interface TopBarItem { label: string; value?: string; href?: string; icon?: ReactNode | string; description?: string; children?: TopBarItem[] }
export interface TopBarProps {
  brand?: { name: string; href?: string; logo?: ReactNode };
  /** Nav links; an item with children opens a tooltip-style dropdown with icon, label and description rows. */
  items?: TopBarItem[];
  value?: string;
  defaultValue?: string;
  onSelect?: (value: string, item: TopBarItem) => void;
  /** true for a default SearchField, or SearchField props. */
  search?: boolean | SearchFieldProps;
  /** IconButtons (notifications, theme) before the user. */
  actions?: ReactNode;
  /** UserMenu props; shown as the avatar variant. */
  user?: UserMenuProps;
  /** A trailing call to action Button ("Buy now", "Get started"). */
  cta?: ReactNode;
  /** bar = full-width glass bar with a bottom hairline (apps); floating = centred glass pill (marketing sites). */
  variant?: "bar" | "floating";
  /** Sticks to the top; on scroll the bar shrinks to 60px and gains shadow-lg. */
  sticky?: boolean;
  label?: string;
}
export interface PageHeaderProps {
  title: ReactNode;
  subtitle?: ReactNode;
  breadcrumbs?: Array<{ label: string; href?: string }>;
  /** Tags above the title. */
  eyebrow?: ReactNode;
  /** A row of small facts under the subtitle (avatars, dates, tags). */
  meta?: ReactNode;
  /** Buttons aligned to the right of the title. */
  actions?: ReactNode;
  /** Usually a SegmentedControl switching the page's views. */
  tabs?: ReactNode;
  gradient?: "primary" | "secondary" | "accent";
  as?: "h1" | "h2";
}
export declare function TopBar(props: TopBarProps): JSX.Element;
export declare function PageHeader(props: PageHeaderProps): JSX.Element;
