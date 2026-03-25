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

// SleepLogStore defines the data access methods used by sleep log handlers.
type SleepLogStore interface {
	CreateSleepLog(ctx context.Context, arg *sqlcdb.CreateSleepLogParams) (sqlcdb.SleepLog, error)
	GetSleepLog(ctx context.Context, id uuid.UUID) (sqlcdb.SleepLog, error)
	ListSleepLogs(ctx context.Context, arg *sqlcdb.ListSleepLogsParams) ([]sqlcdb.SleepLog, error)
	UpdateSleepLog(ctx context.Context, arg *sqlcdb.UpdateSleepLogParams) (sqlcdb.SleepLog, error)
	DeleteSleepLog(ctx context.Context, id uuid.UUID) error
}

// --- Request / Response types ---

type CreateSleepLogRequest struct {
	SleepAt time.Time `json:"sleep_at"`
	WakeAt  time.Time `json:"wake_at"`
	Quality *int16    `json:"quality" validate:"omitempty,min=1,max=5"`
	Note    *string   `json:"note"`
}

type UpdateSleepLogRequest struct {
	SleepAt *time.Time `json:"sleep_at"`
	WakeAt  *time.Time `json:"wake_at"`
	Quality *int16     `json:"quality" validate:"omitempty,min=1,max=5"`
	Note    *string    `json:"note"`
}

type SleepLogResponse struct {
	ID           string  `json:"id"`
	SleepAt      string  `json:"sleep_at"`
	WakeAt       string  `json:"wake_at"`
	DurationMin  *int32  `json:"duration_min"`
	AbnormalWake bool    `json:"abnormal_wake"`
	Quality      *int16  `json:"quality"`
	Note         *string `json:"note"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// nullInt32ToPtr converts sql.NullInt32 to *int32.
func nullInt32ToPtr(n sql.NullInt32) *int32 {
	if !n.Valid {
		return nil
	}
	return &n.Int32
}

func toSleepLogResponse(s sqlcdb.SleepLog) SleepLogResponse {
	return SleepLogResponse{
		ID:           s.ID.String(),
		SleepAt:      s.SleepAt.Format(time.RFC3339),
		WakeAt:       s.WakeAt.Format(time.RFC3339),
		DurationMin:  nullInt32ToPtr(s.DurationMin),
		AbnormalWake: s.AbnormalWake,
		Quality:      nullInt16ToInt16Ptr(s.Quality),
		Note:         nullStringToStringPtr(s.Note),
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Handlers ---

func CreateSleepLog(store SleepLogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateSleepLogRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", err.Error()))
			return
		}
		if req.SleepAt.IsZero() {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "sleep_at is required"))
			return
		}
		if req.WakeAt.IsZero() {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "wake_at is required"))
			return
		}
		if !req.WakeAt.After(req.SleepAt) {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "wake_at must be after sleep_at"))
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

		ctx, cancel := withTimeout(c)
		defer cancel()
		log, err := store.CreateSleepLog(ctx, &sqlcdb.CreateSleepLogParams{
			SleepAt: req.SleepAt,
			WakeAt:  req.WakeAt,
			Quality: int16PtrToNullInt16(req.Quality),
			Note:    stringPtrToNullString(req.Note),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to create sleep log"))
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": toSleepLogResponse(log)})
	}
}

func ListSleepLogs(store SleepLogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromStr := c.Query("from")
		toStr := c.Query("to")
		abnormalOnly := c.Query("abnormal_only") == "true"
		limitStr := c.DefaultQuery("limit", "100")

		limit := int32(100)
		if l, err := strconv.ParseInt(limitStr, 10, 32); err == nil && l > 0 {
			limit = int32(l)
		}

		params := &sqlcdb.ListSleepLogsParams{Limit: limit}

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
		if abnormalOnly {
			params.AbnormalOnly = sql.NullBool{Bool: true, Valid: true}
		}

		ctx, cancel := withTimeout(c)
		defer cancel()
		logs, err := store.ListSleepLogs(ctx, params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to list sleep logs"))
			return
		}

		responses := make([]SleepLogResponse, len(logs))
		for i, l := range logs {
			responses[i] = toSleepLogResponse(l)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": responses,
			"meta": gin.H{"total": len(logs), "from": fromStr, "to": toStr},
		})
	}
}

func UpdateSleepLog(store SleepLogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid id"))
			return
		}

		var req UpdateSleepLogRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", err.Error()))
			return
		}
		if req.SleepAt == nil && req.WakeAt == nil && req.Quality == nil && req.Note == nil {
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

		ctx, cancel := withTimeout(c)
		defer cancel()

		existing, err := store.GetSleepLog(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "sleep log not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to fetch sleep log"))
			return
		}

		resolvedSleepAt := existing.SleepAt
		if req.SleepAt != nil {
			resolvedSleepAt = *req.SleepAt
		}
		resolvedWakeAt := existing.WakeAt
		if req.WakeAt != nil {
			resolvedWakeAt = *req.WakeAt
		}
		if !resolvedWakeAt.After(resolvedSleepAt) {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "wake_at must be after sleep_at"))
			return
		}

		var sleepAtNull, wakeAtNull sql.NullTime
		if req.SleepAt != nil {
			sleepAtNull = sql.NullTime{Time: *req.SleepAt, Valid: true}
		}
		if req.WakeAt != nil {
			wakeAtNull = sql.NullTime{Time: *req.WakeAt, Valid: true}
		}

		updated, err := store.UpdateSleepLog(ctx, &sqlcdb.UpdateSleepLogParams{
			ID:      id,
			SleepAt: sleepAtNull,
			WakeAt:  wakeAtNull,
			Quality: int16PtrToNullInt16(req.Quality),
			Note:    stringPtrToNullString(req.Note),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "sleep log not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to update sleep log"))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": toSleepLogResponse(updated)})
	}
}

func DeleteSleepLog(store SleepLogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid id"))
			return
		}

		ctx, cancel := withTimeout(c)
		defer cancel()
		if _, err := store.GetSleepLog(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "sleep log not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to check sleep log"))
			return
		}

		if err := store.DeleteSleepLog(ctx, id); err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to delete sleep log"))
			return
		}

		c.Status(http.StatusNoContent)
	}
}
