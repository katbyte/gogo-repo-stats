package cache

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
)

// TODO switch to an ORM ?

const ColumnsPR = "repo, number, title, user, state, milestone, merged, merger, created, closed, daysopen, dayswaiting, daystofirst"

type PR struct {
	Repo      string
	Number    int
	Title     string
	User      string
	State     string // todo should we make this boolean "open" or 2 boolean so we have open/closed/merged ?
	Milestone string
	Merged    bool
	Merger    string
	Created   time.Time
	Closed    time.Time

	// calculated
	DaysOpen    sql.NullFloat64
	DaysWaiting sql.NullFloat64
	DaysToFirst sql.NullFloat64
}

func (cache Cache) UpsertRepoPRFromGH(repo string, pr *github.PullRequest) error {
	stmt, err := cache.DB.Prepare(`
		INSERT OR REPLACE INTO prs (repo, number, title, user, state, milestone, merged, merger, created, closed ) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for pr %d: %w", pr.GetNumber(), err)
	}

	_, err = stmt.Exec(
		repo,
		strconv.Itoa(pr.GetNumber()),
		pr.GetTitle(),
		pr.User.GetLogin(),
		pr.GetState(),
		pr.GetMilestone().GetTitle(),
		strconv.FormatBool(pr.GetMerged()),
		pr.MergedBy.GetLogin(),
		pr.GetCreatedAt(),
		pr.GetClosedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert pr %s#%d: %w", repo, pr.GetNumber(), err)
	}
	stmt.Close()

	return nil
}

func (cache Cache) UpsertPRStats(repo string, number int, daysOpen, daysWaiting, daysToFirst float64) error {
	stmt, err := cache.DB.Prepare(`
		UPDATE prs 
		SET daysopen = ?,
		    dayswaiting = ?,
		    daystofirst = ?
		WHERE
		    repo=? AND
			number=?;
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert stats statement for pr %d: %w", number, err)
	}

	if _, err = stmt.Exec(daysOpen, daysWaiting, daysToFirst, repo, number); err != nil {
		return fmt.Errorf("failed to insert stats statement for pr %s#%d: %w", repo, number, err)
	}
	stmt.Close()

	return nil
}

func (cache Cache) QueryForPRs(qfmt string, a ...any) (*[]PR, error) {
	q := fmt.Sprintf(qfmt, a...)

	rows, err := cache.DB.Query(q)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query '%s': %w", q, err)
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}

	prs := make([]PR, 0)

	for rows.Next() {
		pr := PR{}
		err := rows.Scan(
			&pr.Repo,
			&pr.Number,
			&pr.Title,
			&pr.User,
			&pr.State,
			&pr.Milestone,
			&pr.Merged,
			&pr.Merger,
			&pr.Created,
			&pr.Closed,
			&pr.DaysOpen,
			&pr.DaysWaiting,
			&pr.DaysToFirst,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pr: %w", err)
		}

		prs = append(prs, pr)
	}

	return &prs, nil
}

func (cache Cache) GetPR(repo string, number int) (*PR, error) {
	prs, err := cache.QueryForPRs(`
	SELECT %s 
	FROM prs 
	WHERE 
	    repo = '%s' AND
	    number='%d'
	`, ColumnsPR, repo, number)

	if err != nil {
		return nil, fmt.Errorf("failed to query for pr %d: %w", number, err)
	}

	if len(*prs) != 1 {
		return nil, fmt.Errorf("didn't find a single pr: %d", len(*prs))
	}

	pr := (*prs)[0]
	return &pr, nil
}

func (cache Cache) GetAllRepoPRs(repos []string) (*[]PR, error) {
	repoClause := ""
	if len(repos) > 0 {
		repoClause = " WHERE repo in ('" + strings.Join(repos, "', '") + "')"
	}

	return cache.QueryForPRs(`
		SELECT %s FROM prs %s
	`, ColumnsPR, repoClause)
}

func (cache Cache) GetRepoPRsWithState(repos []string, state string) (*[]PR, error) {
	repoClause := ""
	if len(repos) > 0 {
		repoClause = " AND repo in ('" + strings.Join(repos, "', '") + "')"
	}

	return cache.QueryForPRs(`
		SELECT %s FROM prs 
		WHERE
		    state='%s'
			%s
		`, ColumnsPR, state, repoClause)
}

func (cache Cache) GetRepoPRsCreatedForDateRange(repos []string, from, to time.Time) (*[]PR, error) {
	repoClause := ""
	if len(repos) > 0 {
		repoClause = " AND repo in ('" + strings.Join(repos, "', '") + "')"
	}

	return cache.QueryForPRs(`
		SELECT %s FROM prs
		WHERE 
		    created BETWEEN '%s' AND '%s'
			%s
	`, ColumnsPR, from.Format("2006-01-02"), to.Format("2006-01-02"), repoClause)
}

func (cache Cache) GetRepoPRsOpenForDateRange(repos []string, from, to time.Time) (*[]PR, error) {
	repoClause := ""
	if len(repos) > 0 {
		repoClause = " AND repo in ('" + strings.Join(repos, "', '") + "')"
	}

	return cache.QueryForPRs(fmt.Sprintf(`
		SELECT %[1]s  FROM prs
		WHERE
		    (created BETWEEN '%[2]s' AND '%[3]s' OR 
		    closed BETWEEN '%[2]s' AND '%[3]s')
		    %[4]s
	`, ColumnsPR, from.Format("2006-01-02"), to.Format("2006-01-02"), repoClause))
}
