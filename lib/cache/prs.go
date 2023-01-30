package cache

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/go-github/v45/github"
)

// TODO switch to an ORM ?

const Columns = "number, title, user, state, milestone, merged, merger, created, closed"

type PR struct {
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

func (cache Cache) UpsertPRFromGH(pr *github.PullRequest) error {
	stmt, err := cache.DB.Prepare("INSERT OR REPLACE INTO prs (number, title, user, state, milestone, merged, merger, created, closed ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for pr %d: %w", pr.GetNumber(), err)
	}

	_, err = stmt.Exec(
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
		return fmt.Errorf("failed to insert pr %d: %w", pr.GetNumber(), err)
	}
	stmt.Close()

	return nil
}

func (cache Cache) UpsertPRStats(number int, daysOpen, daysWaiting, daysToFirst float64) error {
	stmt, err := cache.DB.Prepare(fmt.Sprintf(`
		UPDATE prs 
		SET daysopen = ?,
		    dayswaiting = ?,
		    daystofirst = ?
		WHERE
			number=?;
	`))
	if err != nil {
		return fmt.Errorf("failed to prepare insert stats statement for pr %d: %w", number, err)
	}

	if _, err = stmt.Exec(daysOpen, daysWaiting, daysToFirst, number); err != nil {
		return fmt.Errorf("failed to insert stats statement for pr %d: %w", number, err)
	}
	stmt.Close()

	return nil
}

func (cache Cache) QueryForPRs(q string) (*[]PR, error) {
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

func (cache Cache) GetPR(number int) (*PR, error) {
	prs, err := cache.QueryForPRs("SELECT number, title, user, state, milestone, merged, merger, created, closed, daysopen, dayswaiting, daystofirst FROM prs WHERE number='" + strconv.Itoa(number) + "'")
	if err != nil {
		return nil, fmt.Errorf("failed to query for pr %d: %w", number, err)
	}

	if len(*prs) != 1 {
		return nil, fmt.Errorf("didn't find a single pr: %d", len(*prs))
	}

	pr := (*prs)[0]
	return &pr, nil
}

func (cache Cache) GetAllPRs() (*[]PR, error) {
	return cache.QueryForPRs("SELECT number, title, user, state, milestone, merged, merger, created, closed, daysopen, dayswaiting, daystofirst FROM prs")
}

func (cache Cache) GetPRsWithState(state string) (*[]PR, error) {
	return cache.QueryForPRs("SELECT number, title, user, state, milestone, merged, merger, created, closed, daysopen, dayswaiting, daystofirst FROM prs WHERE state='" + state + "'")
}

func (cache Cache) GetPRsCreatedForDateRange(from, to time.Time) (*[]PR, error) {
	return cache.QueryForPRs(fmt.Sprintf(`
		SELECT number, title, user, state, milestone, merged, merger, created, closed, daysopen, dayswaiting, daystofirst 
		FROM prs
		WHERE created BETWEEN '%s' AND '%s'
	`, from.Format("2006-01-02"), to.Format("2006-01-02")))
}

func (cache Cache) GetPRsOpenForDateRange(from, to time.Time) (*[]PR, error) {
	return cache.QueryForPRs(fmt.Sprintf(`
		SELECT number, title, user, state, milestone, merged, merger, created, closed, daysopen, dayswaiting, daystofirst 
		FROM prs
		WHERE created BETWEEN '%[1]s' AND '%[2]s' or closed BETWEEN '%[1]s' AND '%[2]s'
	`, from.Format("2006-01-02"), to.Format("2006-01-02")))
}
