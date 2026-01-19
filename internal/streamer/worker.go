package streamer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/adifiedio/location-streamer/internal/db"
	"github.com/adifiedio/location-streamer/pkg/queue"
)

type Worker struct {
	queries  *db.Queries
	consumer *queue.Consumer
	client   *http.Client
}

func NewWorker(queries *db.Queries, consumer *queue.Consumer) *Worker {
	return &Worker{
		queries:  queries,
		consumer: consumer,
		client:   &http.Client{Timeout: 5 * time.Second}, // fail fast for streaming
	}
}

func (w *Worker) Start(ctx context.Context) {
	log.Println("streamer worker started...")
	for {
		select {
		case <-ctx.Done():
			log.Println("streamer worker context done, shutting down")
			return
		default:
			log.Println("waiting for next kafka message...")
			// consume message
			msg, err := w.consumer.Consume(ctx)
			if err != nil {
				log.Printf("error consuming message: %v", err)
				continue
			}
			log.Printf("received message from kafka: key=%s, length=%d", string(msg.Key), len(msg.Value))

			// parse data
			var loc db.CreateLocationParams
			if err := json.Unmarshal(msg.Value, &loc); err != nil {
				log.Printf("error parsing message value: %v, raw: %s", err, string(msg.Value))
				continue
			}
			log.Printf("successfully parsed location: lat=%f, lng=%f", loc.Latitude, loc.Longitude)

			// persist to database (use separate context to avoid canceling DB writes)
			log.Printf("archiving location to db for tenant %v...", loc.TenantID)
			if _, err := w.queries.CreateLocation(context.Background(), loc); err != nil {
				log.Printf("error archiving location to db: %v", err)
				// TODO: we can implement DLQ here
			} else {
				log.Println("successfully archived location to db")
			}

			// stream to 3rd party
			log.Println("spawning goroutine for webhook stream...")
			go w.processStream(loc)
		}
	}
}

func (w *Worker) processStream(loc db.CreateLocationParams) {
	// get tenant configuration (webhook URL & API Key)
	// optimizing by fetching invalid tenant ID would happen here, but we trust the ID from the event flow
	tenant, err := w.queries.GetTenant(context.Background(), loc.TenantID)
	if err != nil {
		log.Printf("failed to get tenant config for %v: %v", loc.TenantID, err)
		return
	}

	if !tenant.WebhookUrl.Valid || tenant.WebhookUrl.String == "" {
		// no streaming configured for this tenant
		return
	}

	// forward data
	payload := map[string]interface{}{
		"lat":       loc.Latitude,
		"lng":       loc.Longitude,
		"timestamp": loc.Timestamp.Time.Unix(),
		"user_id":   loc.UserID,
	}

	var body []byte
	var req *http.Request
	body, err = json.Marshal(payload)
	if err != nil {
		log.Printf("failed to marshal webhook payload for tenant %s: %v", tenant.Name, err)
		return
	}

	req, err = http.NewRequest("POST", tenant.WebhookUrl.String, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("failed to create webhook request for tenant %s: %v", tenant.Name, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if tenant.ApiKey.Valid {
		req.Header.Set("X-Api-Key", tenant.ApiKey.String)
	}

	// execute with exponential backoff retry
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	var resp *http.Response

	for attempt := 0; attempt < maxRetries; attempt++ {
		// clone the request body for retries (as body is consumed on first attempt)
		var bodyReader *bytes.Reader
		if attempt > 0 {
			bodyReader = bytes.NewReader(body)
			req.Body = io.NopCloser(bodyReader)
		}

		resp, err = w.client.Do(req)
		if err != nil {
			// network error, retry with backoff
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(attempt)) // 100, 200, 400ms
				log.Printf("webhook attempt %d/%d failed for tenant %s: %v, retrying in %v",
					attempt+1, maxRetries, tenant.Name, err, delay)
				time.Sleep(delay)
				continue
			}
			log.Printf("webhook failed after %d attempts for tenant %s: %v", maxRetries, tenant.Name, err)
			return
		}

		// check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// success
			resp.Body.Close()
			// log.Printf("successfully streamed location for tenant %s", tenant.Name)
			return
		}

		// 4xx errors are client errors, don't retry
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			log.Printf("tenant %s webhook returned client error: %d (not retrying)", tenant.Name, resp.StatusCode)
			resp.Body.Close()
			return
		}

		// 5xx errors are server errors, so retry
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(attempt))
				log.Printf("webhook attempt %d/%d returned %d for tenant %s, retrying in %v",
					attempt+1, maxRetries, resp.StatusCode, tenant.Name, delay)
				time.Sleep(delay)
				continue
			}
			log.Printf("webhook failed after %d attempts for tenant %s: status %d",
				maxRetries, tenant.Name, resp.StatusCode)
			return
		}

		// unexpected status code
		resp.Body.Close()
		return
	}
}
