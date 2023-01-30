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
	c "github.com/gookit/color"
	"github.com/katbyte/gogo-repo-stats/lib/cache"
	"github.com/spf13/cobra"
)

func CmdGraphs(_ *cobra.Command, args []string) error {
	var err error
	f := GetFlags()

	// todo add to flags
	outPath := "graphs"

	// ensure path exists
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		err := os.MkdirAll(outPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}
	}

	// default to past year
	from := time.Now().AddDate(-1, 0, 0)
	from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	to := time.Now()

	aragc := len(args)
	if aragc > 1 {
		from, err = time.Parse("2006-01", args[0])
		if err != nil {
			return fmt.Errorf("failed to parse time %s : %w", args[0], err)
		}

		if aragc == 2 {
			to, err = time.Parse("2006-01", args[1])
			if err != nil {
				return fmt.Errorf("failed to parse time %s : %w", args[1], err)
			}
		}
	}

	// open cache
	cache, err := cache.Open(f.CachePath)
	if err != nil {
		return fmt.Errorf("opening cache %s: %w", f.CachePath, err)
	}
	defer cache.DB.Close()

	c.Printf("Generating graphs for PRs from <white>%s</> to <white>%s</>...\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
	if len(f.Authors) > 0 {
		c.Printf("  for authors: <green>%s</>\n", strings.Join(f.Authors, "</>, <green>"))
	}

	// todo collect these in an array and then do each one in a loop
	err = GraphsDailyOpenedPRs(cache, outPath, from, to)
	if err != nil {
		return fmt.Errorf("failed to generate daily pr graphs path: %w", err)
	}

	err = GraphDailyTotalPRs(cache, outPath, from, to)
	if err != nil {
		return fmt.Errorf("failed to generate daily total pr graphs path: %w", err)
	}

	err = GraphDailyOpenPRs(cache, outPath, from, to)
	if err != nil {
		return fmt.Errorf("failed to generate daily open pr graphs path: %w", err)
	}

	return nil
}

func GraphsDailyOpenedPRs(cache *cache.Cache, outPath string, from, to time.Time) error {
	f := GetFlags() // todo out path ends up in flags

	var xAxis []string
	var merged, closed, open []opts.BarData

	// get data for each day?
	data := [][]string{{"date", "total", "merged", "closed", "open"}}
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)

		dayEnd := dayStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
		if dayEnd.After(to) {
			dayEnd = to
		}

		stats, err := cache.CalculatePRStatsForDateRange(dayStart, dayEnd, f.Authors)
		if err != nil {
			return fmt.Errorf("failed to query stats: %w", err)
		}

		xAxis = append(xAxis, dayStart.Format("2006-01-02"))
		open = append(open, opts.BarData{Value: stats.Open})
		closed = append(closed, opts.BarData{Value: stats.Closed - stats.Merged})
		merged = append(merged, opts.BarData{Value: stats.Merged})

		data = append(data,
			[]string{dayStart.Format("2006-01-02"),
				strconv.Itoa(stats.Total),
				strconv.Itoa(stats.Merged),
				strconv.Itoa(stats.Closed - stats.Merged),
				strconv.Itoa(stats.Open),
			})
	}

	// write raw data
	file, err := os.Create(outPath + "/daily-prs-opened.csv")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	csv := csv.NewWriter(file)
	defer csv.Flush()

	for _, r := range data {
		err := csv.Write(r)
		if err != nil {
			return fmt.Errorf("writing to csv vile file: %w", err)
		}
	}

	// render graph
	graph := charts.NewBar()
	graph.SetGlobalOptions(
		// charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    "PRs Opened (daily)",
			Subtitle: "By Current Status: merged, closed, open",
			Left:     "center"}),

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
			Left: "center",
		}),
	)

	// Put data into instance
	graph.SetXAxis(xAxis).
		AddSeries("Merged", merged).
		AddSeries("Closed", closed).
		AddSeries("Open", open).
		SetSeriesOptions(charts.WithBarChartOpts(opts.BarChart{
			Stack: "total",
		}))

	// Where the magic happens
	file, err = os.Create(outPath + "/daily-prs-opened.html")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = graph.Render(file)
	if err != nil {
		return fmt.Errorf("failed to render graph chart: %w", err)
	}

	return nil
}

func GraphDailyTotalPRs(cache *cache.Cache, outPath string, from, to time.Time) error {
	f := GetFlags() // todo out path ends up in flags

	var totalCount, mergedCount, closedCount, openCount int

	// previous totals
	/*
		stats, err := cache.CalculatePRStatsForDateRange(from.AddDate(-47, 0, 0), from.Add(-time.Nanosecond), f.Authors)
		if err != nil {
			return fmt.Errorf("failed to query stats: %w", err)
		}

		totalCount = stats.Total
		mergedCount = stats.Merged
		closedCount = stats.Closed - stats.Merged
		openCount = stats.Open

	*/

	var xAxis []string
	var merged, closed, open []opts.LineData

	var data [][]string
	data = append(data, []string{"date", "total", "closed", "merged", "open"})
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)

		dayEnd := dayStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
		if dayEnd.After(to) {
			dayEnd = to
		}

		stats, err := cache.CalculatePRStatsForDateRange(dayStart, dayEnd, f.Authors)
		if err != nil {
			return fmt.Errorf("failed to query stats: %w", err)
		}

		totalCount += stats.Total
		mergedCount += stats.Merged
		closedCount += stats.Closed - stats.Merged
		openCount += stats.Open

		xAxis = append(xAxis, dayStart.Format("2006-01-02"))
		open = append(open, opts.LineData{Value: openCount})
		closed = append(closed, opts.LineData{Value: closedCount})
		merged = append(merged, opts.LineData{Value: mergedCount})

		data = append(data,
			[]string{dayStart.Format("2006-01-02"),
				strconv.Itoa(totalCount),
				strconv.Itoa(mergedCount),
				strconv.Itoa(closedCount),
				strconv.Itoa(openCount),
			})
	}

	// write raw data
	file, err := os.Create(outPath + "/daily-prs-total.csv")
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
			Title:    "Total PRs (daily)",
			Subtitle: "By Current Status: merged, closed, open",
			Left:     "center"}),

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
			Left: "center",
		}),
	)

	// Put data into instance
	graph.SetXAxis(xAxis).
		AddSeries("Merged", merged).
		AddSeries("Closed", closed).
		AddSeries("Open", open).
		SetSeriesOptions(charts.WithAreaStyleOpts(opts.AreaStyle{
			Opacity: 0.7,
		}),
			charts.WithLineChartOpts(opts.LineChart{
				Stack: "prs",
			}))

	// Where the magic happens
	file, err = os.Create(outPath + "/daily-prs-total.html")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = graph.Render(file)
	if err != nil {
		return fmt.Errorf("failed to render graph graph: %w", err)
	}

	return nil
}

type DayStats struct {
	Total       int
	Open        int
	Blocked     int
	Waiting     int
	WaitingOver int
	Approved    int
}

func GraphDailyOpenPRs(c *cache.Cache, outPath string, from, to time.Time) error {
	// f := GetFlags() // todo out path ends up in flags

	// populate dates
	dates := map[string]DayStats{}
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		k := day.Format("2006-01-02")
		dates[k] = DayStats{}
	}

	// TODO need to calculate the dates BEFORE the start to get a baseline?
	// todo just calculate ALL prs and then do a cutoff?

	// get all prs for range
	prs, err := c.GetPRsOpenForDateRange(from, to)
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
		events, err := c.GetEventsForPR(pr.Number)
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
			if day.Before(from.AddDate(0, 0, -1)) {
				continue
			}

			k := day.Format("2006-01-02")

			d := dates[k]
			d.Total++

			//
			for ; eventIndex < len(eventDates) && eventDates[eventIndex].Before(day.AddDate(0, 0, 1)); eventIndex++ {
				e := eventMap[eventDates[eventIndex]]

				if e.Event == "milestoned" && e.Milestone == "Blocked" {
					state = "blocked"
				} else if e.Event == "milestoned" && e.Milestone != "Blocked" && state == "blocked" {
					state = "waiting"
				} else if e.Event == "labeled" && e.Label == "waiting-response" {
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

			dates[k] = d

			// check is here so PRs open for less than 1 day are counted
			if !day.Before(closed) {
				break
			}

			if day.After(to) {
				break
			}
		}

	}

	var xAxis []string
	var lineOpen, lineBlocked, lineWaiting, lineWaitingOver []opts.LineData

	data := [][]string{{"date", "total", "open", "blocked", "waiting", "waiting-over", "approved"}}

	days := make([]string, 0, len(dates))
	for day := range dates {
		days = append(days, day)
	}
	sort.Strings(days)

	for _, date := range days {
		day := dates[date]
		data = append(data, []string{date, strconv.Itoa(day.Total), strconv.Itoa(day.Open), strconv.Itoa(day.Blocked), strconv.Itoa(day.Waiting), strconv.Itoa(day.WaitingOver), strconv.Itoa(day.Approved)})

		xAxis = append(xAxis, date)
		// totalLine = append(totalLine, opts.LineData{Value: day.Total})
		lineOpen = append(lineOpen, opts.LineData{Value: day.Open})
		lineBlocked = append(lineBlocked, opts.LineData{Value: day.Blocked})
		lineWaiting = append(lineWaiting, opts.LineData{Value: day.Waiting})
		lineWaitingOver = append(lineWaitingOver, opts.LineData{Value: day.WaitingOver})

	}

	// write raw data
	file, err := os.Create(outPath + "/daily-prs-open.csv")
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
			Subtitle: "By State: open, waiting, waiting (over 14 days), blocked, approved",
			Left:     "center"}),

		charts.WithXAxisOpts(opts.XAxis{
			Name:        "Date",
			MinInterval: 20,
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
		charts.WithColorsOpts(opts.Colors{"#C13530", "#2E4555", "#62A0A8"}),
		charts.WithToolboxOpts(opts.Toolbox{Show: true}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Trigger:   "axis",
			TriggerOn: "mousemove",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: true,
			Top:  "bottom",
			Left: "center",
		}),
	)

	// Put data into instance
	graph.SetXAxis(xAxis).
		AddSeries("Blocked", lineBlocked).
		AddSeries("Waiting Over 14", lineWaitingOver).
		AddSeries("Waiting", lineWaiting).
		AddSeries("Open", lineOpen).
		SetSeriesOptions(charts.WithAreaStyleOpts(opts.AreaStyle{
			Opacity: 0.7,
		}),
			charts.WithLineChartOpts(opts.LineChart{
				Stack: "prs",
			}))

	// Where the magic happens
	file, err = os.Create(outPath + "/daily-prs-open.html")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = graph.Render(file)
	if err != nil {
		return fmt.Errorf("failed to render graph graph: %w", err)
	}

	return nil
}
