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
	"github.com/jackc/pgx/v5/pgconn"
)

// DailyActivityStore defines the data access methods used by daily activity handlers.
type DailyActivityStore interface {
	CreateDailyActivity(ctx context.Context, arg *sqlcdb.CreateDailyActivityParams) (sqlcdb.DailyActivity, error)
	GetDailyActivity(ctx context.Context, id uuid.UUID) (sqlcdb.DailyActivity, error)
	ListDailyActivities(ctx context.Context, arg *sqlcdb.ListDailyActivitiesParams) ([]sqlcdb.DailyActivity, error)
	UpdateDailyActivity(ctx context.Context, arg *sqlcdb.UpdateDailyActivityParams) (sqlcdb.DailyActivity, error)
	DeleteDailyActivity(ctx context.Context, id uuid.UUID) error
}

// --- Request / Response types ---

type CreateDailyActivityRequest struct {
	ActivityDate   string               `json:"activity_date" validate:"required"`
	Steps          *int32               `json:"steps" validate:"omitempty,min=0"`
	CommuteMode    *sqlcdb.CommuteMode  `json:"commute_mode" validate:"omitempty,oneof=scooter train walk other"`
	CommuteMinutes *int32               `json:"commute_minutes" validate:"omitempty,min=0"`
	Note           *string              `json:"note"`
}

type UpdateDailyActivityRequest struct {
	Steps          *int32               `json:"steps" validate:"omitempty,min=0"`
	CommuteMode    *sqlcdb.CommuteMode  `json:"commute_mode" validate:"omitempty,oneof=scooter train walk other"`
	CommuteMinutes *int32               `json:"commute_minutes" validate:"omitempty,min=0"`
	Note           *string              `json:"note"`
}

type DailyActivityResponse struct {
	ID             string  `json:"id"`
	ActivityDate   string  `json:"activity_date"`
	Steps          *int32  `json:"steps"`
	CommuteMode    *string `json:"commute_mode"`
	CommuteMinutes *int32  `json:"commute_minutes"`
	Note           *string `json:"note"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func int32PtrToNullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}

func nullCommuteModeToStringPtr(n sqlcdb.NullCommuteMode) *string {
	if !n.Valid {
		return nil
	}
	s := string(n.CommuteMode)
	return &s
}

func commuteModeToNull(cm *sqlcdb.CommuteMode) sqlcdb.NullCommuteMode {
	if cm == nil {
		return sqlcdb.NullCommuteMode{}
	}
	return sqlcdb.NullCommuteMode{CommuteMode: *cm, Valid: true}
}

func toDailyActivityResponse(a sqlcdb.DailyActivity) DailyActivityResponse {
	return DailyActivityResponse{
		ID:             a.ID.String(),
		ActivityDate:   a.ActivityDate.Format("2006-01-02"),
		Steps:          nullInt32ToPtr(a.Steps),
		CommuteMode:    nullCommuteModeToStringPtr(a.CommuteMode),
		CommuteMinutes: nullInt32ToPtr(a.CommuteMinutes),
		Note:           nullStringToStringPtr(a.Note),
		CreatedAt:      a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      a.UpdatedAt.Format(time.RFC3339),
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// --- Handlers ---

func CreateDailyActivity(store DailyActivityStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateDailyActivityRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", err.Error()))
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

		date, err := time.Parse("2006-01-02", req.ActivityDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid activity_date, expected YYYY-MM-DD"))
			return
		}

		activity, err := store.CreateDailyActivity(c.Request.Context(), &sqlcdb.CreateDailyActivityParams{
			ActivityDate:   date,
			Steps:          int32PtrToNullInt32(req.Steps),
			CommuteMode:    commuteModeToNull(req.CommuteMode),
			CommuteMinutes: int32PtrToNullInt32(req.CommuteMinutes),
			Note:           stringPtrToNullString(req.Note),
		})
		if err != nil {
			if isUniqueViolation(err) {
				c.JSON(http.StatusConflict, errResponse("CONFLICT", "activity for this date already exists"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to create daily activity"))
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": toDailyActivityResponse(activity)})
	}
}

func ListDailyActivities(store DailyActivityStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromStr := c.Query("from")
		toStr := c.Query("to")
		limitStr := c.DefaultQuery("limit", "100")

		limit := int32(100)
		if l, err := strconv.ParseInt(limitStr, 10, 32); err == nil && l > 0 {
			limit = int32(l)
		}

		params := &sqlcdb.ListDailyActivitiesParams{Limit: limit}

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

		activities, err := store.ListDailyActivities(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to list daily activities"))
			return
		}

		responses := make([]DailyActivityResponse, len(activities))
		for i, a := range activities {
			responses[i] = toDailyActivityResponse(a)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": responses,
			"meta": gin.H{"total": len(activities), "from": fromStr, "to": toStr},
		})
	}
}

func UpdateDailyActivity(store DailyActivityStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid id"))
			return
		}

		var req UpdateDailyActivityRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", err.Error()))
			return
		}
		if req.Steps == nil && req.CommuteMode == nil && req.CommuteMinutes == nil && req.Note == nil {
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

		updated, err := store.UpdateDailyActivity(c.Request.Context(), &sqlcdb.UpdateDailyActivityParams{
			ID:             id,
			Steps:          int32PtrToNullInt32(req.Steps),
			CommuteMode:    commuteModeToNull(req.CommuteMode),
			CommuteMinutes: int32PtrToNullInt32(req.CommuteMinutes),
			Note:           stringPtrToNullString(req.Note),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "daily activity not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to update daily activity"))
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": toDailyActivityResponse(updated)})
	}
}

func DeleteDailyActivity(store DailyActivityStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", "invalid id"))
			return
		}

		if _, err := store.GetDailyActivity(c.Request.Context(), id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "daily activity not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to check daily activity"))
			return
		}

		if err := store.DeleteDailyActivity(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", "failed to delete daily activity"))
			return
		}

		c.Status(http.StatusNoContent)
	}
}
