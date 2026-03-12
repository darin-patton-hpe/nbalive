package nbalive

import "context"

// Scoreboard fetches today's scoreboard.
func (c *Client) Scoreboard(ctx context.Context) (*ScoreboardResponse, error) {
	var resp ScoreboardResponse
	if err := c.get(ctx, "scoreboard/todaysScoreboard_00.json", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LiveGames returns only in-progress games from today's scoreboard.
func (c *Client) LiveGames(ctx context.Context) ([]Game, error) {
	sb, err := c.Scoreboard(ctx)
	if err != nil {
		return nil, err
	}
	var live []Game
	for _, g := range sb.Scoreboard.Games {
		if g.GameStatus == GameInProgress {
			live = append(live, g)
		}
	}
	return live, nil
}
