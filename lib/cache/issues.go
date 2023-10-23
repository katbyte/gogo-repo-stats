package cache

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
)

const ColumnsIssues = "repo, number, title, user, state, milestone, labels, created, closed, daysopen"

type Issue struct {
	Repo      string
	Number    int
	Title     string
	User      string
	State     string // todo should we make this boolean "open" or 2 boolean so we have open/closed/merged ?
	Milestone string
	Labels    string
	Created   time.Time
	Closed    time.Time

	// calculated
	DaysOpen sql.NullFloat64
}

func (cache Cache) UpsertRepoIssueFromGH(repo string, issue *github.Issue) error {
	stmt, err := cache.DB.Prepare(`
		INSERT OR REPLACE INTO issues (repo, number, title, user, state, milestone, labels, created, closed ) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for issue %d: %w", issue.GetNumber(), err)
	}

	// get labels
	labels := make([]string, 0)
	for _, label := range issue.Labels {
		labels = append(labels, label.GetName())
	}

	_, err = stmt.Exec(
		repo,
		strconv.Itoa(issue.GetNumber()),
		issue.GetTitle(),
		issue.User.GetLogin(),
		issue.GetState(),
		issue.GetMilestone().GetTitle(),
		strings.Join(labels, ","),
		issue.GetCreatedAt(),
		issue.GetClosedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert issue %s#%d: %w", repo, issue.GetNumber(), err)
	}
	stmt.Close()

	return nil
}

func (cache Cache) UpsertIssueStats(repo string, number int, daysOpen, daysWaiting, daysToFirst float64) error {
	stmt, err := cache.DB.Prepare(`
		UPDATE issues 
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

func (cache Cache) QueryForIssues(qfmt string, a ...any) (*[]Issue, error) {
	q := fmt.Sprintf(qfmt, a...)

	rows, err := cache.DB.Query(q)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare issue query '%s': %w", q, err)
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to run issue query: %w", err)
	}

	issues := make([]Issue, 0)

	for rows.Next() {
		pr := Issue{}
		err := rows.Scan(
			&pr.Repo,
			&pr.Number,
			&pr.Title,
			&pr.User,
			&pr.State,
			&pr.Milestone,
			&pr.Labels,
			&pr.Created,
			&pr.Closed,
			&pr.DaysOpen,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pr: %w", err)
		}

		issues = append(issues, pr)
	}

	return &issues, nil
}

func (cache Cache) GetIssue(repo string, number int) (*Issue, error) {
	issues, err := cache.QueryForIssues(`
	SELECT %s 
	FROM issues 
	WHERE 
	    repo = '%s' AND
	    number='%d'
	`, ColumnsIssues, repo, number)

	if err != nil {
		return nil, fmt.Errorf("failed to query for issue %d: %w", number, err)
	}

	if len(*issues) != 1 {
		return nil, nil
	}

	issue := (*issues)[0]
	return &issue, nil
}

func (cache Cache) GetAllRepoIssues(repos []string) (*[]Issue, error) {
	repoClause := ""
	if len(repos) > 0 {
		repoClause = " WHERE repo in ('" + strings.Join(repos, "', '") + "')"
	}

	return cache.QueryForIssues(`
		SELECT %s FROM prs %s
	`, ColumnsIssues, repoClause)
}

func (cache Cache) GetRepoIssuesCreatedForDateRange(repos []string, from, to time.Time) (*[]Issue, error) {
	repoClause := ""
	if len(repos) > 0 {
		repoClause = " AND repo in ('" + strings.Join(repos, "', '") + "')"
	}

	return cache.QueryForIssues(`
		SELECT %s FROM prs
		WHERE 
		    created BETWEEN '%s' AND '%s'
			%s
	`, ColumnsIssues, from.Format("2006-01-02"), to.Format("2006-01-02"), repoClause)
}

func (cache Cache) GetRepoIssuesOpenForDateRange(repos []string, from, to time.Time) (*[]Issue, error) {
	repoClause := ""
	if len(repos) > 0 {
		repoClause = " AND repo in ('" + strings.Join(repos, "', '") + "')"
	}

	return cache.QueryForIssues(fmt.Sprintf(`
		SELECT %[1]s  FROM issues
		WHERE
		    (created BETWEEN '%[2]s' AND '%[3]s' OR 
		    closed BETWEEN '%[2]s' AND '%[3]s' OR
		    closed < '1977-7-7')
		    %[4]s
	`, ColumnsIssues, from.Format("2006-01-02"), to.Format("2006-01-02"), repoClause))
}
