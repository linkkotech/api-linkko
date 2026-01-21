package repo

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func toStrPtr(t pgtype.Text) *string {
	if t.Valid {
		return &t.String
	}
	return nil
}

func toInt32Ptr(i pgtype.Int4) *int32 {
	if i.Valid {
		return &i.Int32
	}
	return nil
}

func toFloat64Ptr(f pgtype.Float8) *float64 {
	if f.Valid {
		return &f.Float64
	}
	return nil
}

func toTimePtr(ts pgtype.Timestamp) *time.Time {
	if ts.Valid {
		t := ts.Time
		return &t
	}
	return nil
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
