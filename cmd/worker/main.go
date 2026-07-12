// cmd/worker/main.go
//
// Background worker for processing asynchronous jobs.
// Handles: WhatsApp/Email delivery, exports, reminders, and scheduled tasks.
//
// Usage:
//
//	go run cmd/worker/main.go              # Start worker with default queues
//	go run cmd/worker/main.go -queues=all  # Process all queue types
//	go run cmd/worker/main.go -queues=whatsapp,email  # Specific queues only
//	go run cmd/worker/main.go -concurrency=5          # 5 parallel workers
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"guestflow/internal/config"
	"guestflow/internal/repository"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// JobType represents the type of background job.
type JobType string

const (
	JobWhatsApp JobType = "whatsapp"
	JobEmail    JobType = "email"
	JobExport   JobType = "export"
	JobReminder JobType = "reminder"
	JobReport   JobType = "report"
	JobCleanup  JobType = "cleanup"
)

// Job represents a unit of work.
type Job struct {
	ID          string
	Type        JobType
	Payload     map[string]interface{}
	TenantID    string
	CreatedAt   time.Time
	Attempts    int
	MaxAttempts int
}

// Worker processes jobs from a queue.
type Worker struct {
	id      int
	jobType JobType
	db      *sqlx.DB
	redis   *redis.Client
	stop    chan struct{}
	wg      *sync.WaitGroup
}

// Queue keys in Redis.
const (
	QueuePrefix      = "guestflow:queue:"
	DeadLetterPrefix = "guestflow:dlq:"
	JobLockPrefix    = "guestflow:lock:"
)

func main() {
	// Parse flags
	var (
		queues       = flag.String("queues", "all", "Comma-separated queue types (whatsapp,email,export,reminder,report,cleanup,all)")
		concurrency  = flag.Int("concurrency", 3, "Number of parallel workers per queue")
		pollInterval = flag.Duration("poll", 5*time.Second, "Queue polling interval")
	)
	flag.Parse()

	initLogger()

	slog.Info("starting GuestFlow worker",
		"queues", *queues,
		"concurrency", *concurrency,
		"poll_interval", *pollInterval,
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := repository.NewPostgresConnection(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Connect to Redis
	redisClient, err := repository.NewRedisConnection(cfg.Redis)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Parse queue types
	queueTypes := parseQueues(*queues)
	if len(queueTypes) == 0 {
		slog.Error("no valid queues specified")
		os.Exit(1)
	}

	// Start workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	workerCount := 0

	for _, qt := range queueTypes {
		for i := 0; i < *concurrency; i++ {
			workerCount++
			wg.Add(1)
			w := &Worker{
				id:      workerCount,
				jobType: qt,
				db:      db,
				redis:   redisClient,
				stop:    make(chan struct{}),
				wg:      &wg,
			}
			go w.run(ctx, *pollInterval)
		}
	}

	slog.Info("workers started", "count", workerCount)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down workers...")
	cancel()
	wg.Wait()
	slog.Info("all workers stopped")
}

// run starts the worker loop.
func (w *Worker) run(ctx context.Context, pollInterval time.Duration) {
	defer w.wg.Done()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("worker stopping", "worker", w.id, "queue", w.jobType)
			return
		case <-ticker.C:
			if err := w.processJob(ctx); err != nil {
				if err != context.Canceled {
					slog.Error("job processing error",
						"worker", w.id,
						"queue", w.jobType,
						"error", err,
					)
				}
			}
		}
	}
}

// processJob fetches and processes one job from the queue.
func (w *Worker) processJob(ctx context.Context) error {
	queueKey := QueuePrefix + string(w.jobType)

	// Try to get a job from Redis list (RPOP)
	result, err := w.redis.BRPop(ctx, 0, queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // No jobs available
		}
		return fmt.Errorf("queue pop failed: %w", err)
	}

	if len(result) < 2 {
		return nil
	}

	jobData := result[1]

	slog.Info("processing job",
		"worker", w.id,
		"queue", w.jobType,
		"job_data", jobData,
	)

	// Process based on job type
	switch w.jobType {
	case JobWhatsApp:
		return w.processWhatsApp(ctx, jobData)
	case JobEmail:
		return w.processEmail(ctx, jobData)
	case JobExport:
		return w.processExport(ctx, jobData)
	case JobReminder:
		return w.processReminder(ctx, jobData)
	case JobReport:
		return w.processReport(ctx, jobData)
	case JobCleanup:
		return w.processCleanup(ctx, jobData)
	default:
		return fmt.Errorf("unknown job type: %s", w.jobType)
	}
}

func (w *Worker) processWhatsApp(ctx context.Context, jobData string) error {
	// TODO: Integrate with WhatsApp Business API
	// 1. Parse job payload
	// 2. Load template
	// 3. Render message with variables
	// 4. Send via WhatsApp Business Cloud API
	// 5. Update delivery status
	slog.Info("processing WhatsApp message", "worker", w.id, "job", jobData)
	return nil
}

func (w *Worker) processEmail(ctx context.Context, jobData string) error {
	// TODO: Integrate with SMTP/transactional email provider
	// 1. Parse job payload
	// 2. Render email template
	// 3. Send via SMTP
	// 4. Update delivery status
	slog.Info("processing email", "worker", w.id, "job", jobData)
	return nil
}

func (w *Worker) processExport(ctx context.Context, jobData string) error {
	// TODO: Generate Excel/CSV export
	// 1. Parse export parameters
	// 2. Query database
	// 3. Generate file
	// 4. Upload to object storage
	// 5. Notify user
	slog.Info("processing export", "worker", w.id, "job", jobData)
	return nil
}

func (w *Worker) processReminder(ctx context.Context, jobData string) error {
	// TODO: Send reminder messages
	// 1. Parse reminder parameters
	// 2. Load guest list based on filter
	// 3. Queue individual messages
	slog.Info("processing reminder", "worker", w.id, "job", jobData)
	return nil
}

func (w *Worker) processReport(ctx context.Context, jobData string) error {
	// TODO: Generate scheduled reports
	// 1. Parse report parameters
	// 2. Aggregate data
	// 3. Generate PDF/Excel
	// 4. Store and notify
	slog.Info("processing report", "worker", w.id, "job", jobData)
	return nil
}

func (w *Worker) processCleanup(ctx context.Context, jobData string) error {
	// TODO: Cleanup tasks
	// 1. Expired token cleanup
	// 2. Old audit log archiving
	// 3. Soft-deleted record purging (after retention period)
	slog.Info("processing cleanup", "worker", w.id, "job", jobData)
	return nil
}

// parseQueues converts a comma-separated queue list to JobType slice.
func parseQueues(queues string) []JobType {
	if queues == "all" {
		return []JobType{JobWhatsApp, JobEmail, JobExport, JobReminder, JobReport, JobCleanup}
	}

	var result []JobType
	for _, q := range strings.Split(queues, ",") {
		q = strings.TrimSpace(q)
		switch q {
		case "whatsapp":
			result = append(result, JobWhatsApp)
		case "email":
			result = append(result, JobEmail)
		case "export":
			result = append(result, JobExport)
		case "reminder":
			result = append(result, JobReminder)
		case "report":
			result = append(result, JobReport)
		case "cleanup":
			result = append(result, JobCleanup)
		}
	}
	return result
}

// initLogger initializes structured logging.
func initLogger() {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}
