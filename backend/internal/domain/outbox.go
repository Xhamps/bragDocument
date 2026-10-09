package domain

// OutboxMessage is one event written in an action's transaction and relayed
// to a stream (ADR-0015). Payload is the topic's JSON document.
type OutboxMessage struct {
	ID       int64
	TenantID string
	Topic    string
	Payload  []byte
}

// TopicAudit carries PRD-0009 audit entries.
const TopicAudit = "audit.entry"
