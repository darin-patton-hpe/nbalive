package live

import (
	"context"

	"github.com/darin-patton-hpe/nbalive"
)

func (c *Client) Scoreboard(ctx context.Context) (*nbalive.ScoreboardResponse, error) {
	var resp nbalive.ScoreboardResponse
	if err := c.get(ctx, "scoreboard/todaysScoreboard_00.json", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) LiveGames(ctx context.Context) ([]nbalive.Game, error) {
	sb, err := c.Scoreboard(ctx)
	if err != nil {
		return nil, err
	}
	var live []nbalive.Game
	for _, g := range sb.Scoreboard.Games {
		if g.GameStatus == nbalive.GameInProgress {
			live = append(live, g)
		}
	}
	return live, nil
}
