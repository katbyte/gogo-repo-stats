package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	c "github.com/gookit/color"
	"github.com/katbyte/gogo-repo-stats/lib/cache"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// output a report
//
// 2022-01
//   W1  xx/yy open, zz open, zz waiting, zz to first (x greater then 2 weeks)
//   W2
//   W3
//   W4
//  **

// report (last month till now)
// report YYYY-MM (date till now)
// report YYYY-MM YYYY-MM (range a to b)

func CmdReport(_ *cobra.Command, args []string) error {
	var err error
	f := GetFlags()

	from := time.Now().AddDate(0, -1, 0)
	from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	to := time.Now()

	aragc := len(args)
	if aragc == 1 {
		from, err = time.Parse("2006-01", args[0])
		if err != nil {
			return fmt.Errorf("failed to parse time %s : %w", args[0], err)
		}

		if aragc == 2 {
			from, err = time.Parse("2006-01", args[1])
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

	c.Printf("Generating reports forall PRs from <white>%s</> to <white>%s</>...\n", from.Format("2006-01-02"), to.Format("2006-01-02"))

	// todo  look at https://github.com/jedib0t/go-pretty

	// for each month
	for month := from; month.Before(to); month = month.AddDate(0, 1, 0) {
		monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)

		monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		if monthEnd.After(to) {
			monthEnd = to
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{month.Format("2006-01"), "Opened", "Open", "Days Open", "Days Wait", "Days First", "First Over"})
		table.SetBorder(false)
		table.SetAlignment(1)

		n := 0
		for weekStart := monthStart; weekStart.Before(monthEnd); weekStart = weekStart.AddDate(0, 0, 7) {
			n++
			weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Nanosecond)
			if weekEnd.After(monthEnd) {
				weekEnd = monthEnd
			}

			stats, err := cache.CalculateStatsForDateRange(weekStart, weekEnd)
			if err != nil {
				return fmt.Errorf("failed to query stats: %w", err)
			}

			table.Append([]string{
				"W" + strconv.Itoa(n),
				strconv.Itoa(stats.Total),
				strconv.Itoa(stats.Open),
				strconv.FormatFloat(stats.DaysOpenAverage, 'f', 2, 64),
				strconv.FormatFloat(stats.DaysWaitingAverage, 'f', 2, 64),
				strconv.FormatFloat(stats.DaysToFirstAverage, 'f', 2, 64),
				strconv.Itoa(stats.DaysToFirstOver),
			})
		}

		stats, err := cache.CalculateStatsForDateRange(monthStart, monthEnd)
		if err != nil {
			return fmt.Errorf("failed to query stats: %w", err)
		}

		table.SetFooter([]string{
			"SUM",
			strconv.Itoa(stats.Total),
			strconv.Itoa(stats.Open),
			strconv.FormatFloat(stats.DaysOpenAverage, 'f', 2, 64),
			strconv.FormatFloat(stats.DaysWaitingAverage, 'f', 2, 64),
			strconv.FormatFloat(stats.DaysToFirstAverage, 'f', 2, 64),
			strconv.Itoa(stats.DaysToFirstOver),
		})

		table.Render() // Send output
		fmt.Println()
		fmt.Println()
	}

	return nil
}
