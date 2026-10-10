import { cn } from "@bragdoc/ui";
import logo from "../assets/logo.svg";

/** The brand mark. Decorative: the "Brag Document" name always sits next to it. */
export function Logo({ className }: { className?: string }) {
  return <img src={logo} alt="" className={cn("h-[26px] w-auto", className)} />;
}
