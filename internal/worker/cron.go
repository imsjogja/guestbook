package worker

import (
	"context"
	"log"
	"time"

	"guestflow/internal/service"
)

// StartCronJob starts a background goroutine that runs periodic tasks.
// For a production app, a robust cron scheduler like robfig/cron is recommended.
func StartCronJob(ctx context.Context, billingSvc *service.BillingService) {
	go func() {
		// Run once on startup
		if err := billingSvc.ProcessExpiredSubscriptions(ctx); err != nil {
			log.Printf("[Cron] Error processing expired subscriptions on startup: %v", err)
		} else {
			log.Println("[Cron] Successfully processed expired subscriptions on startup")
		}

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[Cron] Stopping cron worker")
				return
			case <-ticker.C:
				if err := billingSvc.ProcessExpiredSubscriptions(ctx); err != nil {
					log.Printf("[Cron] Error processing expired subscriptions: %v", err)
				} else {
					log.Println("[Cron] Successfully processed expired subscriptions")
				}
			}
		}
	}()
}
