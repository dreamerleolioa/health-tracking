package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	sqlcdb "health-tracking/backend/db/sqlc"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

// Store defines the data access methods used by body metrics handlers.
type Store interface {
	CreateBodyMetric(ctx context.Context, arg *sqlcdb.CreateBodyMetricParams) (sqlcdb.BodyMetric, error)
	GetBodyMetric(ctx context.Context, id uuid.UUID) (sqlcdb.BodyMetric, error)
	ListBodyMetrics(ctx context.Context, arg *sqlcdb.ListBodyMetricsParams) ([]sqlcdb.BodyMetric, error)
	UpdateBodyMetric(ctx context.Context, arg *sqlcdb.UpdateBodyMetricParams) (sqlcdb.BodyMetric, error)
	DeleteBodyMetric(ctx context.Context, id uuid.UUID) error
}

// --- Request / Response types ---

type CreateBodyMetricRequest struct {
	WeightKg    *float64  `json:"weight_kg" validate:"omitempty,min=30,max=300"`
	BodyFatPct  *float64  `json:"body_fat_pct" validate:"omitempty,min=1,max=70"`
	MusclePct   *float64  `json:"muscle_pct" validate:"omitempty,min=10,max=80"`
	VisceralFat *int16    `json:"visceral_fat" validate:"omitempty,min=1,max=30"`
	RecordedAt  time.Time `json:"recorded_at"`
	Note        *string   `json:"note"`
}

type UpdateBodyMetricRequest struct {
	WeightKg    *float64 `json:"weight_kg" validate:"omitempty,min=30,max=300"`
	BodyFatPct  *float64 `json:"body_fat_pct" validate:"omitempty,min=1,max=70"`
	MusclePct   *float64 `json:"muscle_pct" validate:"omitempty,min=10,max=80"`
	VisceralFat *int16   `json:"visceral_fat" validate:"omitempty,min=1,max=30"`
	Note        *string  `json:"note"`
}

type BodyMetricResponse struct {
	ID          string   `json:"id"`
	WeightKg    *float64 `json:"weight_kg"`
	BodyFatPct  *float64 `json:"body_fat_pct"`
	MusclePct   *float64 `json:"muscle_pct"`
	VisceralFat *int16   `json:"visceral_fat"`
	RecordedAt  string   `json:"recorded_at"`
	Note        *string  `json:"note"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type errorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type apiError struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []errorDetail `json:"details,omitempty"`
}

func errResponse(code, message string, details ...errorDetail) gin.H {
	return gin.H{"error": apiError{Code: code, Message: message, Details: details}}
}

// --- Type converters ---

func nullStringToFloat64(s sql.NullString) *float64 {
	if !s.Valid || s.String == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s.String, 64)
	if err != nil {
		return nil
	}
	return &f
}

func float64ToNullString(f *float64) sql.NullString {
	if f == nil {
		return sql.NullString{}
	}
	return sql.NullString{
		String: strconv.FormatFloat(*f, 'f', -1, 64),
		Valid:  true,
	}
}

func nullInt16ToInt16Ptr(n sql.NullInt16) *int16 {
	if !n.Valid {
		return nil
	}
	return &n.Int16
}

func int16PtrToNullInt16(i *int16) sql.NullInt16 {
	if i == nil {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: *i, Valid: true}
}

func nullStringToStringPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func stringPtrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toBodyMetricResponse(m sqlcdb.BodyMetric) BodyMetricResponse {
	return BodyMetricResponse{
		ID:          m.ID.String(),
		WeightKg:    nullStringToFloat64(m.WeightKg),
		BodyFatPct:  nullStringToFloat64(m.BodyFatPct),
		MusclePct:   nullStringToFloat64(m.MusclePct),
		VisceralFat: nullInt16ToInt16Ptr(m.VisceralFat),
		RecordedAt:  m.RecordedAt.Format(time.RFC3339),
		Note:        nullStringToStringPtr(m.Note),
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Handlers ---

func CreateBodyMetric(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateBodyMetricRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", err.Error()))
			return
		}
		if req.RecordedAt.IsZero() {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "recorded_at is required"))
			return
		}
		if err := validate.Struct(req); err != nil {
			var verrs validator.ValidationErrors
			errors.As(err, &verrs)
			details := make([]errorDetail, len(verrs))
			for i, fe := range verrs {
				details[i] = errorDetail{Field: fe.Field(), Message: fe.Tag()}
			}
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "validation failed", details...))
			return
		}

		metric, err := store.CreateBodyMetric(c.Request.Context(), &sqlcdb.CreateBodyMetricParams{
			WeightKg:    float64ToNullString(req.WeightKg),
			BodyFatPct:  float64ToNullString(req.BodyFatPct),
			MusclePct:   float64ToNullString(req.MusclePct),
			VisceralFat: int16PtrToNullInt16(req.VisceralFat),
			RecordedAt:  req.RecordedAt,
			Note:        stringPtrToNullString(req.Note),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to create body metric"))
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": toBodyMetricResponse(metric)})
	}
}

func ListBodyMetrics(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromStr := c.Query("from")
		toStr := c.Query("to")
		limitStr := c.DefaultQuery("limit", "100")

		params := &sqlcdb.ListBodyMetricsParams{Limit: 100}
		if l, err := strconv.ParseInt(limitStr, 10, 32); err == nil && l > 0 {
			params.Limit = int32(l)
		}

		var fromTime, toTime time.Time
		if fromStr != "" {
			t, err := time.Parse("2006-01-02", fromStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid from date, expected YYYY-MM-DD"))
				return
			}
			fromTime = t
			params.From = sql.NullTime{Time: t, Valid: true}
		}
		if toStr != "" {
			t, err := time.Parse("2006-01-02", toStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid to date, expected YYYY-MM-DD"))
				return
			}
			toTime = t
			params.To = sql.NullTime{Time: t, Valid: true}
		}
		if !fromTime.IsZero() && !toTime.IsZero() && toTime.Before(fromTime) {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "to must not be earlier than from"))
			return
		}

		metrics, err := store.ListBodyMetrics(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to list body metrics"))
			return
		}

		responses := make([]BodyMetricResponse, len(metrics))
		for i, m := range metrics {
			responses[i] = toBodyMetricResponse(m)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": responses,
			"meta": gin.H{"total": len(metrics), "from": fromStr, "to": toStr},
		})
	}
}

func UpdateBodyMetric(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid id"))
			return
		}

		var req UpdateBodyMetricRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", err.Error()))
			return
		}
		if req.WeightKg == nil && req.BodyFatPct == nil && req.MusclePct == nil && req.VisceralFat == nil && req.Note == nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "no fields to update"))
			return
		}
		if err := validate.Struct(req); err != nil {
			var verrs validator.ValidationErrors
			errors.As(err, &verrs)
			details := make([]errorDetail, len(verrs))
			for i, fe := range verrs {
				details[i] = errorDetail{Field: fe.Field(), Message: fe.Tag()}
			}
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "validation failed", details...))
			return
		}

		metric, err := store.UpdateBodyMetric(c.Request.Context(), &sqlcdb.UpdateBodyMetricParams{
			ID:          id,
			WeightKg:    float64ToNullString(req.WeightKg),
			BodyFatPct:  float64ToNullString(req.BodyFatPct),
			MusclePct:   float64ToNullString(req.MusclePct),
			VisceralFat: int16PtrToNullInt16(req.VisceralFat),
			Note:        stringPtrToNullString(req.Note),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "body metric not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to update body metric"))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": toBodyMetricResponse(metric)})
	}
}

func DeleteBodyMetric(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid id"))
			return
		}

		if _, err := store.GetBodyMetric(c.Request.Context(), id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "body metric not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to check body metric"))
			return
		}

		if err := store.DeleteBodyMetric(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to delete body metric"))
			return
		}

		c.Status(http.StatusNoContent)
	}
}
