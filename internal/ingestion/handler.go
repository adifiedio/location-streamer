package ingestion

import (
	"log"
	"net/http"
	"time"

	"github.com/adifiedio/location-streamer/internal/db"
	"github.com/adifiedio/location-streamer/pkg/queue"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// LocationData represents the streaming payload
type LocationData struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Timestamp int64   `json:"timestamp" binding:"required"` // unix epoch
}

type Handler struct {
	queries  *db.Queries
	producer *queue.Producer
}

func NewHandler(queries *db.Queries, producer *queue.Producer) *Handler {
	return &Handler{
		queries:  queries,
		producer: producer,
	}
}

// IngestLocation accept high frequency location data
func (h *Handler) IngestLocation(c *gin.Context) {
	// identify user
	userSub, exists := c.Get("user_sub")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "identity missing"})
		return
	}

	var req LocationData
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// lookup tenant and user info
	// OPTIMIZATION: In high-scale, cache this user lookup in Redis to avoid DB hit per request.
	user, err := h.queries.GetUserByCognitoSub(c, userSub.(string))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not registered"})
		return
	}

	// prepare data ensure correct pg type for time
	params := db.CreateLocationParams{
		TenantID:  user.TenantID,
		UserID:    user.ID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Timestamp: pgtype.Timestamptz{Time: time.Unix(req.Timestamp, 0), Valid: true},
	}

	// push to kafka using tenant id as the key for ordering within a partition
	// the streamer service will actually insert into db and post to 3rd party
	// this makes "ingestion" extremely fast and decoupled
	err = h.producer.Produce(c, user.TenantID.String(), params)
	if err != nil {
		log.Printf("failed to produce message: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "event buffering failed"})
		return
	}

	c.Status(http.StatusAccepted)
}

// note: the 'worker' function is removed from here because consuming is now
// the responsibility of the separate 'streamer service'
