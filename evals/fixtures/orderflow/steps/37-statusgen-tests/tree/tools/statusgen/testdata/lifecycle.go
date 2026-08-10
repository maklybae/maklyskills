package sample

type Status uint8

const (
	StatusDraft Status = iota
	StatusInTransit
	StatusDone
)

const (
	maxRetries    = 3
	defaultBucket = "orders"
)

type Priority uint8

const (
	PriorityLow Priority = iota
	PriorityHigh
)
