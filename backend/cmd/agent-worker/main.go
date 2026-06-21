package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gitflame-codepilot/backend/internal/agent"
	"gitflame-codepilot/backend/internal/config"
	"gitflame-codepilot/backend/internal/domain"
	"gitflame-codepilot/backend/internal/queue"
	"gitflame-codepilot/backend/internal/repository"
	"gitflame-codepilot/backend/internal/service"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check worker dependencies and exit")
	flag.Parse()
	cfg := config.Load()
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" {
		log.Fatal("agent-worker requires DATABASE_URL and REDIS_URL")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := repository.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	broker, err := queue.NewRedisBroker(cfg.RedisURL, cfg.AgentQueueName, cfg.AgentConsumerGroup, cfg.QueueMaxLength)
	if err != nil {
		log.Fatal(err)
	}
	engine := agent.NewClient(cfg.AgentEngineURL, cfg.AgentTimeout)
	if *healthcheck {
		if err := broker.Ping(ctx); err != nil {
			log.Fatal(err)
		}
		if err := engine.Ready(ctx); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := broker.EnsureGroup(ctx); err != nil {
		log.Fatal(err)
	}
	workflow := service.NewWorkflow(store, engine)
	hostname, _ := os.Hostname()
	consumer := hostname + "-1"
	log.Printf("Agent Worker started: stream=%s consumer=%s concurrency=1", cfg.AgentQueueName, consumer)
	for ctx.Err() == nil {
		message, err := broker.Read(ctx, consumer)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			if message != nil && message.ID != "" {
				if deadLetterErr := broker.DeadLetter(ctx, *message, err); deadLetterErr == nil {
					_ = broker.Ack(ctx, message.ID)
				}
			}
			log.Printf("read task: %v", err)
			continue
		}
		if message == nil {
			continue
		}
		process(ctx, workflow, broker, *message, cfg.WorkerMaxRetries)
	}
}

type taskExecutor interface {
	ExecuteTask(context.Context, domain.AgentJob) error
	RetryTask(domain.AgentJob) error
}

func process(ctx context.Context, workflow taskExecutor, broker queue.Broker, message queue.Message, maxRetries int) {
	err := workflow.ExecuteTask(ctx, message.Job)
	if err == nil {
		if ackErr := broker.Ack(ctx, message.ID); ackErr != nil {
			log.Printf("ack completed task %s: %v", message.Job.TaskID, ackErr)
		}
		return
	}
	if temporary(err) && message.Job.Attempt < maxRetries {
		message.Job.Attempt++
		if retryErr := workflow.RetryTask(message.Job); retryErr == nil {
			if publishErr := broker.Publish(ctx, message.Job); publishErr == nil {
				_ = broker.Ack(ctx, message.ID)
				log.Printf("task %s scheduled for retry %d/%d", message.Job.TaskID, message.Job.Attempt, maxRetries)
				return
			}
		}
	}
	if deadLetterErr := broker.DeadLetter(ctx, message, err); deadLetterErr != nil {
		log.Printf("dead-letter task %s: %v", message.Job.TaskID, deadLetterErr)
		return
	}
	_ = broker.Ack(ctx, message.ID)
	log.Printf("task %s failed permanently: %v", message.Job.TaskID, err)
}

func temporary(err error) bool {
	var engineError *agent.Error
	if !errors.As(err, &engineError) {
		return false
	}
	return engineError.Status == http.StatusBadGateway ||
		engineError.Status == http.StatusServiceUnavailable ||
		engineError.Status == http.StatusGatewayTimeout
}
