# Architecture

## Overview

`nbalive` is a single Go package that wraps NBA.com's live CDN endpoints. The library provides typed Go structs for JSON deserialization, one-shot fetch methods, and a channel-based live game watcher with ETag caching and play-by-play deduplication.

## Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                          Caller                                  │
│                                                                  │
│   client.Scoreboard(ctx)    client.Watch(ctx, gameID, cfg)       │
│   client.PlayByPlay(ctx,id)       │                              │
│   client.BoxScore(ctx, id)        │                              │
│   client.LiveGames(ctx)           │                              │
└──────────┬────────────────────────┼──────────────────────────────┘
           │                        │
           │ One-shot               │ Polling loop (goroutine)
           │                        │
┌──────────▼────────────────────────▼──────────────────────────────┐
│                         Client                                   │
│                                                                  │
│  ┌─────────────────────┐    ┌──────────────────────────────────┐ │
│  │     get(ctx,path)   │    │         Watch goroutine          │ │
│  │                     │    │                                  │ │
│  │  Build request      │    │  ticker.C ──► getIfModified()    │ │
│  │  Set headers        │    │              (ETag caching)      │ │
│  │  httpClient.Do()    │    │                  │               │ │
│  │  Decode JSON → dst  │    │       ┌──────────┼────────┐     │ │
│  └─────────────────────┘    │       │          │        │     │ │
│                             │    new actions  304     error   │ │
│                             │    (deduped by  (skip)  (emit   │ │
│                             │    orderNumber)         + cont) │ │
│                             │       │                   │     │ │
│                             │       ▼                   ▼     │ │
│                             │    ch <- Event{Kind: ...}       │ │
│                             │                                  │ │
│                             │  GameStatus==Final ──► close(ch) │ │
│                             │  ctx.Done()        ──► close(ch) │ │
│                             └──────────────────────────────────┘ │
└──────────────────────────────────┬───────────────────────────────┘
                                   │
                                   │ HTTP GET
                                   ▼
┌──────────────────────────────────────────────────────────────────┐
│                  NBA CDN (cdn.nba.com)                            │
│                                                                  │
│  /static/json/liveData/scoreboard/todaysScoreboard_00.json       │
│  /static/json/liveData/playbyplay/playbyplay_{gameID}.json       │
│  /static/json/liveData/boxscore/boxscore_{gameID}.json           │
│                                                                  │
│  Updates: ~10-15s during live games                              │
│  Auth: None (public CDN, browser User-Agent)                     │
│  Caching: ETag / If-None-Match → 304 Not Modified                │
└──────────────────────────────────────────────────────────────────┘
```

### Type Hierarchy

```
ScoreboardResponse          PlayByPlayResponse        BoxScoreResponse
├── Meta                    ├── Meta                  ├── Meta
└── Scoreboard              └── PlayByPlayGame        └── BoxScoreGame
    └── []Game                  ├── GameID                ├── GameStatus
        ├── GameStatus          └── []Action              ├── Arena
        ├── Duration (clock)        ├── Duration (clock)  ├── []Official
        ├── GameTeam (home)         ├── OrderNumber       ├── BoxTeam (home)
        ├── GameTeam (away)         ├── ActionType        │   ├── []BoxPlayer
        │   └── []Period            ├── shot coords       │   │   ├── BoolString (starter)
        └── GameLeaders             └── ScoreHome/Away    │   │   └── PlayerStats
            └── PlayerLeader                              │   └── TeamStats
                                                          └── BoxTeam (away)

Event (tagged union from Watch)
├── Kind: EventAction    → Action *Action
├── Kind: EventBoxScore  → BoxScore *BoxScoreGame
├── Kind: EventGameOver  → BoxScore *BoxScoreGame
└── Kind: EventError     → Err error
```

## Architecture Decisions

### 1. Methods on `*Client`, Not Service Pattern

**Decision**: All endpoints are methods directly on `*Client` (`client.Scoreboard()`, `client.PlayByPlay()`, etc.) rather than using a service decomposition like `client.PlayByPlay.Get()`.

**Why**: The NBA CDN exposes 3 endpoints. A `go-github`-style service pattern (`client.PullRequests.List()`) is designed for APIs with dozens of resource groups. For 3 endpoints, services add indirection without organizational benefit. Direct methods keep the API surface minimal and discoverable.

### 2. Single `<-chan Event` With `Kind` Enum

**Decision**: `Watch()` returns a single `<-chan Event` where `Event` is a tagged union with a `Kind` field discriminating between `EventAction`, `EventBoxScore`, `EventGameOver`, and `EventError`.

**Why**: Separate channels per event type (actions channel, boxscore channel, error channel) would force callers into awkward multi-select patterns and create ordering ambiguities. A single channel preserves emission order and allows a simple `for event := range ch` + `switch event.Kind` pattern. The `Kind` enum makes the closed set of event types explicit at compile time.

### 3. ETag Caching in Watcher Only

**Decision**: `getIfModified()` (which handles `If-None-Match` / `304 Not Modified`) is used exclusively by the watcher. One-shot `get()` does not support ETags.

**Why**: One-shot callers always want fresh data — they call once and use the result. ETag caching only matters for repeated polling where you want to skip JSON decoding when data hasn't changed. Keeping ETag logic out of `get()` simplifies the common path and avoids leaking caching concerns into the one-shot API.

### 4. Custom `Duration` Type for ISO 8601

**Decision**: A custom `Duration` type wraps `time.Duration` and implements `json.Unmarshaler` to parse NBA's ISO 8601 clock format (`"PT11M58.00S"`).

**Why**: NBA's JSON encodes game clocks and player minutes as ISO 8601 duration strings. Go's `time.Duration` cannot be directly unmarshaled from this format. A custom type with regex-based parsing handles all observed patterns (`"PT11M58.00S"`, `"PT0S"`, `""`, `null`) while exposing the familiar `time.Duration` interface for downstream use.

### 5. Separate `PlayerStats` and `TeamStats`

**Decision**: `PlayerStats` and `TeamStats` are separate structs despite overlapping fields, rather than using embedding or a shared base.

**Why**: The NBA JSON returns flat objects for both, but `TeamStats` has ~15 additional team-specific fields (`BenchPoints`, `BiggestLead`, `LeadChanges`, `TrueShootingPercentage`, etc.) that don't exist on players. Embedding would create a misleading type relationship — a team stat is not "a player stat plus extras." Separate structs keep each type honest about its JSON shape and avoid confusing accessor behavior.

### 6. Minimal Constructor Options

**Decision**: `NewClient` accepts only two options: `WithHTTPClient(*http.Client)` and `WithBaseURL(string)`.

**Why**: Options like `WithUserAgent`, `WithTimeout`, and `WithHeaders` were considered and rejected. All of these are already configurable through `WithHTTPClient` — callers can construct an `*http.Client` with any transport, timeout, or header policy they want. Two options cover 100% of use cases without the library reinventing `http.Client` configuration.

### 7. `orderNumber` for Play-by-Play Deduplication

**Decision**: The watcher deduplicates play-by-play actions using `orderNumber` (a monotonically increasing integer), tracking the highest seen value and only emitting actions with `orderNumber > lastOrder`.

**Why**: The CDN returns the full action list on every request. Without deduplication, a watcher polling every 10 seconds would re-emit hundreds of actions. `orderNumber` is the NBA's canonical sort key — it increases monotonically and never resets. Tracking the high-water mark and filtering is O(n) per poll with no extra storage.

### 8. Buffered Channel (64)

**Decision**: The watcher's event channel has a buffer of 64.

**Why**: A buffered channel decouples the polling goroutine from the consumer. If the consumer is briefly slow (e.g., writing to a database), the watcher can continue emitting without blocking. 64 is large enough to absorb a burst of play-by-play actions from a single poll (a typical NBA quarter has ~100-150 total actions across all periods) while being small enough that a truly stuck consumer will apply backpressure rather than accumulating unbounded memory.

### 9. `BoolString` Custom Type

**Decision**: A custom `BoolString` type handles NBA's `"1"`/`"0"` JSON string booleans.

**Why**: NBA's box score JSON encodes boolean fields (`starter`, `oncourt`, `played`) as string `"1"` or `"0"` rather than JSON booleans. Without a custom type, these would deserialize as plain strings, forcing every caller to write `player.Starter == "1"`. `BoolString` provides `Bool()` and handles the edge case where some fields arrive as actual JSON booleans.

### 10. `testing/synctest` for Watcher Tests

**Decision**: Watcher tests use Go 1.26's `testing/synctest` with a fake `http.RoundTripper` instead of `httptest.NewServer` and real time delays.

**Why**: The watcher uses `time.NewTicker` for polling. Testing with real time would require either very short intervals (flaky under CI load) or long waits (slow test suite). `synctest` provides a fake clock — `time.Sleep` and `time.NewTicker` advance instantly within a synctest bubble. A fake `RoundTripper` replaces `httptest.NewServer` because synctest bubbles cannot perform real network I/O. This gives deterministic, sub-second watcher tests with full coverage of timing-dependent behavior.

## File Responsibilities

| File | Responsibility |
|---|---|
| `doc.go` | Package-level godoc with quick start examples |
| `client.go` | `Client` struct, functional options, `get()`, `getIfModified()` |
| `scoreboard.go` | `Scoreboard()` and `LiveGames()` methods |
| `playbyplay.go` | `PlayByPlay()` method |
| `boxscore.go` | `BoxScore()` method |
| `watcher.go` | `Watch()`, `Event`, `EventKind`, `WatchConfig`, `emit()` |
| `types.go` | All JSON response/model structs, `GameStatus`, `BoolString` |
| `duration.go` | `Duration` type with ISO 8601 marshal/unmarshal |

## Constraints

- **Zero dependencies**: stdlib only. No third-party HTTP clients, JSON libraries, or testing frameworks.
- **No authentication**: The NBA CDN is public. The library sends browser-like headers but requires no API keys or tokens.
- **Current season only**: The CDN serves current-season data. Historical game data may return 404 after a season ends.
- **Minimum poll interval**: 5 seconds, enforced by `WatchConfig.withDefaults()`. The CDN updates every ~10-15 seconds; polling faster wastes bandwidth without yielding new data.
