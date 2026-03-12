// Package nbalive provides a Go client for NBA.com's live game data CDN.
//
// The NBA CDN serves static JSON files that are updated in near-real-time
// during live games (~10-15 second intervals). This package wraps those
// endpoints with typed Go structs and supports both one-shot fetches and
// live polling with ETag-based change detection.
//
// # Endpoints
//
//   - Scoreboard: today's games (scores, status, leaders)
//   - PlayByPlay: per-game event stream (shots, fouls, turnovers, etc.)
//   - BoxScore:   per-game stats (player and team statistics)
//
// # Quick Start
//
//	client := nbalive.NewClient()
//	ctx := context.Background()
//
//	// Fetch today's scoreboard.
//	sb, err := client.Scoreboard(ctx)
//
//	// Fetch play-by-play for a specific game.
//	pbp, err := client.PlayByPlay(ctx, "0022400123")
//
//	// Watch a live game for real-time updates.
//	for event := range client.Watch(ctx, gameID, nbalive.WatchConfig{BoxScore: true}) {
//	    switch event.Kind {
//	    case nbalive.EventAction:
//	        fmt.Println(event.Action.Description)
//	    case nbalive.EventGameOver:
//	        fmt.Println("Game over!")
//	    }
//	}
//
// # Data Sources
//
// All data is fetched from https://cdn.nba.com/static/json/liveData/.
// No authentication or API keys are required. The CDN is public and serves
// current-season game data. Historical data may be unavailable after a season ends.
//
// # Zero Dependencies
//
// This package uses only the Go standard library.
package nbalive
