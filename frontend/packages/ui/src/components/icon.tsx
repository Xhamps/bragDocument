import type * as React from "react";
import {
  ArrowLeft,
  ArrowRight,
  Bell,
  BookOpen,
  ChartColumn,
  Check,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  ChevronsUpDown,
  CircleHelp,
  CirclePlus,
  CreditCard,
  Download,
  Ellipsis,
  ExternalLink,
  Eye,
  History,
  House,
  LayoutGrid,
  LogOut,
  Mail,
  Menu,
  Minus,
  Moon,
  Pencil,
  Play,
  Plus,
  Search,
  Send,
  Settings,
  Share,
  SlidersHorizontal,
  Sun,
  Trash2,
  User,
  X,
  type LucideIcon,
} from "lucide-react";

const ICONS: Record<string, LucideIcon> = {
  chevron: ChevronRight,
  chevronDown: ChevronDown,
  search: Search,
  check: Check,
  plus: Plus,
  minus: Minus,
  mail: Mail,
  arrowLeft: ArrowLeft,
  arrowRight: ArrowRight,
  download: Download,
  card: CreditCard,
  plusCircle: CirclePlus,
  moon: Moon,
  sun: Sun,
  play: Play,
  home: House,
  bell: Bell,
  sliders: SlidersHorizontal,
  eye: Eye,
  share: Share,
  close: X,
  external: ExternalLink,
  menu: Menu,
  user: User,
  signOut: LogOut,
  settings: Settings,
  grid: LayoutGrid,
  book: BookOpen,
  chart: ChartColumn,
  help: CircleHelp,
  sort: ChevronsUpDown,
  sortUp: ChevronUp,
  sortDown: ChevronDown,
  more: Ellipsis,
  edit: Pencil,
  trash: Trash2,
  send: Send,
  history: History,
};

export type IconProp = React.ReactNode | string;

function Icon({
  name,
  size = 20,
  className,
}: {
  name: string;
  size?: number;
  className?: string;
}) {
  const C = ICONS[name];
  return C ? (
    <C size={size} strokeWidth={1.5} aria-hidden="true" className={className} />
  ) : null;
}

function renderIcon(icon: IconProp, size = 20) {
  return typeof icon === "string" ? <Icon name={icon} size={size} /> : icon;
}

export { Icon, renderIcon };
