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
	"github.com/katbyte/gogo-repo-stats/lib/gh"
)

func GraphsRepoOpenedPRsDaily(cache *cache.Cache, outPath string, from, to time.Time, repos []string) error {
	f := GetFlags() // todo out path ends up in flags

	c.Printf("    PRs opened daily..\n")
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

		stats, err := cache.CalculateRepoPRStatsForDateRange(dayStart, dayEnd, repos, f.Authors)
		if err != nil {
			return fmt.Errorf("failed to query stats: %w", err)
		}

		xAxis = append(xAxis, dayStart.Format("2006-01-02"))
		open = append(open, opts.BarData{Value: stats.Open})
		closed = append(closed, opts.BarData{Value: stats.Closed - stats.Merged})
		merged = append(merged, opts.BarData{Value: stats.Merged})

		data = append(data,
			[]string{
				dayStart.Format("2006-01-02"),
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

	var repoShortNames []string
	for _, r := range repos {
		repoShortNames = append(repoShortNames, gh.RepoShortName(r))
	}

	// render graph
	graph := charts.NewBar()
	graph.SetGlobalOptions(
		// charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    strings.Join(repoShortNames, ",") + " PRs Opened (daily)",
			Subtitle: "By Current Status: merged, closed, open",
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

func GraphRepoTotalPRsDaily(cache *cache.Cache, outPath string, from, to time.Time, repos []string) error {
	f := GetFlags() // todo out path ends up in flags

	var totalCount, mergedCount, closedCount, openCount int

	c.Printf("    Total PRs daily..\n")

	// previous totals todo ???? these would be the PRs from before the start date
	/*
		stats, err := cache.CalculateRepoPRStatsForDateRange(from.AddDate(-47, 0, 0), from.Add(-time.Nanosecond), f.Authors)
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

		stats, err := cache.CalculateRepoPRStatsForDateRange(dayStart, dayEnd, repos, f.Authors)
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
			[]string{
				dayStart.Format("2006-01-02"),
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

	var repoShortNames []string
	for _, r := range repos {
		repoShortNames = append(repoShortNames, gh.RepoShortName(r))
	}

	// render graph
	graph := charts.NewLine()
	graph.SetGlobalOptions(
		// charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    strings.Join(repoShortNames, ",") + " Total PRs (daily)",
			Subtitle: "By Current Status: merged, closed, open",
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

type DayStatsPRs struct {
	Date          time.Time
	Total         int
	Open          int
	Blocked       int
	Waiting       int
	WaitingOver   int
	Approved      int
	TrendSevenDay int
}

func GraphRepoOpenPRsDaily(theCache *cache.Cache, outPath string, from, to time.Time, repos []string) error {
	// f := GetFlags() // todo out path ends up in flags

	c.Printf("    PRs open daily..\n")

	// populate dates
	dates := map[string]DayStatsPRs{}
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		k := day.Format("2006-01-02")
		dates[k] = DayStatsPRs{
			Date: day,
		}
	}

	// get all prs for range
	prs, err := theCache.GetRepoPRsOpenForDateRange(repos, from, to)
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
		events, err := theCache.GetEventsFor(pr.Repo, pr.Number)
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

	// calculate 7 day average
	i := 1
	for day := from.AddDate(0, 0, -1); day.Before(to.AddDate(0, 0, 2)); day = day.AddDate(0, 0, 1) {
		k := day.Format("2006-01-02")

		if i < 7 {
			i += 1
		}

		trendTotal := 0
		for dayT := day.AddDate(0, 0, -(i - 1)); !dayT.After(day); dayT = dayT.AddDate(0, 0, 1) {
			d := dates[dayT.Format("2006-01-02")]
			trendTotal += d.Open + d.Blocked + d.Waiting + d.WaitingOver
		}

		d := dates[k]
		d.TrendSevenDay = trendTotal / i
		dates[k] = d
	}

	var xAxis []string
	var lineOpen, lineBlocked, lineWaiting, lineWaitingOver, lineTrendLine []opts.LineData

	data := [][]string{{"date", "total", "open", "blocked", "waiting", "waiting-over", "approved", "7 day trend"}}

	days := make([]string, 0, len(dates))
	for day := range dates {
		days = append(days, day)
	}
	sort.Strings(days)

	for _, date := range days {
		day := dates[date]
		data = append(data, []string{date, strconv.Itoa(day.Total), strconv.Itoa(day.Open), strconv.Itoa(day.Blocked), strconv.Itoa(day.Waiting), strconv.Itoa(day.WaitingOver), strconv.Itoa(day.Approved)})

		if day.Date.Before(from.AddDate(0, 0, -1)) {
			continue
		}

		if day.Date.After(to) {
			continue
		}

		xAxis = append(xAxis, date)
		// totalLine = append(totalLine, opts.LineData{Value: day.Total})
		lineOpen = append(lineOpen, opts.LineData{Value: day.Open})
		lineBlocked = append(lineBlocked, opts.LineData{Value: day.Blocked})
		lineWaiting = append(lineWaiting, opts.LineData{Value: day.Waiting})
		lineWaitingOver = append(lineWaitingOver, opts.LineData{Value: day.WaitingOver})
		lineTrendLine = append(lineTrendLine, opts.LineData{Value: day.TrendSevenDay})
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

	var repoShortNames []string
	for _, r := range repos {
		repoShortNames = append(repoShortNames, gh.RepoShortName(r))
	}

	// render graph
	graph := charts.NewLine()
	graph.SetGlobalOptions(
		// charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    strings.Join(repoShortNames, ",") + " PRs Open (daily)",
			Subtitle: "By State: open, waiting, waiting (over 14 days), blocked, approved",
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
		charts.WithColorsOpts(opts.Colors{"#C13530", "#2E4555", "#62A0A8", "#5470c6", "#000000"}),
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
	graph.SetXAxis(xAxis)

	prStackOps := []charts.SeriesOpts{
		charts.WithAreaStyleOpts(opts.AreaStyle{Opacity: 0.8}),
		charts.WithLineChartOpts(opts.LineChart{Stack: "prs"}),
		charts.WithLineStyleOpts(opts.LineStyle{Width: 1, Opacity: 0.9}),
	}

	graph.AddSeries("Blocked", lineBlocked)
	graph.AddSeries("Waiting Over 14", lineWaitingOver)
	graph.AddSeries("Waiting", lineWaiting)
	graph.AddSeries("Open", lineOpen).SetSeriesOptions(prStackOps...)

	graph.AddSeries("Total - 7 day avg", lineTrendLine)
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

type DayStatsPRsByAuthor struct {
	Date    time.Time
	Total   int
	ByGroup map[string]int
}

func GraphRepoOpenPRsByAuthorsDaily(theCache *cache.Cache, outPath string, from, to time.Time, repos []string) error {
	// f := GetFlags()

	// todo make args
	authorGroupings := map[string][]string{
		"hashicorp": {"katbyte", "tombuildsstuff", "mbfrahry", "jackofallops", "manicminer", "stephybun"},
		"microsoft": {"wodansson", "magodo", "mybayern1974", "xuzhang3", "wuxu92", "teowa", "ms-henglu", "lonegunmanb", "xiaxyi", "ms-zhenhua", "myc2h6o", "neil-yechenwei", "sinbai", "jiaweitao001", "liuwuliuyun", "ziyeqf"},
		"community": {"*"},
	}

	authorLookup := map[string]string{}
	for group, authors := range authorGroupings {
		for _, author := range authors {
			authorLookup[author] = group
		}
	}

	c.Printf("    PRs open daily (by Author)..\n")

	// populate dates
	dates := map[string]DayStatsPRsByAuthor{}
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		k := day.Format("2006-01-02")

		byGroup := map[string]int{}
		for group := range authorGroupings {
			byGroup[group] = 0
		}

		dates[k] = DayStatsPRsByAuthor{
			Date:    day,
			ByGroup: byGroup,
		}
	}

	// get all prs for range
	prs, err := theCache.GetRepoPRsOpenForDateRange(repos, from, to)
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

		group, ok := authorLookup[pr.User]
		if !ok {
			group = "community"
		}

		fmt.Printf("group: %s\n", group)
		for day := opened; ; day = day.AddDate(0, 0, 1) {
			if day.Before(from.AddDate(0, 0, -1)) {
				continue
			}

			k := day.Format("2006-01-02")

			d := dates[k]
			d.Total++
			d.ByGroup[group] = d.ByGroup[group] + 1
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

	lineData := map[string][]opts.LineData{}
	authroGroups := []string{}
	for group := range authorGroupings {
		lineData[group] = []opts.LineData{}
		authroGroups = append(authroGroups, group)
	}

	data := [][]string{{"date", "total"}}
	data[0] = append(data[0], authroGroups...)

	days := make([]string, 0, len(dates))
	for day := range dates {
		days = append(days, day)
	}
	sort.Strings(days)

	for _, date := range days {
		day := dates[date]
		data = append(data, []string{date, strconv.Itoa(day.Total)})

		for _, group := range authroGroups {
			data = append(data, []string{date, strconv.Itoa(day.ByGroup[group])})
			lineData[group] = append(lineData[group], opts.LineData{Value: day.ByGroup[group]})
		}

		if day.Date.Before(from.AddDate(0, 0, -1)) {
			continue
		}

		if day.Date.After(to) {
			continue
		}

		xAxis = append(xAxis, date)
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

	var repoShortNames []string
	for _, r := range repos {
		repoShortNames = append(repoShortNames, gh.RepoShortName(r))
	}

	// render graph
	graph := charts.NewLine()
	graph.SetGlobalOptions(
		// charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    strings.Join(repoShortNames, ",") + " PRs Open (daily)",
			Subtitle: "By State: open, waiting, waiting (over 14 days), blocked, approved",
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
		charts.WithColorsOpts(opts.Colors{"#C13530", "#2E4555", "#62A0A8", "#5470c6", "#000000"}),
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
	graph.SetXAxis(xAxis)

	prStackOps := []charts.SeriesOpts{
		charts.WithAreaStyleOpts(opts.AreaStyle{Opacity: 0.8}),
		charts.WithLineChartOpts(opts.LineChart{Stack: "prs"}),
		charts.WithLineStyleOpts(opts.LineStyle{Width: 1, Opacity: 0.9}),
	}

	var series *charts.Line
	for group, line := range lineData {
		series = graph.AddSeries(group, line)
	}
	series.SetSeriesOptions(prStackOps...)

	// Where the magic happens
	file, err = os.Create(outPath + "/daily-prs-open-by-authors.html")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = graph.Render(file)
	if err != nil {
		return fmt.Errorf("failed to render graph graph: %w", err)
	}

	return nil
}

type WeekStatsPRs struct {
	Total           int
	OpenDays        float64
	WaitingDays     float64
	DaysTillFirst   float64
	WaitingOverDays float64
}

/*
func GraphRepoWeeklyPrStats(repo string, c *cache.Cache, outPath string, from, to time.Time) error {
	// f := GetFlags() // todo out path ends up in flags

	// Adjust the start date to the first Sunday before or on the start date
	for from.Weekday() != time.Sunday {
		from = from.AddDate(0, 0, -1)
	}

	//setup csv
	data := [][]string{{"date", "total", "open", "blocked", "waiting", "waiting-over", "approved"}}

	// go week by week
	weeks := []string{}
	var daysOpen, daysWaiting, daysTillFirst  []opts.BarData{}
	for day := from; day.Before(to); day = day.AddDate(0, 0, 7) {
		year, week := day.ISOWeek()

		stats, err := c.CalculateRepoPRStatsForDateRange(repo, day, day.AddDate(0, 0, 7).Add(-time.Nanosecond), []string{})
		if err != nil {
			return fmt.Errorf("failed to query stats: %w", err)
		}

		// bar chart data
		yearWeek := fmt.Sprintf("%d-%d", year, week)
		weeks = append(weeks, yearWeek)
		daysOpen = append(daysOpen, opts.BarData{Value: stats.DaysOpenAverage.Float64})
		daysWaiting = append(daysWaiting, opts.BarData{Value: stats.DaysWaitingAverage.Float64})
		daysTillFirst = append(daysTillFirst, opts.BarData{Value: stats.DaysToFirstAverage.Float64})

		// csv data

	}

	var xAxis []string
	var lineOpen, lineBlocked, lineWaiting, lineWaitingOver []opts.LineData

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
			Left: "center", // nolint:misspell
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

*/
