package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/platform/events"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	defaultMaxBufferedRecords = 10_000
	defaultFlushTimeout       = 5 * time.Second
	defaultDeliveryTimeout    = 10 * time.Second
)

type Config struct {
	Brokers            []string
	Topic              string
	ClientID           string
	MaxBufferedRecords int
	FlushTimeout       time.Duration
}

type Producer struct {
	client       *kgo.Client
	topic        string
	flushTimeout time.Duration
	log          logger.Logger
}

func New(config Config, log logger.Logger) (*Producer, error) {
	brokers := normalizedBrokers(config.Brokers)
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}

	topic := strings.TrimSpace(config.Topic)
	if topic == "" {
		return nil, errors.New("kafka access event topic is required")
	}

	clientID := strings.TrimSpace(config.ClientID)
	if clientID == "" {
		return nil, errors.New("kafka client id is required")
	}
	if log == nil {
		return nil, errors.New("logger is required")
	}

	maxBufferedRecords := config.MaxBufferedRecords
	if maxBufferedRecords <= 0 {
		maxBufferedRecords = defaultMaxBufferedRecords
	}

	flushTimeout := config.FlushTimeout
	if flushTimeout <= 0 {
		flushTimeout = defaultFlushTimeout
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID(clientID),
		kgo.DefaultProduceTopic(topic),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.MaxBufferedRecords(maxBufferedRecords),
		kgo.RecordDeliveryTimeout(defaultDeliveryTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}

	return &Producer{
		client:       client,
		topic:        topic,
		flushTimeout: flushTimeout,
		log:          log,
	}, nil
}

func (p *Producer) PublishGatewayAccess(ctx context.Context, event events.GatewayAccessEvent) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate gateway access event: %w", err)
	}

	record, err := accessEventRecord(p.topic, event)
	if err != nil {
		return err
	}

	produceContext := context.Background()
	if ctx != nil {
		// Access-event publishing is asynchronous and must outlive the HTTP request.
		// A cancelable request context would remove the buffered record before Kafka
		// has a chance to send it.
		produceContext = context.WithoutCancel(ctx)
	}

	// TryProduce preserves fail-open semantics: it never blocks the HTTP request.
	// If the local producer buffer is full, the callback receives ErrMaxBuffered.
	p.client.TryProduce(produceContext, record, func(_ *kgo.Record, produceErr error) {
		if produceErr != nil {
			p.log.Warnf(
				"gateway access event delivery failed: event_id=%s topic=%s error=%v",
				event.EventID,
				p.topic,
				produceErr,
			)
		}
	})

	return nil
}

func (p *Producer) Close() error {
	flushContext, cancel := context.WithTimeout(context.Background(), p.flushTimeout)
	defer cancel()

	flushErr := p.client.Flush(flushContext)
	p.client.Close()

	if flushErr != nil {
		return fmt.Errorf("flush kafka producer: %w", flushErr)
	}

	return nil
}

func accessEventRecord(topic string, event events.GatewayAccessEvent) (*kgo.Record, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal gateway access event: %w", err)
	}

	key := strings.TrimSpace(event.RequestID)
	if event.ProjectID != "" && event.RouteName != "" {
		key = event.ProjectID + ":" + event.RouteName
	}

	return &kgo.Record{
		Topic:     topic,
		Key:       []byte(key),
		Value:     payload,
		Timestamp: event.OccurredAt,
		Headers: []kgo.RecordHeader{
			{Key: "event-type", Value: []byte(events.GatewayAccessEventType)},
			{Key: "schema-version", Value: []byte(strconv.Itoa(event.SchemaVersion))},
		},
	}, nil
}

func normalizedBrokers(rawBrokers []string) []string {
	brokers := make([]string, 0, len(rawBrokers))
	seen := make(map[string]struct{}, len(rawBrokers))

	for _, rawBroker := range rawBrokers {
		broker := strings.TrimSpace(rawBroker)
		if broker == "" {
			continue
		}
		if _, exists := seen[broker]; exists {
			continue
		}

		seen[broker] = struct{}{}
		brokers = append(brokers, broker)
	}

	return brokers
}
