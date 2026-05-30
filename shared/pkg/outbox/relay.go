package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jmoiron/sqlx"
	"github.com/twmb/franz-go/pkg/kgo"

	"flashsale/shared/pkg/resilience"
)

// OutboxMessage merepresentasikan satu baris dari tabel outbox_messages.
type OutboxMessage struct {
	ID            int    `db:"id"`
	AggregateType string `db:"aggregate_type"`
	EventType     string `db:"event_type"`
	Payload       string `db:"payload"`
	TracePayload  string `db:"trace_payload"`
}

// RelayWorker adalah komponen yang secara periodik membaca pesan dari tabel
// outbox_messages di PostgreSQL dan mempublishnya ke Kafka.
//
// Desain:
//   - Polling berbasis ticker (bukan push) untuk menjaga kesederhanaan
//   - FOR UPDATE SKIP LOCKED agar aman dijalankan paralel (scale out)
//   - Setiap publish gagal akan di-retry dengan exponential backoff
//   - Pesan yang gagal setelah max retry ditandai 'FAILED' (bukan hilang)
type RelayWorker struct {
	db     *sqlx.DB
	client *kgo.Client
	logger *log.Helper
	retry  resilience.RetryConfig
}

// NewRelayWorker membuat RelayWorker baru yang terhubung ke PostgreSQL dan Kafka.
func NewRelayWorker(db *sqlx.DB, kafkaBrokers []string, logger log.Logger) (*RelayWorker, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(kafkaBrokers...),
		// Ack dari semua replika sebelum dianggap sukses (durabilitas)
		kgo.RequiredAcks(kgo.AllISRAcks()),
		// Compression: gunakan Snappy untuk efisiensi jaringan & I/O tinggi
		kgo.ProducerBatchCompression(kgo.SnappyCompression()),
	)
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi Kafka client: %w", err)
	}

	return &RelayWorker{
		db:     db,
		client: cl,
		logger: log.NewHelper(logger),
		// Retry 5x dengan exponential backoff untuk publish ke Kafka
		retry: resilience.RetryConfig{
			MaxAttempts:     5,
			InitialInterval: 200 * time.Millisecond,
			MaxInterval:     10 * time.Second,
			Multiplier:      2.0,
			Jitter:          true,
		},
	}, nil
}

// Start menjalankan polling loop hingga ctx dibatalkan.
func (w *RelayWorker) Start(ctx context.Context, topic string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer w.client.Close()

	w.logger.Infof("Outbox Relay Worker dimulai untuk topic: %s", topic)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Outbox Relay Worker dihentikan")
			return
		case <-ticker.C:
			w.processPendingMessages(ctx, topic)
		}
	}
}

// processPendingMessages memproses batch pesan PENDING dari outbox_messages.
func (w *RelayWorker) processPendingMessages(ctx context.Context, topic string) {
	// Gunakan transaksi agar FOR UPDATE SKIP LOCKED berfungsi
	tx, err := w.db.BeginTxx(ctx, nil)
	if err != nil {
		w.logger.Errorf("Gagal membuka transaksi outbox: %v", err)
		return
	}
	defer tx.Rollback()

	var msgs []OutboxMessage
	err = tx.SelectContext(ctx, &msgs, `
		SELECT id, aggregate_type, event_type, payload, COALESCE(trace_payload, '') as trace_payload
		FROM outbox_messages
		WHERE status = 'PENDING'
		ORDER BY id ASC
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		w.logger.Errorf("Gagal polling outbox messages: %v", err)
		return
	}

	if len(msgs) == 0 {
		tx.Rollback()
		return
	}

	for _, msg := range msgs {
		// Coba publish ke Kafka dengan retry + exponential backoff
		publishErr := resilience.DoWithRetry(ctx, w.retry, func(attempt int) error {
			if attempt > 1 {
				w.logger.Warnf("Retry publish event id=%d (attempt=%d)", msg.ID, attempt)
			}

			record := &kgo.Record{
				Topic: topic,
				Key:   []byte(msg.AggregateType),
				Value: []byte(msg.Payload),
			}

			// Propagasi trace context ke Kafka header
			if msg.TracePayload != "" {
				record.Headers = append(record.Headers, kgo.RecordHeader{
					Key:   "traceparent",
					Value: []byte(msg.TracePayload),
				})
			}

			return w.client.ProduceSync(ctx, record).FirstErr()
		})

		if publishErr != nil {
			// Tandai FAILED agar mudah dimonitor dan tidak di-retry tanpa batas
			w.logger.Errorf("Gagal publish event id=%d setelah semua retry: %v", msg.ID, publishErr)
			_, _ = tx.ExecContext(ctx,
				"UPDATE outbox_messages SET status = 'FAILED', updated_at = NOW() WHERE id = $1",
				msg.ID,
			)
			continue
		}

		// Tandai SENT jika berhasil
		if _, err = tx.ExecContext(ctx,
			"UPDATE outbox_messages SET status = 'SENT', updated_at = NOW() WHERE id = $1",
			msg.ID,
		); err != nil {
			w.logger.Errorf("Gagal update status SENT untuk id=%d: %v", msg.ID, err)
		} else {
			w.logger.Infof("Event %s (id=%d) berhasil dikirim ke Kafka", msg.EventType, msg.ID)
		}
	}

	if err = tx.Commit(); err != nil {
		w.logger.Errorf("Gagal commit transaksi outbox: %v", err)
	}
}
