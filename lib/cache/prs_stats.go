package cache

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type PRsStats struct {
	Total  int
	Open   int
	Merged int
	Closed int

	DaysOpenAverage    sql.NullFloat64
	DaysWaitingAverage sql.NullFloat64
	DaysToFirstAverage sql.NullFloat64

	DaysToFirstOver int
}

func (cache Cache) CalculateRepoPRStatsForDateRange(from, to time.Time, repos []string, authors []string) (*PRsStats, error) {
	authorClause := ""
	if len(authors) > 0 {
		authorClause = " AND user in ('" + strings.Join(authors, "', '") + "')"
	}

	repoClause := ""
	if len(repos) > 0 {
		repoClause = " AND repo in ('" + strings.Join(repos, "', '") + "')"
	}

	// COUNT(case WHEN merged is 'true' THEN 1 END) as merged,
	q := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN state = 'open'  THEN 1 END) as open,
			COUNT(CASE WHEN state = 'closed' THEN 1 END) as closed,
			COUNT(case WHEN merger != '' THEN 1 END) as merged,
			AVG(daysopen) as openAvg,
			AVG(dayswaiting) as waitAvg,
			AVG(daystofirst) as firstAvg,
			COUNT(CASE WHEN daystofirst  > 14 THEN 1 END) as firstGreaterThen
		FROM prs
		WHERE 
		    created BETWEEN '%s' AND '%s' %s %s
	`, from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"), authorClause, repoClause)
	row := cache.DB.QueryRow(q)

	r := PRsStats{}
	err := row.Scan(
		&r.Total,
		&r.Open,
		&r.Closed,
		&r.Merged,
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
