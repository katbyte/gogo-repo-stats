package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/katbyte/gogo-repo-stats/lib/cache"
	"github.com/katbyte/gogo-repo-stats/lib/gh"
)

func GraphMultiRepoTotalPRsDaily(cache *cache.Cache, outPath string, from, to time.Time, repos []string) error {
	f := GetFlags() // todo out path ends up in flags

	// previous totals todo ???? these would be the PRs from before the start date

	var xAxis []string
	totals := make(map[string]int)
	lineData := make(map[string][]opts.LineData)
	for _, repo := range repos {
		totals[repo] = 0
		lineData[repo] = []opts.LineData{}
	}

	var csvdata [][]string
	csvdata = append(csvdata, append([]string{"date"}, repos...))

	var shortRepos []string
	for _, repo := range repos {
		shortRepos = append(shortRepos, gh.RepoShortName(repo))
	}

	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)

		dayEnd := dayStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
		if dayEnd.After(to) {
			dayEnd = to
		}

		// add day to x axis
		xAxis = append(xAxis, dayStart.Format("2006-01-02"))

		csvLine := []string{dayStart.Format("2006-01-02")}

		// for each repo get stats for the day and add to totals
		for _, repo := range repos {
			stats, err := cache.CalculateRepoPRStatsForDateRange(dayStart, dayEnd, []string{repo}, f.Authors)
			if err != nil {
				return fmt.Errorf("failed to query stats: %w", err)
			}

			totals[repo] += stats.Total
			lineData[repo] = append(lineData[repo], opts.LineData{Value: totals[repo]})

			csvLine = append(csvLine, strconv.Itoa(totals[repo]))
		}

		csvdata = append(csvdata, csvLine)
	}

	// write raw csvdata
	file, err := os.Create(outPath + "/daily-prs-total-by-repo.csv")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	csv := csv.NewWriter(file)
	defer csv.Flush()

	for _, r := range csvdata {
		err := csv.Write(r)
		if err != nil {
			panic(err)
		}
	}

	// render graph
	graph := charts.NewLine()
	graph.SetGlobalOptions(
		// charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Total PRs (daily)",
			Subtitle: "By Repo: " + strings.Join(shortRepos, ", "),
			Left:     "center", // nolint:misspell
		}),

		charts.WithXAxisOpts(opts.XAxis{
			Name: "Date",
			// AxisLabel: &opts.AxisLabel{Show: true, Formatter: "{value} x-unit"},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "PRs",
			// AxisLabel: &opts.AxisLabel{Show: true, Formatter: "{value} x-unit"},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1500px",
			Height: "750px",
		}),
		charts.WithColorsOpts(opts.Colors{"#2E4555", "#62A0A8", "#C13530"}),
		charts.WithToolboxOpts(opts.Toolbox{Show: true}),
		charts.WithLegendOpts(opts.Legend{
			Show: true,
			Top:  "bottom",
			Left: "center", // nolint:misspell
		}),
	)

	// Put csvdata into instance
	g := graph.SetXAxis(xAxis)
	for _, r := range repos {
		g = g.AddSeries(gh.RepoShortName(r), lineData[r])
	}
	g.SetSeriesOptions(charts.WithAreaStyleOpts(opts.AreaStyle{
		Opacity: 0.7,
	}),
		charts.WithLineChartOpts(opts.LineChart{
			Stack: "prs",
		}))

	// Where the magic happens
	file, err = os.Create(outPath + "/daily-prs-total-by-repo.html")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = graph.Render(file)
	if err != nil {
		return fmt.Errorf("failed to render graph graph: %w", err)
	}

	return nil
}

func GraphMultiRepoOpenPRsDaily(c *cache.Cache, outPath string, from, to time.Time, repos []string) error {
	// f := GetFlags() // todo out path ends up in flags

	var shortRepos []string
	for _, repo := range repos {
		shortRepos = append(shortRepos, gh.RepoShortName(repo))
	}

	// populate dates
	dates := map[string]map[string]DayStats{}
	for day := from; day.Before(to.AddDate(0, 0, 1)); day = day.AddDate(0, 0, 1) {
		k := day.Format("2006-01-02")
		dates[k] = map[string]DayStats{}
		for _, r := range repos {
			dates[k][r] = DayStats{}
		}
	}

	// get all prs for range
	prs, err := c.GetRepoPRsOpenForDateRange(repos, from, to)
	if err != nil {
		return fmt.Errorf("getting PRs: %w", err)
	}

	// for each pr in range
	for _, pr := range *prs {
		opened := time.Date(pr.Created.Year(), pr.Created.Month(), pr.Created.Day(), 0, 0, 0, 0, time.UTC)

		closed := pr.Closed
		if pr.State == "open" {
			closed = time.Now()
		}
		closed = time.Date(closed.Year(), closed.Month(), closed.Day(), 0, 0, 0, 0, time.UTC)

		// figure out timeline of events that matter
		// array of times -> "state" it is now in: waiting, approved, blocked
		events, err := c.GetEventsForPR(pr.Repo, pr.Number)
		if err != nil {
			return fmt.Errorf("getting events for PR %d: %w", pr.Number, err)
		}

		eventMap := map[time.Time]cache.Event{}
		var eventDates []time.Time
		for _, e := range events {
			eventDates = append(eventDates, e.Date)
			eventMap[e.Date] = e
		}

		// for each day from open to closed (or now) count this PR using the above array to figure out its "state"
		// by playing back events to "set the state" until the events

		state := "open"
		daysWaiting := 0
		eventIndex := 0
		for day := opened; ; day = day.AddDate(0, 0, 1) {
			if day.Before(from) {
				continue
			}

			k := day.Format("2006-01-02")

			d := dates[k][pr.Repo]
			d.Total++

			//
			for ; eventIndex < len(eventDates) && eventDates[eventIndex].Before(day.AddDate(0, 0, 1)); eventIndex++ {
				e := eventMap[eventDates[eventIndex]]

				// nolint: gocritic
				if e.Event == "milestoned" && e.Milestone == "Blocked" {
					state = "blocked"
				} else if e.Event == "milestoned" && e.Milestone != "Blocked" && state == "blocked" {
					state = "waiting"
				} else if e.Event == "labeled" && e.Label == "waiting-response" { // nolint:misspell
					state = "open"
				} else if e.Event == "unlabeled" && e.Label == "waiting-response" {
					state = "waiting"
				}

				if state != "waiting" {
					daysWaiting = 0
				}
			}

			switch state {
			case "waiting":
				daysWaiting++
				if daysWaiting > 14 {
					d.WaitingOver++
				} else {
					d.Waiting++
				}
			case "blocked":
				d.Blocked++
			case "approved":
				d.Approved++
			default:
				d.Open++
			}

			if day.After(to) {
				break
			}

			dates[k][pr.Repo] = d

			// check is here so PRs open for less than 1 day are counted
			if !day.Before(closed) {
				break
			}

		}
	}

	var xAxis []string
	lineData := make(map[string][]opts.LineData)
	for _, repo := range repos {
		lineData[repo] = []opts.LineData{}
	}

	data := [][]string{append([]string{"date"}, repos...)}
	for _, r := range shortRepos {
		data[0] = append(data[0], r)
	}

	days := make([]string, 0, len(dates))
	for day := range dates {
		days = append(days, day)
	}
	sort.Strings(days)

	for _, date := range days {
		day := dates[date]
		dayData := []string{date}
		xAxis = append(xAxis, date)

		for _, r := range repos {
			dayData = append(dayData, strconv.Itoa(day[r].Total))
			lineData[r] = append(lineData[r], opts.LineData{Value: day[r].Total})
		}
		data = append(data, dayData)
	}

	// write raw data
	file, err := os.Create(outPath + "/daily-prs-open-by-repo.csv")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	csv := csv.NewWriter(file)
	defer csv.Flush()

	for _, r := range data {
		err := csv.Write(r)
		if err != nil {
			panic(err)
		}
	}

	// render graph
	graph := charts.NewLine()
	graph.SetGlobalOptions(
		// charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    "PRs Open (daily)",
			Subtitle: "By Project: " + strings.Join(shortRepos, ","),
			Left:     "center", // nolint:misspell
		}),

		charts.WithXAxisOpts(opts.XAxis{
			Name: "Date",
			// AxisLabel: &opts.AxisLabel{Show: true, Formatter: "{value} x-unit"},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "PRs",
			// AxisLabel: &opts.AxisLabel{Show: true, Formatter: "{value} x-unit"},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1500px",
			Height: "750px",
		}),
		charts.WithColorsOpts(opts.Colors{"#C13530", "#2E4555", "#62A0A8", "#8C9BD4", "#AAD2E6"}),
		charts.WithToolboxOpts(opts.Toolbox{Show: true}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Trigger:   "axis",
			TriggerOn: "mousemove",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: true,
			Top:  "bottom",
			Left: "center", // nolint:misspell
		}),
	)

	// Put data into instance
	g := graph.SetXAxis(xAxis)
	for _, r := range repos {
		g = g.AddSeries(gh.RepoShortName(r), lineData[r])
	}
	g.SetSeriesOptions(charts.WithAreaStyleOpts(opts.AreaStyle{
		Opacity: 0.7,
	}),
		charts.WithLineChartOpts(opts.LineChart{
			Stack: "prs",
		}))

	// Where the magic happens
	file, err = os.Create(outPath + "/daily-prs-open-by-repo.html")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = graph.Render(file)
	if err != nil {
		return fmt.Errorf("failed to render graph graph: %w", err)
	}

	return nil
}
