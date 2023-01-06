package cache

import (
	"fmt"
	"time"
)

type PRsStats struct {
	Total  int
	Open   int
	Merged int
	Closed int

	DaysOpenAverage    float64
	DaysWaitingAverage float64
	DaysToFirstAverage float64

	DaysToFirstOver int
}

func (cache Cache) CalculateStatsForDateRange(from, to time.Time) (*PRsStats, error) {
	row := cache.DB.QueryRow(fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN state = 'open'  THEN 1 END) as open,
			COUNT(CASE WHEN state = 'closed' THEN 1 END) as closed,
			COUNT(merged) as merged,
			AVG(daysopen) as openAvg,
			AVG(dayswaiting) as waitAvg,
			AVG(daystofirst) as firstAvg,
			COUNT(CASE WHEN daystofirst  > 14 THEN 1 END) as firstGreaterThen
		FROM prs
		WHERE created BETWEEN '%s' AND '%s'
	`, from.Format("2006-01-02"), to.Format("2006-01-02")))

	r := PRsStats{}
	err := row.Scan(
		&r.Total,
		&r.Open,
		&r.Merged,
		&r.Closed,
		&r.DaysOpenAverage,
		&r.DaysWaitingAverage,
		&r.DaysToFirstAverage,
		&r.DaysToFirstOver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}

	return &r, nil
}
