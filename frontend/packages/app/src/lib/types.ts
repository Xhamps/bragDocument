export type Me = {
  id: string;
  email: string;
  display_name: string;
  role: "admin" | "member";
  tenant: { id: string; name: string };
};

export type DocRole = "owner" | "editor" | "viewer";
export type GrantRole = Exclude<DocRole, "owner">;

export type Document = {
  id: string;
  owner_id: string;
  title: string;
  description: string;
  state: "active" | "archived";
  created_at: string;
  updated_at: string;
  log_count: number;
  last_log_at: string | null;
  role: DocRole;
  /** Present on shared documents. */
  owner_name?: string;
  /** Shared with the caller and not opened yet. */
  is_new: boolean;
};

export type DocumentList = { owned: Document[]; shared: Document[] };

export type Member = {
  id: string;
  email: string;
  display_name: string;
  role: "admin" | "member";
  created_at: string;
};

export type Invitation = {
  id: string;
  email: string;
  created_at: string;
  /** Set for document invitations. */
  document_title?: string;
};

export type Impact = "low" | "medium" | "high" | "critical";
export type LogStatus = "idea" | "in_progress" | "done" | "dropped";
export type LogLink = { url: string; label: string };

export type Log = {
  id: string;
  document_id: string;
  name: string;
  description: string;
  impact: Impact;
  /** null: not checked; "": checked, none found (PRD-0007). */
  impact_statement: string | null;
  status: LogStatus;
  is_example: boolean;
  tags: string[];
  links: LogLink[];
  created_at: string;
  created_by: string;
  updated_at: string;
  updated_by: string;
};

export type LogList = { items: Log[]; total: number };

export type TelegramStatus = {
  linked: boolean;
  linked_at?: string;
  document_id?: string;
};

export type TelegramCode = {
  code: string;
  expires_at: string;
  bot_url?: string;
};

export type Grant = {
  user_id: string;
  email: string;
  display_name: string;
  role: GrantRole;
  granted_at: string;
};

export type DocumentInvitation = {
  id: string;
  email: string;
  role: GrantRole;
  created_at: string;
};

export type Sharing = { grants: Grant[]; invitations: DocumentInvitation[] };

export type AuditEntry = {
  id: number;
  actor_email: string;
  action:
    | "grant"
    | "invite"
    | "role_change"
    | "revoke"
    | "invite_cancel"
    | "invite_accept"
    | "transfer";
  document_id: string;
  document_title: string;
  target: string;
  role: string;
  at: string;
};

export type Bucket = { key: string; count: number };

export type Dashboard = {
  from: string;
  to: string;
  total: number;
  in_period: number;
  high_impact: number;
  in_progress: number;
  months: Bucket[];
  tags: Bucket[];
  statuses: Bucket[];
  impacts: Bucket[];
  coverage: Bucket[];
};

export type ExportStatus = "queued" | "running" | "done" | "failed";

export type ExportJob = {
  id: string;
  status: ExportStatus;
  progress: number;
  error: string;
  created_at: string;
  expires_at: string;
  downloadable: boolean;
};

export type ReportSettings = {
  goals_this_year: string;
  goals_next_year: string;
  section_map: Record<string, string>;
};
