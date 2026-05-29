package kafka

import (
	"context"
	"encoding/json"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/go-kratos/kratos/v2/log"

	"flashsale/order-service/internal/application/usecase"
	"flashsale/order-service/internal/domain/model"
)

type Consumer struct {
	client  *kgo.Client
	usecase *usecase.OrderSagaUsecase
	logger  *log.Helper
}

func NewKafkaConsumer(brokers []string, groupID string, uc *usecase.OrderSagaUsecase, logger log.Logger) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics("flashsale.inventory.events", "flashsale.payment.events"),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:  cl,
		usecase: uc,
		logger:  log.NewHelper(logger),
	}, nil
}

func (c *Consumer) Start(ctx context.Context) {
	c.logger.Info("Starting Order Service Kafka Consumer")

	for {
		select {
		case <-ctx.Done():
			c.client.Close()
			return
		default:
			fetches := c.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				c.logger.Errorf("Poll errors: %v", errs)
				continue
			}

			fetches.EachRecord(func(record *kgo.Record) {
				c.logger.Infof("Received message on topic %s", record.Topic)

				switch record.Topic {
				case "flashsale.inventory.events":
					var event model.StockReservedEvent
					if err := json.Unmarshal(record.Value, &event); err == nil {
						c.usecase.HandleStockReserved(ctx, &event)
					} else {
						c.logger.Errorf("Failed to unmarshal StockReservedEvent: %v", err)
					}

				case "flashsale.payment.events":
					var event model.PaymentCompletedEvent
					if err := json.Unmarshal(record.Value, &event); err == nil {
						c.usecase.HandlePaymentCompleted(ctx, &event)
					} else {
						c.logger.Errorf("Failed to unmarshal PaymentCompletedEvent: %v", err)
					}
				}
			})
		}
	}
}
