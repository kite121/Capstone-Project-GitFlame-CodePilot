package queue

import (
	"context"
	"errors"

	"gitflame-codepilot/backend/internal/domain"
)

var ErrQueueFull = errors.New("agent task queue is full")

type Message struct {
	ID  string
	Job domain.AgentJob
}

type Broker interface {
	Ping(context.Context) error
	Publish(context.Context, domain.AgentJob) error
	EnsureGroup(context.Context) error
	Read(context.Context, string) (*Message, error)
	Ack(context.Context, string) error
	DeadLetter(context.Context, Message, error) error
}
