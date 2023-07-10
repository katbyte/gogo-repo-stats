package cache

import (
	"fmt"
	"math"
	"time"

	c "github.com/gookit/color" // nolint:misspell
	"github.com/katbyte/gogo-repo-stats/lib/clog"
)

func (cache Cache) ComputeAndUpdatePRStats(repo string, number int) (open, waiting, tofirst *float64, err error) {
	// check cache
	pr, err := cache.GetPR(repo, number)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get pr %d: %w", number, err)
	}

	// get events
	events, err := cache.GetEventsForPR(repo, number)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getting events for PR %d: %w", number, err)
	}
	clog.Log.Debugf(c.Sprintf("   with <magenta>%d</> events: ", len(events)))
	for _, e := range events {
		clog.Log.Debugf(c.Sprintf("%s, ", e.Event))
	}
	clog.Log.Debugf(c.Sprintf("\n"))

	// var
	var duration time.Duration
	opened := pr.Created
	for _, e := range events {
		if e.Event == "closed" {
			d := e.Date.Sub(opened)
			clog.Log.Debugf(c.Sprintf("      closed @ %s after %00.00f days \n", e.Date.Format("2006-01-02"), d.Hours()/24))
			duration += d
		}

		if e.Event == "reopened" {
			opened = e.Date
			clog.Log.Debugf(c.Sprintf("      opened @ %s\n", e.Date.Format("2006-01-02")))
		}

		if e.Event == "merged" {
			d := e.Date.Sub(opened)
			clog.Log.Debugf(c.Sprintf("      merged @ %s\n", e.Date.Format("2006-01-02")))
			duration += d
			break
		}

		// clog.Log.Debugf(c.Sprintf("  %s\n", e.Event)
	}

	// catch prs without above events
	if duration == 0 {
		// if closed uses closed, if open used open
		if pr.State == "closed" {
			duration += pr.Closed.Sub(opened)
		} else {
			duration += time.Since(opened)
		}
	}
	daysOpen := duration.Hours() / 24

	// waiting := 0
	// calculate days waiting
	duration = time.Duration(0)
	opened = pr.Created
	for _, e := range events {
		if e.Event == "labeled" && e.Label == "waiting-response" { // nolint:misspell
			d := e.Date.Sub(opened)
			clog.Log.Debugf(c.Sprintf("      labeled waiting-response @ %s after %.2f days \n", e.Date.Format("2006-01-02"), d.Hours()/24)) // nolint:misspell
			duration += d
		}

		if e.Event == "unlabeled" && e.Label == "waiting-response" {
			opened = e.Date
			clog.Log.Debugf(c.Sprintf("      unlabeled waiting-response @ %s\n", e.Date.Format("2006-01-02")))
		}
	}
	if duration == 0 {
		// if closed uses closed, if open used open
		if pr.State == "closed" {
			duration += pr.Closed.Sub(opened)
		} else {
			duration += time.Since(opened)
		}
	}
	daysWaiting := duration.Hours() / 24

	// first := 0
	// calculate days to first action:
	// - milestone blocked
	// - label waiting for response
	// - merged/closed
	duration = time.Duration(0)
	for _, e := range events {
		if e.Event == "milestoned" && e.Milestone == "Blocked" {
			duration = e.Date.Sub(pr.Created)
			clog.Log.Debugf(c.Sprintf("      first: milestoned @ %s\n", e.Date.Format("2006-01-02")))
			break
		}

		if e.Event == "labeled" && e.Label == "Waiting-for-Response" { // nolint:misspell
			duration = e.Date.Sub(pr.Created)
			clog.Log.Debugf(c.Sprintf("      first: labeled @ %s\n", e.Date.Format("2006-01-02"))) // nolint:misspell
			break
		}

		if e.Event == "reviewed" || e.Event == "merged" {
			duration = e.Date.Sub(pr.Created)
			clog.Log.Debugf(c.Sprintf("      first: reviewed @ %s\n", e.Date.Format("2006-01-02")))
			break
		}
	}
	if duration == 0 {
		// if closed uses closed, if open used open
		if pr.State == "closed" {
			duration += pr.Closed.Sub(opened)
		} else {
			duration += time.Since(opened)
		}
	}
	daysToFirst := duration.Hours() / 24

	clog.Log.Debugf(c.Sprintf("  days open: <green>%.2f</> waiting: <green>%.2f</> to first: <green>%.2f</> \n", daysOpen, daysWaiting, daysToFirst))

	// update row in DB:
	err = cache.UpsertPRStats(repo, pr.Number, math.Floor(daysOpen*100)/100, math.Floor(daysWaiting*100)/100, math.Floor(daysToFirst*100)/100)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("update cache pr stats %d: %w", pr.Number, err)
	}

	return &daysOpen, &daysWaiting, &daysToFirst, nil
}
