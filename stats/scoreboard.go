package stats

import (
	"context"
	"fmt"

	"github.com/darin-patton-hpe/nbalive"
)

// ScoreboardByDate fetches the scoreboard for a specific date.
// The date must be in YYYY-MM-DD format (e.g. "2024-11-15").
//
// This calls the NBA Stats API ScoreboardV3 endpoint, which returns
// the same JSON shape as the CDN's live scoreboard.
func (c *Client) ScoreboardByDate(ctx context.Context, date string) (*nbalive.ScoreboardResponse, error) {
	if !dateRe.MatchString(date) {
		return nil, fmt.Errorf("nbalive/stats: invalid date format %q, want YYYY-MM-DD", date)
	}
	var resp nbalive.ScoreboardResponse
	if err := c.get(ctx, fmt.Sprintf("scoreboardv3?GameDate=%s&LeagueID=00", date), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
