package cmd

import (
	"flag"
	"fmt"
	"time"

	"gopkg.in/gorp.v1"

	"euphoria.io/scope"
)

func init() {
	register("analyze-stats", &analyzeStatsCmd{})
}

type analyzeStatsCmd struct {
	backfill bool
}

func (analyzeStatsCmd) desc() string  { return "analyze recent activity and update stats tables" }
func (analyzeStatsCmd) usage() string { return "analyze-stats" }

func (cmd *analyzeStatsCmd) longdesc() string {
	return `
	Analyze recent activity in order to fill in stats data, such as
	tracking of user engagement.

	By default, this command runs one round of analysis. Multiple rounds
	may be required to catch up to the current time. Run with -backfill
	to automatically loop until caught up.
`[1:]
}

func (cmd *analyzeStatsCmd) flags() *flag.FlagSet {
	fs := flag.NewFlagSet("analyze-stats", flag.ExitOnError)
	fs.BoolVar(&cmd.backfill, "backfill", false, "repeat until caught up")
	return fs
}

func (cmd *analyzeStatsCmd) run(ctx scope.Context, args []string) error {
	heim, b, err := getHeimWithPsqlBackend(ctx)
	if err != nil {
		return err
	}
	defer heim.Backend.Close()

	for {
		var row struct {
			Last gorp.NullTime
		}
		err := b.DbMap.SelectOne(&row, "SELECT stats_sessions_analyze() AS last")
		if err != nil {
			return err
		}
		if !row.Last.Valid {
			fmt.Printf("stored procedure returned NULL, finished?\n")
			break
		}
		fmt.Printf("analyzed up to %s\n", row.Last.Time)
		if !cmd.backfill || row.Last.Time.After(time.Now().Add(-time.Hour-time.Minute)) {
			break
		}
	}
	return nil
}
