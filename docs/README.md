# Brag Document System

A multi-tenant web application for keeping **brag documents**: a running record of the work you did, why it mattered, and what you learned. The concept and the content guidance come from Julia Evans' article [Get your work recognized: write a brag document](https://jvns.ca/blog/brag-documents/).

## Why a brag document

- People forget their own accomplishments within months ("wait, what *did* I do in the last 6 months?").
- Managers cannot remember everything you did; a written record lets them advocate for you in reviews and promotions.
- Writing it down surfaces themes, growth, and gaps in your own work.
- When managers change, the document transfers the history.

The article's guidance that shapes the product:

| Guidance | How the product applies it |
|---|---|
| Keep it as a living document, updated as things happen | Logs are added continuously, from the UI or from a Telegram bot, with a creation date |
| Explain the **impact**, not just the activity | Every log has an explicit `impact` field |
| Include fuzzy, collaborative, and non-technical work | Tags such as `mentorship`, `documentation`, `company-building`, `learning`, `outside-of-work` are built in and reportable |
| Do not oversell: describe it "exactly as good as it is" | Free-text fields plus links to evidence (PRs, docs, dashboards) |
| Share with your manager before reviews, and with peers | Document-level RBAC with invitations, and a PDF report to hand over |
| 5 to 10 pages a year is normal | Filtering and a dashboard make a year of logs navigable |

## What the product does

- A **tenant** (company or team) has users. Users authenticate through Supabase.
- A user owns one or more **brag documents** (for example, one per year or one per role).
- A document holds **logs**. Each log has: name, description, tags, reference links, impact, status, and creation date.
- Logs are added from the web UI or by sending a message to a **Telegram bot** linked to the user.
- Document owners **invite** other users and assign a role (owner, editor, viewer) on the document.
- A **logs page** lists and filters logs by every field.
- When a log is saved, an LLM extracts its impact statement from the description and warns when none is stated.
- A **dashboard** shows metrics about the logs (volume over time, by tag, by status, by impact).
- A **PDF report** of a document, or a filtered subset, can be generated to share with a manager or team.

Detailed requirements live in the PRDs under [`prd/`](prd/).

## Technical shape

| Concern | Decision | ADR |
|---|---|---|
| Backend | Go with the Gin HTTP framework, hexagonal layout, one Cobra binary | [ADR-0001](adr/0001-go-and-gin-for-the-backend.md), [ADR-0012](adr/0012-hexagonal-backend-layout.md) |
| Frontend | React with Tailwind CSS | [ADR-0002](adr/0002-react-and-tailwind-for-the-frontend.md) |
| Repository | Monorepo: `backend/` and `frontend/` side by side | [ADR-0003](adr/0003-monorepo-layout.md) |
| Identity | Supabase Auth as the identity provider | [ADR-0004](adr/0004-supabase-as-identity-provider.md) |
| Database | PostgreSQL, provisioned locally with Docker | [ADR-0005](adr/0005-postgresql-as-primary-database.md) |
| Cache | Redis, provisioned locally with Docker | [ADR-0006](adr/0006-redis-as-cache.md) |
| Multi-tenancy | Shared database, shared schema, `tenant_id` column with row-level security | [ADR-0007](adr/0007-multi-tenancy-strategy.md) |
| Local environment | `docker-compose.yml` provisions every service | [ADR-0008](adr/0008-docker-compose-for-local-provisioning.md) |
| Telegram | Bot built on the Telegram Bot API, long polling locally, webhook in production | [ADR-0009](adr/0009-telegram-bot-integration.md) |
| PDF | Server-rendered HTML converted by Gotenberg | [ADR-0010](adr/0010-pdf-generation-with-gotenberg.md) |
| Authorization | Role-based access control per document, enforced in the API | [ADR-0011](adr/0011-rbac-model.md) |
| Impact extraction | OpenAI Chat Completions with structured output, optional (disabled without a key) | [ADR-0013](adr/0013-openai-for-impact-extraction.md) |
| Email | Resend for share notifications, optional (disabled without a key) | [ADR-0014](adr/0014-resend-for-transactional-email.md) |

## Repository layout

```
.
├── docker-compose.yml     # postgres, redis, gotenberg, backend, frontend
├── backend/               # Go + Gin API (and the Telegram bot worker)
├── frontend/              # npm workspace
│   └── packages/
│       ├── ui/            # @bragdoc/ui: design system (Tailwind v4 tokens, shadcn-style components)
│       └── app/           # @bragdoc/app: React + Tailwind SPA
└── docs/
    ├── README.md          # this file
    ├── prd/               # Product Requirements Documents
    │   ├── TEMPLATE.md
    │   └── NNNN-*.md
    └── adr/               # Architecture Decision Records (MADR)
        ├── TEMPLATE.md
        └── NNNN-*.md
```

## Documentation rules

Every decision is written down. There are two kinds of decision, with one folder each.

- **Product decisions** go in `docs/prd/` as a PRD. Copy `prd/TEMPLATE.md`, number it with the next free `NNNN`, and keep the sections in order. A PRD says *what* and *why*, never *how*.
- **Architecture decisions** go in `docs/adr/` as an ADR in the [MADR](https://adr.github.io/madr/) format. Copy `adr/TEMPLATE.md`, number it with the next free `NNNN`. ADRs are immutable once accepted: a change is a new ADR that supersedes the old one.
- File names are `NNNN-title-with-dashes.md`, lowercase.
- A PRD that forces a technical choice links to the ADR that records that choice, and vice versa.
- Status values: `proposed`, `accepted`, `rejected`, `deprecated`, `superseded by …`.

## Index

### PRDs

| # | Title | Status |
|---|---|---|
| [0001](prd/0001-tenants-users-and-brag-documents.md) | Tenants, users, and brag documents | accepted |
| [0002](prd/0002-logs.md) | Logs: fields, creation, and the filtered list page | accepted |
| [0003](prd/0003-telegram-bot.md) | Adding logs through a Telegram bot | accepted |
| [0004](prd/0004-sharing-and-rbac.md) | Sharing documents: invitations and roles | accepted |
| [0005](prd/0005-dashboard.md) | Dashboard with log metrics | accepted |
| [0006](prd/0006-pdf-report.md) | PDF report for sharing with the team | accepted |
| [0007](prd/0007-impact-extraction.md) | Impact statement extracted from the description | accepted |
| [0008](prd/0008-run-all-services-and-service-tagged-logs.md) | Run all backend services with one command, with service-tagged JSON logs | proposed |
| [0009](prd/0009-audit-log.md) | Audit log for every action | proposed |

### ADRs

| # | Title | Status |
|---|---|---|
| [0000](adr/0000-use-madr-for-architecture-decisions.md) | Use MADR for architecture decisions | accepted |
| [0001](adr/0001-go-and-gin-for-the-backend.md) | Go and Gin for the backend | accepted |
| [0002](adr/0002-react-and-tailwind-for-the-frontend.md) | React and Tailwind CSS for the frontend | accepted |
| [0003](adr/0003-monorepo-layout.md) | Monorepo layout | accepted |
| [0004](adr/0004-supabase-as-identity-provider.md) | Supabase as identity provider | accepted |
| [0005](adr/0005-postgresql-as-primary-database.md) | PostgreSQL as primary database | accepted |
| [0006](adr/0006-redis-as-cache.md) | Redis as cache | accepted |
| [0007](adr/0007-multi-tenancy-strategy.md) | Multi-tenancy: shared schema with row-level security | accepted |
| [0008](adr/0008-docker-compose-for-local-provisioning.md) | Docker Compose for local provisioning | accepted |
| [0009](adr/0009-telegram-bot-integration.md) | Telegram bot integration | accepted |
| [0010](adr/0010-pdf-generation-with-gotenberg.md) | PDF generation with Gotenberg | accepted |
| [0011](adr/0011-rbac-model.md) | RBAC model for documents | accepted |
| [0012](adr/0012-hexagonal-backend-layout.md) | Hexagonal backend layout with a single Cobra binary | accepted |
| [0013](adr/0013-openai-for-impact-extraction.md) | OpenAI for impact extraction | accepted |
| [0014](adr/0014-resend-for-transactional-email.md) | Resend for transactional email | accepted |
