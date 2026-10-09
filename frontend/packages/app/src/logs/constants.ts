import type { Impact, LogStatus } from "../lib/types";

export const IMPACTS: Impact[] = ["low", "medium", "high", "critical"];
export const STATUSES: LogStatus[] = ["idea", "in_progress", "done", "dropped"];
export const STATUS_LABEL: Record<LogStatus, string> = {
  idea: "Idea",
  in_progress: "In progress",
  done: "Done",
  dropped: "Dropped",
};

/** The article's sections, offered alongside the tenant's own tags (PRD-0002 FR-10). */
export const SUGGESTED_TAGS = [
  "project",
  "collaboration",
  "mentorship",
  "design",
  "documentation",
  "company-building",
  "learning",
  "outside-of-work",
];

export const SORTS = [
  { value: "-created_at", label: "Newest first" },
  { value: "created_at", label: "Oldest first" },
  { value: "name", label: "Name A–Z" },
  { value: "-name", label: "Name Z–A" },
  { value: "-impact", label: "Impact: highest first" },
  { value: "impact", label: "Impact: lowest first" },
  { value: "status", label: "Status: idea → dropped" },
  { value: "-status", label: "Status: dropped → idea" },
];

/** Native <select>/<textarea> styled like the ui Input. */
export const FIELD =
  "rounded-md border border-input bg-transparent px-3 py-1.5 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50";
