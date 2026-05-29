package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/go-kratos/kratos/v2/log"
)

type OutboxMessage struct {
	ID            int    `db:"id"`
	AggregateType string `db:"aggregate_type"`
	EventType     string `db:"event_type"`
	Payload       string `db:"payload"`
}

type RelayWorker struct {
	db     *sqlx.DB
	client *kgo.Client
	logger *log.Helper
}

func NewRelayWorker(db *sqlx.DB, kafkaBrokers []string, logger log.Logger) (*RelayWorker, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(kafkaBrokers...),
	)
	if err != nil {
		return nil, err
	}

	return &RelayWorker{
		db:     db,
		client: cl,
		logger: log.NewHelper(logger),
	}, nil
}

func (w *RelayWorker) Start(ctx context.Context, topic string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	w.logger.Infof("Starting Outbox Relay Worker for topic: %s", topic)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping Outbox Relay Worker")
			w.client.Close()
			return
		case <-ticker.C:
			w.processPendingMessages(ctx, topic)
		}
	}
}

func (w *RelayWorker) processPendingMessages(ctx context.Context, topic string) {
	// Polling messages from database
	var msgs []OutboxMessage
	err := w.db.SelectContext(ctx, &msgs, `
		SELECT id, aggregate_type, event_type, payload 
		FROM outbox_messages 
		WHERE status = 'PENDING' 
		ORDER BY id ASC LIMIT 50 FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		w.logger.Errorf("Failed to poll outbox messages: %v", err)
		return
	}

	if len(msgs) == 0 {
		return
	}

	for _, msg := range msgs {
		record := &kgo.Record{
			Topic: topic,
			Key:   []byte(msg.AggregateType),
			Value: []byte(msg.Payload),
		}

		// Karena ini scaffold, kita pakai produce sync
		if err := w.client.ProduceSync(ctx, record).FirstErr(); err != nil {
			w.logger.Errorf("Failed to produce to Kafka (id=%d): %v", msg.ID, err)
			continue
		}

		// Update status jadi SENT
		_, err = w.db.ExecContext(ctx, "UPDATE outbox_messages SET status = 'SENT' WHERE id = $1", msg.ID)
		if err != nil {
			w.logger.Errorf("Failed to update status to SENT (id=%d): %v", msg.ID, err)
		} else {
			w.logger.Infof("Successfully relayed event %s to Kafka", msg.EventType)
		}
	}
}
