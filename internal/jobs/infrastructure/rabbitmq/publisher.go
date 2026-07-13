package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrPublishNotConfirmed = errors.New("job publish was not confirmed")
	ErrPublisherClosed     = errors.New("job publisher is closed")
)

type Config struct {
	URL            string
	Exchange       string
	Queue          string
	PublishTimeout time.Duration
}

type Publisher struct {
	connection     *amqp.Connection
	channel        *amqp.Channel
	exchange       string
	publishTimeout time.Duration
	mutex          sync.Mutex
	closed         bool
}

func New(config Config) (*Publisher, error) {
	connection, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("dial RabbitMQ: %w", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	publisher := &Publisher{
		connection:     connection,
		channel:        channel,
		exchange:       config.Exchange,
		publishTimeout: config.PublishTimeout,
	}

	if err := publisher.declareTopology(config.Queue); err != nil {
		_ = publisher.Close()
		return nil, err
	}

	if err := channel.Confirm(false); err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("enable RabbitMQ publisher confirms: %w", err)
	}
	return publisher, nil
}

func (p *Publisher) Publish(ctx context.Context, envelope platformjobs.Envelope) error {
	if err := envelope.Validate(); err != nil {
		return fmt.Errorf("validate job envelope: %w", err)
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal job envelope: %w", err)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return ErrPublisherClosed
	}

	publishCtx := ctx
	if p.publishTimeout > 0 {
		var cancel context.CancelFunc
		publishCtx, cancel = context.WithTimeout(ctx, p.publishTimeout)
		defer cancel()
	}

	message := amqp.Publishing{
		Headers: amqp.Table{
			"schema-version": int32(envelope.SchemaVersion),
			"attempt":        int32(envelope.Attempt),
			"max-attempts":   int32(envelope.MaxAttempts),
		},
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    envelope.ID,
		Type:         string(envelope.Type),
		Timestamp:    envelope.CreatedAt,
		Body:         body,
	}

	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(
		publishCtx,
		p.exchange,
		string(envelope.Type),
		false,
		false,
		message,
	)
	if err != nil {
		return fmt.Errorf("publish RabbitMQ message: %w", err)
	}
	if confirmation == nil {
		return ErrPublishNotConfirmed
	}

	acknowledged, err := confirmation.WaitContext(publishCtx)
	if err != nil {
		return fmt.Errorf("wait for RabbitMQ publish confirmation: %w", err)
	}
	if !acknowledged {
		return ErrPublishNotConfirmed
	}

	return nil
}

func (p *Publisher) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	var closeError error
	if p.channel != nil {
		if err := p.channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			closeError = err
		}
	}
	if p.connection != nil {
		if err := p.connection.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) && closeError == nil {
			closeError = err
		}
	}

	return closeError
}

func (p *Publisher) declareTopology(queueName string) error {
	if err := p.channel.ExchangeDeclare(
		p.exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare jobs exchange: %w", err)
	}

	if _, err := p.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{"x-queue-type": "quorum"},
	); err != nil {
		return fmt.Errorf("declare jobs queue: %w", err)
	}

	if err := p.channel.QueueBind(
		queueName,
		"jobs.#",
		p.exchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind jobs queue: %w", err)
	}

	return nil
}
