package cache

import (
	"fmt"
	"time"

	"github.com/google/go-github/v45/github"
)

// for now all we care about is events. pr events with automation is how we can do an easy "time waiting"
//    open -> Blocked/Waiting For Response -> demilstoned/unlabled
// this is more reliable than just looking at reviews as waiting for response can be added without a review via a comment left (same for its removal)
// therefore it is the best proxy we have for "how long has this PR been waiting"

// TODO - table per event type?

type Event struct {
	Repo  string
	PR    int
	Date  time.Time
	Event string
	User  string

	State     string
	Label     string
	Milestone string
	Body      string

	URL string
}

func (cache Cache) UpsertEvent(repo string, pr int, event *github.Timeline) error {
	stmt, err := cache.DB.Prepare("INSERT OR REPLACE INTO events (repo, pr, date, event, user, state, label, milestone, body, url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for event %d/%s: %w", pr, event.GetURL(), err)
	}

	// get user - it is either User/Actor
	u := ""
	if event.Actor != nil && event.User != nil {
		if event.GetActor() == event.GetUser() {
			return fmt.Errorf("both actor and user exist and differ  %d/%s: %s != %s", pr, event.GetURL(), event.GetActor(), event.GetUser())
		}
	}

	if event.Actor != nil {
		u = event.Actor.GetLogin()
	}
	if event.User != nil {
		u = event.User.GetLogin()
	}

	// get correct date - createdAt opr SubmittedAt
	if event.CreatedAt != nil && event.SubmittedAt != nil {
		if event.GetActor() == event.GetUser() {
			return fmt.Errorf("both actor and user exist and differ %d:  %s  == %s", pr, event.GetActor(), event.GetUser())
		}
	}

	var t time.Time
	if event.CreatedAt != nil {
		t = event.GetCreatedAt()
	}
	if event.SubmittedAt != nil {
		t = event.GetSubmittedAt()
	}

	_, err = stmt.Exec(
		repo,
		pr,
		t,
		event.GetEvent(),
		u,

		event.GetState(),
		event.GetLabel().GetName(),
		event.GetMilestone().GetTitle(),
		event.GetBody(),

		event.GetURL(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert event %d/%s: %w", pr, event.GetURL(), err)
	}
	stmt.Close()

	return nil
}

func (cache Cache) GetEventsForPR(repo string, number int) ([]Event, error) {
	rows, err := cache.DB.Query(fmt.Sprintf(`
		SELECT repo, pr, date, event, user, state, label, milestone, body, url 
		FROM events 
		WHERE
			repo='%s' AND
		    pr='%d' ORDER BY date
	`, repo, number))
	if err != nil {
		return nil, fmt.Errorf("failed to query events for pr %d: %w", number, err)
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to get events for pr %d: %w", number, err)
	}

	var events []Event
	for rows.Next() {
		e := Event{}
		err = rows.Scan(
			&e.Repo,
			&e.PR,
			&e.Date,
			&e.Event,
			&e.User,
			&e.State,
			&e.Label,
			&e.Milestone,
			&e.Body,
			&e.URL,
		)

		events = append(events, e)
		if err != nil {
			return nil, fmt.Errorf("failed to scan events for pr %d: %w", number, err)
		}
	}

	return events, nil
}
