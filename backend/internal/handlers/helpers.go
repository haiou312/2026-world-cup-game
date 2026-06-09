package handlers

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func textVal(t pgtype.Text) any {
	if !t.Valid {
		return nil
	}
	return t.String
}

func int8val(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

func int4val(v pgtype.Int4) any {
	if v.Valid {
		return v.Int32
	}
	return nil
}

func tsVal(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time.Format(time.RFC3339)
}

func dateVal(d pgtype.Date) any {
	if !d.Valid {
		return nil
	}
	return d.Time.Format("2006-01-02")
}

func maskKey(k string) string {
	if len(k) <= 4 {
		return "••••"
	}
	return "••••" + k[len(k)-4:]
}

func playersOrEmpty(p []string) []string {
	if p == nil {
		return []string{}
	}
	return p
}
