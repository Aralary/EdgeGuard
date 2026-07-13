package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/aralary/edgeguard/internal/analytics/usecase"
	"github.com/aralary/edgeguard/internal/platform/events"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/twmb/franz-go/pkg/kgo"
)

const defaultPingTimeout = 10 * time.Second

type GatewayAccessProcessor interface {
	ProcessGatewayAccess(
		ctx context.Context,
		event events.GatewayAccessEvent,
	) (domain.IngestionResult, error)
}

type Config struct {
	Brokers            []string
	Topic              string
	ConsumerGroup      string
	ClientID           string
	RetryMin           time.Duration
	RetryMax           time.Duration
	MaxProcessAttempts int
}

type Consumer struct {
	client             *kgo.Client
	processor          GatewayAccessProcessor
	retryMin           time.Duration
	retryMax           time.Duration
	maxProcessAttempts int
	log                logger.Logger
}

func New(
	ctx context.Context,
	config Config,
	processor GatewayAccessProcessor,
	log logger.Logger,
) (*Consumer, error) {
	brokers := normalized(config.Brokers)
	if len(brokers) == 0 {
		return nil, errors.New("Kafka brokers are required")
	}

	topic := strings.TrimSpace(config.Topic)
	if topic == "" {
		return nil, errors.New("Kafka topic is required")
	}

	consumerGroup := strings.TrimSpace(config.ConsumerGroup)
	if consumerGroup == "" {
		return nil, errors.New("Kafka consumer group is required")
	}

	clientID := strings.TrimSpace(config.ClientID)
	if clientID == "" {
		return nil, errors.New("Kafka client ID is required")
	}
	if processor == nil {
		return nil, errors.New("gateway access processor is required")
	}
	if log == nil {
		return nil, errors.New("logger is required")
	}
	if config.RetryMin <= 0 {
		return nil, errors.New("retry minimum must be positive")
	}
	if config.RetryMax < config.RetryMin {
		return nil, errors.New("retry maximum must be greater than or equal to retry minimum")
	}
	if config.MaxProcessAttempts <= 0 {
		return nil, errors.New("maximum process attempts must be positive")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID(clientID),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return nil, fmt.Errorf("create Kafka analytics consumer: %w", err)
	}

	pingCtx := ctx
	if pingCtx == nil {
		pingCtx = context.Background()
	}
	pingCtx, cancel := context.WithTimeout(pingCtx, defaultPingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping Kafka analytics broker: %w", err)
	}

	return &Consumer{
		client:             client,
		processor:          processor,
		retryMin:           config.RetryMin,
		retryMax:           config.RetryMax,
		maxProcessAttempts: config.MaxProcessAttempts,
		log:                log,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		fetches := c.client.PollRecords(ctx, 1)

		for _, fetchErr := range fetches.Errors() {
			c.log.Warnf(
				"Kafka analytics fetch failed: topic=%s partition=%d error=%v",
				fetchErr.Topic,
				fetchErr.Partition,
				fetchErr.Err,
			)
		}

		var handleErr error
		for _, record := range fetches.Records() {
			handleErr = c.handleRecord(ctx, record)
		}

		// BlockRebalanceOnPoll is enabled, so every poll must release the group
		// after the fetched record has either been committed or failed.
		c.client.AllowRebalance()

		if ctx.Err() != nil {
			return nil
		}
		if handleErr != nil {
			return handleErr
		}
	}
}

func (c *Consumer) Close() {
	if c == nil || c.client == nil {
		return
	}

	c.client.CloseAllowingRebalance()
}

func (c *Consumer) handleRecord(ctx context.Context, record *kgo.Record) error {
	event, err := decodeGatewayAccessEvent(record.Value)
	if err != nil {
		c.log.Errorf(
			"skipping invalid gateway access event: topic=%s partition=%d offset=%d error=%v",
			record.Topic,
			record.Partition,
			record.Offset,
			err,
		)

		return c.commitRecord(ctx, record)
	}

	result, err := c.processWithRetry(ctx, event, record)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidGatewayAccessEvent) {
			c.log.Errorf(
				"skipping permanently invalid gateway access event: event_id=%s partition=%d offset=%d error=%v",
				event.EventID,
				record.Partition,
				record.Offset,
				err,
			)

			return c.commitRecord(ctx, record)
		}

		return err
	}

	if result.Duplicate {
		c.log.Debugf(
			"gateway access event already processed: event_id=%s partition=%d offset=%d",
			event.EventID,
			record.Partition,
			record.Offset,
		)
	}

	return c.commitRecord(ctx, record)
}

func (c *Consumer) processWithRetry(
	ctx context.Context,
	event events.GatewayAccessEvent,
	record *kgo.Record,
) (domain.IngestionResult, error) {
	delay := c.retryMin
	attempt := 1

	for attempt <= c.maxProcessAttempts {
		result, err := c.processor.ProcessGatewayAccess(ctx, event)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, usecase.ErrInvalidGatewayAccessEvent) {
			return domain.IngestionResult{}, err
		}
		if ctx.Err() != nil {
			return domain.IngestionResult{}, ctx.Err()
		}

		if attempt == c.maxProcessAttempts {
			return domain.IngestionResult{}, fmt.Errorf(
				"process gateway access event after %d attempts: %w",
				c.maxProcessAttempts,
				err,
			)
		}

		c.log.Warnf(
			"gateway access event processing failed, retrying: event_id=%s partition=%d offset=%d attempt=%d retry_in=%s error=%v",
			event.EventID,
			record.Partition,
			record.Offset,
			attempt,
			delay,
			err,
		)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return domain.IngestionResult{}, ctx.Err()
		case <-timer.C:
		}

		delay = nextRetryDelay(delay, c.retryMax)
		attempt++
	}

	return domain.IngestionResult{}, errors.New("gateway access event processing attempts exhausted")
}

func (c *Consumer) commitRecord(ctx context.Context, record *kgo.Record) error {
	if err := c.client.CommitRecords(ctx, record); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return fmt.Errorf(
			"commit Kafka analytics offset: topic=%s partition=%d offset=%d: %w",
			record.Topic,
			record.Partition,
			record.Offset,
			err,
		)
	}

	return nil
}

func decodeGatewayAccessEvent(payload []byte) (events.GatewayAccessEvent, error) {
	var event events.GatewayAccessEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return events.GatewayAccessEvent{}, fmt.Errorf("decode gateway access event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return events.GatewayAccessEvent{}, fmt.Errorf("validate gateway access event: %w", err)
	}

	return event, nil
}

func nextRetryDelay(current time.Duration, maximum time.Duration) time.Duration {
	if current >= maximum {
		return maximum
	}
	if current > maximum/2 {
		return maximum
	}

	return current * 2
}

func normalized(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}

		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}
