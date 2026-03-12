# nbalive

Go client library for NBA.com's live game data CDN. Fetches real-time scoreboards, play-by-play, and box scores with zero external dependencies.

## Requirements

- Go 1.26+

## Install

```sh
go get github.com/darin-patton-hpe/nbalive
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/darin-patton-hpe/nbalive"
)

func main() {
    client := nbalive.NewClient()
    ctx := context.Background()

    // Today's scoreboard
    sb, err := client.Scoreboard(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, g := range sb.Scoreboard.Games {
        fmt.Printf("%s %d - %d %s (%s)\n",
            g.AwayTeam.TeamTricode, g.AwayTeam.Score,
            g.HomeTeam.Score, g.HomeTeam.TeamTricode,
            g.GameStatus)
    }

    // Only in-progress games
    live, err := client.LiveGames(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%d games in progress\n", len(live))
}
```

### One-Shot Fetches

```go
// Play-by-play for a specific game
pbp, err := client.PlayByPlay(ctx, "0022400123")
for _, a := range pbp.Game.Actions {
    fmt.Printf("Q%d %s: %s\n", a.Period, a.Clock, a.Description)
}

// Box score for a specific game
bs, err := client.BoxScore(ctx, "0022400123")
fmt.Printf("%s %d - %d %s\n",
    bs.Game.AwayTeam.TeamTricode, bs.Game.AwayTeam.Score,
    bs.Game.HomeTeam.Score, bs.Game.HomeTeam.TeamTricode)
```

### Live Game Watcher

`Watch` polls a game and emits events on a channel. The channel closes when the game ends or the context is cancelled.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

ch := client.Watch(ctx, "0022400123", nbalive.WatchConfig{
    PollInterval: 10 * time.Second, // min 5s, default 15s
    BoxScore:     true,              // also fetch box scores each tick
})

for event := range ch {
    switch event.Kind {
    case nbalive.EventAction:
        fmt.Printf("[%s] %s\n", event.Action.Clock, event.Action.Description)
    case nbalive.EventBoxScore:
        fmt.Printf("Score: %d-%d\n", event.BoxScore.AwayTeam.Score, event.BoxScore.HomeTeam.Score)
    case nbalive.EventGameOver:
        fmt.Printf("Final: %d-%d\n", event.BoxScore.AwayTeam.Score, event.BoxScore.HomeTeam.Score)
    case nbalive.EventError:
        log.Printf("transient error: %v", event.Err)
    }
}
```

## API Reference

### Client

```go
func NewClient(opts ...Option) *Client
func WithHTTPClient(hc *http.Client) Option
func WithBaseURL(url string) Option
```

### Methods

| Method | Description |
|---|---|
| `Scoreboard(ctx)` | Today's full scoreboard |
| `LiveGames(ctx)` | Convenience filter for in-progress games |
| `PlayByPlay(ctx, gameID)` | Play-by-play actions for a game |
| `BoxScore(ctx, gameID)` | Box score (player/team stats) for a game |
| `Watch(ctx, gameID, cfg)` | Live polling with deduplication, returns `<-chan Event` |

### Event Kinds

| Kind | Payload | When |
|---|---|---|
| `EventAction` | `event.Action` | New play-by-play action (deduplicated by `orderNumber`) |
| `EventBoxScore` | `event.BoxScore` | Updated box score (when `WatchConfig.BoxScore = true`) |
| `EventGameOver` | `event.BoxScore` | `gameStatus == Final`; channel closes after this |
| `EventError` | `event.Err` | Transient fetch/decode error; polling continues |

### Key Types

| Type | Purpose |
|---|---|
| `GameStatus` | `GameScheduled` (1), `GameInProgress` (2), `GameFinal` (3) |
| `BoolString` | Handles NBA's `"1"`/`"0"` JSON string booleans |
| `Duration` | Wraps `time.Duration` with NBA's ISO 8601 clock format (`"PT11M58.00S"`) |

### Game IDs

NBA game IDs are 10-digit strings like `0022400123`. Find them from the scoreboard:

```go
sb, _ := client.Scoreboard(ctx)
for _, g := range sb.Scoreboard.Games {
    fmt.Println(g.GameID, g.HomeTeam.TeamTricode, "vs", g.AwayTeam.TeamTricode)
}
```

## Development

### Build

```sh
go build ./...
```

### Test

```sh
go test ./... -race
```

Tests use `testing/synctest` (Go 1.26) for deterministic watcher tests with fake clocks — no real network calls, no sleeps, sub-second execution.

Run verbose:

```sh
go test ./... -v -race -timeout 30s
```

### Lint

```sh
go vet ./...
```

### Project Structure

```
nbalive/
  go.mod              Module definition (Go 1.26, zero dependencies)
  doc.go              Package-level godoc
  client.go           Client, options, HTTP get/getIfModified
  scoreboard.go       Scoreboard(), LiveGames()
  playbyplay.go       PlayByPlay()
  boxscore.go         BoxScore()
  watcher.go          Watch(), Event, EventKind, WatchConfig
  types.go            All JSON structs, GameStatus, BoolString
  duration.go         Duration type (ISO 8601 PT...S parsing)
  client_test.go      Client + endpoint tests (httptest)
  watcher_test.go     Watcher tests (synctest + fake RoundTripper)
  duration_test.go    Duration parsing + round-trip tests
  types_test.go       BoolString + GameStatus tests
```

## Data Source

All data comes from `https://cdn.nba.com/static/json/liveData/`. No API keys or authentication required. The CDN serves current-season data and updates every ~10-15 seconds during live games.

| Endpoint | URL Pattern |
|---|---|
| Scoreboard | `scoreboard/todaysScoreboard_00.json` |
| Play-by-Play | `playbyplay/playbyplay_{gameID}.json` |
| Box Score | `boxscore/boxscore_{gameID}.json` |

## License

MIT
