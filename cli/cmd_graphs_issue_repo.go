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

// issue labels:
//  version - v0.x, v1.x, v2.x, v3.x, v4.x
//  type - enhancement, bug, question, crash, documentation, new-resource, new-datasourcxe
//  service - service/*

type DayStatsIssuesByType struct {
	Total         int
	Bug           int
	Enhancement   int
	Question      int
	Crash         int
	Documentation int
	NewResource   int
	NewDatasource int
	Other         int
}

func GraphRepoOpenPRsDailyByType(cache *cache.Cache, outPath string, from, to time.Time, repos []string) error {
	// f := GetFlags() // todo out path ends up in flags

	c.Printf("    Issues open daily..\n")

	// populate dates
	dates := map[string]DayStatsIssuesByType{}
	for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
		k := day.Format("2006-01-02")
		dates[k] = DayStatsIssuesByType{}
	}

	// get all issues for range
	issues, err := cache.GetRepoIssuesOpenForDateRange(repos, from, to)
	if err != nil {
		return fmt.Errorf("getting PRs: %w", err)
	}
	c.Printf("      %d issues found\n", len(*issues))

	// for each issue in range
	for _, issue := range *issues {
		opened := time.Date(issue.Created.Year(), issue.Created.Month(), issue.Created.Day(), 0, 0, 0, 0, time.UTC)

		closed := issue.Closed
		if issue.State == "open" {
			closed = time.Now()
		}
		closed = time.Date(closed.Year(), closed.Month(), closed.Day(), 0, 0, 0, 0, time.UTC)

		// for each day from issued opened to closed (or now) count this issue using the label

		for day := opened; ; day = day.AddDate(0, 0, 1) {
			if day.Before(from.AddDate(0, 0, -1)) {
				continue
			}

			k := day.Format("2006-01-02")
			d := dates[k]
			d.Total++

			if strings.Contains(issue.Labels, "question") {
				// d.Question++
				d.Question++
			} else if strings.Contains(issue.Labels, "crash") {
				// d.Crash++
				d.Bug++
			} else if strings.Contains(issue.Labels, "bug") {
				d.Bug++
			} else if strings.Contains(issue.Labels, "new-resource") {
				// d.NewResource++
				d.Enhancement++
			} else if strings.Contains(issue.Labels, "new-datasource") {
				// d.NewDatasource++
				d.Enhancement++
			} else if strings.Contains(issue.Labels, "enhancement") {
				d.Enhancement++
			} else if strings.Contains(issue.Labels, "documentation") {
				// d.Documentation++
				d.Enhancement++
			} else {
				d.Other++
			}

			dates[k] = d

			// check is down here so issues open for less than 1 day are counted
			if !day.Before(closed) {
				break // issue is closed, onto next
			}

			if day.After(to) {
				break
			}
		}
	}

	var xAxis []string
	// var lineBug, lineEnhancement, lineQuestion, lineCrash, lineNewResource, lineNewDatasource, lineDocumentation, lineOther []opts.LineData
	var lineBug, lineEnhancement, lineQuestion, lineOther []opts.LineData

	data := [][]string{{"date", "total", "other", "bug", "enhancement", "question"}} // ,"crash",  "new-resource", "new-datasource", "enhancement", "documentation", }}

	days := make([]string, 0, len(dates))
	for day := range dates {
		days = append(days, day)
	}

	sort.Strings(days)
	for _, date := range days {
		day := dates[date]
		data = append(data, []string{date,
			strconv.Itoa(day.Total),
			strconv.Itoa(day.Other),
			strconv.Itoa(day.Bug),
			strconv.Itoa(day.Enhancement),
			strconv.Itoa(day.Question),
			/*
				strconv.Itoa(day.Crash),
				strconv.Itoa(day.NewResource),
				strconv.Itoa(day.NewDatasource),
				strconv.Itoa(day.Enhancement),
				strconv.Itoa(day.Documentation),

			*/
		})

		xAxis = append(xAxis, date)

		// totalLine = append(totalLine, opts.LineData{Value: day.Total})
		lineOther = append(lineOther, opts.LineData{Value: day.Other})
		lineBug = append(lineBug, opts.LineData{Value: day.Bug})
		lineEnhancement = append(lineEnhancement, opts.LineData{Value: day.Enhancement})
		lineQuestion = append(lineEnhancement, opts.LineData{Value: day.Question})

		/*
			lineQuestion = append(lineQuestion, opts.LineData{Value: day.Question})
				lineCrash = append(lineCrash, opts.LineData{Value: day.Crash})
				lineNewResource = append(lineNewResource, opts.LineData{Value: day.NewResource})
				lineNewDatasource = append(lineNewDatasource, opts.LineData{Value: day.NewDatasource})
				lineDocumentation = append(lineDocumentation, opts.LineData{Value: day.Documentation})
		*/
	}

	// write raw data
	file, err := os.Create(outPath + "/daily-issues-open.csv")
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
			Title:    strings.Join(repoShortNames, ",") + " Issues Open (daily)",
			Subtitle: "By Type: other, bug, enhancement, question",
			Left:     "center", // nolint:misspell
		}),

		charts.WithXAxisOpts(opts.XAxis{
			Name: "Date",
			// AxisLabel: &opts.AxisLabel{Show: true, Formatter: "{value} x-unit"},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Issues",
			// AxisLabel: &opts.AxisLabel{Show: true, Formatter: "{value} x-unit"},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1500px",
			Height: "750px",
		}),
		charts.WithColorsOpts(opts.Colors{"#000000", "#C13530", "#2E4555", "#62A0A8", "#5470c6"}),
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
		charts.WithLineChartOpts(opts.LineChart{Stack: "issues"}),
		charts.WithLineStyleOpts(opts.LineStyle{Width: 1, Opacity: 0.9}),
	}

	graph.AddSeries("Other", lineOther)
	graph.AddSeries("Bug", lineBug)
	graph.AddSeries("Enhancement", lineEnhancement)
	graph.AddSeries("Question", lineQuestion).SetSeriesOptions(prStackOps...)

	// graph.AddSeries("Crash", lineCrash)
	// graph.AddSeries("New Resource", lineNewResource)
	// graph.AddSeries("New Datasource", lineNewDatasource)
	// graph.AddSeries("Documentation", lineDocumentation)

	// Where the magic happens
	file, err = os.Create(outPath + "/daily-issues-open-type.html")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = graph.Render(file)
	if err != nil {
		return fmt.Errorf("failed to render graph graph: %w", err)
	}

	return nil
}
