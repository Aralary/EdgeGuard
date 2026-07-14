package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	jobsdomain "github.com/aralary/edgeguard/internal/jobs/domain"
	"github.com/aralary/edgeguard/internal/jobworker/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
	"github.com/aralary/edgeguard/internal/platform/logger"
	platformtracing "github.com/aralary/edgeguard/internal/platform/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
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

type StatusStore interface {
	PrepareAttempt(ctx context.Context, envelope platformjobs.Envelope, attempt int, at time.Time) (jobsdomain.Status, error)
	MarkRetrying(ctx context.Context, jobID string, attempt int, failure string, at time.Time) error
	MarkSucceeded(ctx context.Context, jobID string, attempt int, result jobsdomain.ExecutionResult, at time.Time) error
	MarkFailed(ctx context.Context, jobID string, attempt int, failure string, at time.Time) error
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

func (consumer *Consumer) Ping(context.Context) error {
	if consumer == nil || consumer.connection == nil || consumer.connection.IsClosed() || consumer.channel == nil || consumer.channel.IsClosed() {
		return ErrConsumerClosed
	}

	return nil
}

func (consumer *Consumer) Run(ctx context.Context, processor Processor, statusStore StatusStore, log logger.Logger) error {
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
			if err := consumer.handle(ctx, delivery, processor, statusStore, log); err != nil {
				return err
			}
		}
	}
}

func (consumer *Consumer) handle(
	ctx context.Context,
	delivery amqp.Delivery,
	processor Processor,
	statusStore StatusStore,
	log logger.Logger,
) (handleErr error) {
	traceHeaders := make(map[string]string)
	for name, value := range delivery.Headers {
		switch typed := value.(type) {
		case string:
			traceHeaders[name] = typed
		case []byte:
			traceHeaders[name] = string(typed)
		}
	}
	ctx = platformtracing.Extract(ctx, traceHeaders)
	attempt := deliveryAttempt(delivery.Headers)
	ctx, span := otel.Tracer("edgeguard.jobworker.rabbitmq").Start(
		ctx,
		"rabbitmq process "+delivery.Type,
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
		oteltrace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", consumer.queue),
			attribute.String("messaging.operation.type", "process"),
			attribute.String("messaging.message.id", delivery.MessageId),
			attribute.String("messaging.rabbitmq.routing_key", delivery.RoutingKey),
			attribute.Int("edgeguard.job.attempt", attempt),
		),
	)
	defer func() {
		if handleErr != nil {
			span.RecordError(handleErr)
			span.SetStatus(codes.Error, handleErr.Error())
		}
		span.End()
	}()

	return consumer.handleDelivery(ctx, delivery, processor, statusStore, log)
}

func (consumer *Consumer) handleDelivery(
	ctx context.Context,
	delivery amqp.Delivery,
	processor Processor,
	statusStore StatusStore,
	log logger.Logger,
) error {
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

	status, err := statusStore.PrepareAttempt(ctx, envelope, attempt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("prepare job status: %w", err)
	}
	if status == jobsdomain.StatusSucceeded {
		oteltrace.SpanFromContext(ctx).SetAttributes(attribute.String("edgeguard.job.outcome", "already_succeeded"))
		log.Infof("acknowledging already completed job: job_id=%s type=%s", envelope.ID, envelope.Type)
		return delivery.Ack(false)
	}
	if status == jobsdomain.StatusFailed {
		span := oteltrace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String("edgeguard.job.outcome", "already_failed"))
		span.SetStatus(codes.Error, "job already failed")
		log.Warnf("dead-lettering already failed job: job_id=%s type=%s", envelope.ID, envelope.Type)
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
		oteltrace.SpanFromContext(ctx).SetAttributes(attribute.String("edgeguard.job.outcome", "succeeded"))
		if err := statusStore.MarkSucceeded(ctx, envelope.ID, attempt, jobsdomain.ExecutionResult{
			Message:      result.Message,
			OutputPath:   result.OutputPath,
			AffectedRows: result.AffectedRows,
		}, time.Now().UTC()); err != nil {
			return fmt.Errorf("persist completed job status: %w", err)
		}
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

	span := oteltrace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	if domain.IsPermanent(err) || attempt >= envelope.MaxAttempts {
		span.SetAttributes(attribute.String("edgeguard.job.outcome", "failed"))
		if statusError := statusStore.MarkFailed(ctx, envelope.ID, attempt, err.Error(), time.Now().UTC()); statusError != nil {
			return fmt.Errorf("persist failed job status: %w", statusError)
		}
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

	span.SetAttributes(attribute.String("edgeguard.job.outcome", "retrying"))
	if statusError := statusStore.MarkRetrying(ctx, envelope.ID, attempt, err.Error(), time.Now().UTC()); statusError != nil {
		return fmt.Errorf("persist retrying job status: %w", statusError)
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
