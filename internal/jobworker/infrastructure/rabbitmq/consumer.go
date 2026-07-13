package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aralary/edgeguard/internal/jobworker/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
	"github.com/aralary/edgeguard/internal/platform/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrConsumerClosed = errors.New("RabbitMQ job consumer closed")

type ConsumerConfig struct {
	URL            string
	Exchange       string
	Queue          string
	DeadExchange   string
	DeadQueue      string
	Prefetch       int
	ProcessTimeout time.Duration
}

type Processor interface {
	Process(ctx context.Context, envelope platformjobs.Envelope) (domain.Result, error)
}

type Consumer struct {
	connection     *amqp.Connection
	channel        *amqp.Channel
	queue          string
	processTimeout time.Duration
}

func NewConsumer(config ConsumerConfig) (*Consumer, error) {
	connection, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("dial RabbitMQ: %w", err)
	}
	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	consumer := &Consumer{
		connection:     connection,
		channel:        channel,
		queue:          config.Queue,
		processTimeout: config.ProcessTimeout,
	}
	if err := consumer.declareTopology(config); err != nil {
		_ = consumer.Close()
		return nil, err
	}
	if err := channel.Qos(config.Prefetch, 0, false); err != nil {
		_ = consumer.Close()
		return nil, fmt.Errorf("configure RabbitMQ prefetch: %w", err)
	}

	return consumer, nil
}

func (consumer *Consumer) Run(ctx context.Context, processor Processor, log logger.Logger) error {
	deliveries, err := consumer.channel.Consume(
		consumer.queue,
		"edgeguard-notification-worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume RabbitMQ jobs: %w", err)
	}

	closed := consumer.channel.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case <-ctx.Done():
			_ = consumer.channel.Cancel("edgeguard-notification-worker", false)
			return nil
		case channelError := <-closed:
			if channelError == nil {
				return ErrConsumerClosed
			}
			return fmt.Errorf("RabbitMQ consumer channel closed: %w", channelError)
		case delivery, ok := <-deliveries:
			if !ok {
				return ErrConsumerClosed
			}
			if err := consumer.handle(ctx, delivery, processor, log); err != nil {
				return err
			}
		}
	}
}

func (consumer *Consumer) handle(ctx context.Context, delivery amqp.Delivery, processor Processor, log logger.Logger) error {
	attempt := deliveryAttempt(delivery.Headers)

	var envelope platformjobs.Envelope
	if err := json.Unmarshal(delivery.Body, &envelope); err != nil {
		log.Errorf("dead-lettering malformed job: message_id=%s attempt=%d error=%v", delivery.MessageId, attempt, err)
		return reject(delivery, false)
	}
	if err := envelope.Validate(); err != nil {
		log.Errorf("dead-lettering invalid job: job_id=%s attempt=%d error=%v", envelope.ID, attempt, err)
		return reject(delivery, false)
	}

	processCtx := ctx
	if consumer.processTimeout > 0 {
		var cancel context.CancelFunc
		processCtx, cancel = context.WithTimeout(ctx, consumer.processTimeout)
		defer cancel()
	}

	result, err := processor.Process(processCtx, envelope)
	if err == nil {
		if err := delivery.Ack(false); err != nil {
			return fmt.Errorf("acknowledge completed job: %w", err)
		}
		log.Infof(
			"job completed: job_id=%s type=%s attempt=%d message=%s output=%s affected_rows=%d",
			envelope.ID,
			envelope.Type,
			attempt,
			result.Message,
			result.OutputPath,
			result.AffectedRows,
		)
		return nil
	}

	if domain.IsPermanent(err) || attempt >= envelope.MaxAttempts {
		log.Errorf(
			"dead-lettering failed job: job_id=%s type=%s attempt=%d max_attempts=%d permanent=%t error=%v",
			envelope.ID,
			envelope.Type,
			attempt,
			envelope.MaxAttempts,
			domain.IsPermanent(err),
			err,
		)
		return reject(delivery, false)
	}

	log.Warnf(
		"returning job for delayed retry: job_id=%s type=%s attempt=%d max_attempts=%d error=%v",
		envelope.ID,
		envelope.Type,
		attempt,
		envelope.MaxAttempts,
		err,
	)
	return reject(delivery, true)
}

func (consumer *Consumer) Close() error {
	var closeError error
	if consumer.channel != nil {
		if err := consumer.channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			closeError = err
		}
	}
	if consumer.connection != nil {
		if err := consumer.connection.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) && closeError == nil {
			closeError = err
		}
	}
	return closeError
}

func (consumer *Consumer) declareTopology(config ConsumerConfig) error {
	if err := consumer.channel.ExchangeDeclare(config.Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare jobs exchange: %w", err)
	}
	if err := consumer.channel.ExchangeDeclare(config.DeadExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare jobs dead-letter exchange: %w", err)
	}

	if _, err := consumer.channel.QueueDeclare(
		config.Queue,
		true,
		false,
		false,
		false,
		amqp.Table{"x-queue-type": "quorum"},
	); err != nil {
		return fmt.Errorf("declare jobs queue: %w", err)
	}
	if err := consumer.channel.QueueBind(config.Queue, "jobs.#", config.Exchange, false, nil); err != nil {
		return fmt.Errorf("bind jobs queue: %w", err)
	}

	if _, err := consumer.channel.QueueDeclare(
		config.DeadQueue,
		true,
		false,
		false,
		false,
		amqp.Table{"x-queue-type": "quorum"},
	); err != nil {
		return fmt.Errorf("declare jobs dead-letter queue: %w", err)
	}
	if err := consumer.channel.QueueBind(config.DeadQueue, "jobs.#", config.DeadExchange, false, nil); err != nil {
		return fmt.Errorf("bind jobs dead-letter queue: %w", err)
	}

	return nil
}

func deliveryAttempt(headers amqp.Table) int {
	count := headerInt64(headers, "x-delivery-count")
	if count < 0 {
		count = 0
	}
	return int(count) + 1
}

func headerInt64(headers amqp.Table, key string) int64 {
	value, ok := headers[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case uint8:
		return int64(typed)
	case uint16:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		if typed > uint64(^uint64(0)>>1) {
			return 0
		}
		return int64(typed)
	default:
		return 0
	}
}

func reject(delivery amqp.Delivery, requeue bool) error {
	if err := delivery.Reject(requeue); err != nil {
		return fmt.Errorf("reject RabbitMQ job: %w", err)
	}
	return nil
}
