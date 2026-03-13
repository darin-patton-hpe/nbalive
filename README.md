# nbalive

Go client library for NBA live data and stats endpoints with zero external dependencies.

The module is split into focused packages:

- `github.com/darin-patton-hpe/nbalive` (shared response/model types only)
- `github.com/darin-patton-hpe/nbalive/live` (NBA CDN live endpoints + watcher)
- `github.com/darin-patton-hpe/nbalive/stats` (NBA Stats API client)

## Requirements

- Go 1.26+

## Install

```sh
go get github.com/darin-patton-hpe/nbalive
```

## Quick Start

### Live CDN client (`live` package)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/darin-patton-hpe/nbalive/live"
)

func main() {
    client := live.NewClient()
    ctx := context.Background()

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
}
```

### Stats API client (`stats` package)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/darin-patton-hpe/nbalive/stats"
)

func main() {
    client := stats.NewClient()
    ctx := context.Background()

    sb, err := client.ScoreboardByDate(ctx, "2024-11-15")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("%s: %d games\n", sb.Scoreboard.GameDate, len(sb.Scoreboard.Games))
}
```

## Live Watcher

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

client := live.NewClient()
ch := client.Watch(ctx, "0022400123", live.WatchConfig{
    PollInterval: 10 * time.Second,
    BoxScore:     true,
})

for event := range ch {
    switch event.Kind {
    case live.EventAction:
        fmt.Printf("[%s] %s\n", event.Action.Clock, event.Action.Description)
    case live.EventBoxScore:
        fmt.Printf("Score: %d-%d\n", event.BoxScore.AwayTeam.Score, event.BoxScore.HomeTeam.Score)
    case live.EventGameOver:
        fmt.Printf("Final: %d-%d\n", event.BoxScore.AwayTeam.Score, event.BoxScore.HomeTeam.Score)
    case live.EventError:
        log.Printf("transient error: %v", event.Err)
    }
}
```

## API Reference

### Shared types (`nbalive`)

The root package provides shared JSON models and helpers used by both sub-packages:

- responses (`ScoreboardResponse`, `PlayByPlayResponse`, `BoxScoreResponse`)
- domain models (`Game`, `Action`, `BoxScoreGame`, `PlayerStats`, `TeamStats`, ...)
- utility types (`GameStatus`, `BoolString`, `Duration`)

### Live package (`live`)

```go
func NewClient(opts ...Option) *Client
func WithHTTPClient(hc *http.Client) Option
func WithBaseURL(url string) Option
```

| Method | Description |
|---|---|
| `Scoreboard(ctx)` | Today's full scoreboard from CDN |
| `LiveGames(ctx)` | In-progress games from today's scoreboard |
| `PlayByPlay(ctx, gameID)` | Play-by-play actions for a game |
| `BoxScore(ctx, gameID)` | Box score for a game |
| `Watch(ctx, gameID, cfg)` | Polling watcher with ETag + dedupe, returns `<-chan Event` |

Watcher event kinds:

- `EventAction`
- `EventBoxScore`
- `EventGameOver`
- `EventError`

### Stats package (`stats`)

```go
func NewClient(opts ...Option) *Client
func WithHTTPClient(hc *http.Client) Option
func WithBaseURL(url string) Option
```

| Method | Description |
|---|---|
| `ScoreboardByDate(ctx, date)` | ScoreboardV3 for a specific `YYYY-MM-DD` date |

## Development

### Build

```sh
go build ./...
```

### Test

```sh
go test ./... -race
```

Watcher tests use `testing/synctest` (Go 1.26) for deterministic fake-clock polling tests.

### Lint

```sh
go vet ./...
```

### Project Structure

```
nbalive/
  go.mod
  doc.go
  types.go
  duration.go
  types_test.go
  duration_test.go
  live/
    client.go
    scoreboard.go
    playbyplay.go
    boxscore.go
    watcher.go
    client_test.go
    watcher_test.go
  stats/
    client.go
    scoreboard.go
    scoreboard_test.go
```

## Data Sources

### Live CDN (`live`)

Base URL: `https://cdn.nba.com/static/json/liveData`

| Endpoint | URL Pattern |
|---|---|
| Scoreboard | `scoreboard/todaysScoreboard_00.json` |
| Play-by-Play | `playbyplay/playbyplay_{gameID}.json` |
| Box Score | `boxscore/boxscore_{gameID}.json` |

### Stats API (`stats`)

Base URL: `https://stats.nba.com/stats`

| Endpoint | URL Pattern |
|---|---|
| ScoreboardV3 | `scoreboardv3?GameDate=YYYY-MM-DD&LeagueID=00` |

## License

MIT
