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
