export type Me = {
  id: string;
  email: string;
  display_name: string;
  role: "admin" | "member";
  tenant: { id: string; name: string };
};

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
};

export type DocumentList = { owned: Document[]; shared: Document[] };

export type Member = {
  id: string;
  email: string;
  display_name: string;
  role: "admin" | "member";
  created_at: string;
};

export type Invitation = { id: string; email: string; created_at: string };

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
