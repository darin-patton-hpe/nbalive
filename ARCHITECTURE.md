# Architecture

## Overview

`nbalive` is a multi-package Go module:

- `nbalive` (root): shared response/model types only
- `nbalive/live`: NBA CDN client (scoreboard, play-by-play, box score, watcher)
- `nbalive/stats`: NBA Stats API client (`ScoreboardByDate`)

This split keeps transport-specific behavior in each client package while reusing one canonical set of JSON structs.

## Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                               Caller                                 │
│                                                                      │
│   live.NewClient()                               stats.NewClient()    │
│      │                                                 │             │
│      │ Scoreboard / LiveGames / PBP / Box / Watch     │ ScoreboardByDate
└──────┼─────────────────────────────────────────────────┼─────────────┘
       │                                                 │
       ▼                                                 ▼
┌──────────────────────────────┐              ┌──────────────────────────────┐
│       package live           │              │       package stats          │
│                              │              │                              │
│ get(ctx,path)                │              │ get(ctx,path)                │
│ getIfModified(ctx,path,etag) │              │ stats headers + date check   │
│ Watch() polling + ETag dedupe│              │ ScoreboardV3 by date         │
└───────────────┬──────────────┘              └───────────────┬──────────────┘
                │                                             │
                ▼                                             ▼
     https://cdn.nba.com/static/json/liveData      https://stats.nba.com/stats
                │                                             │
                └─────────────────────┬───────────────────────┘
                                      ▼
                        ┌──────────────────────────────┐
                        │        package nbalive       │
                        │   shared JSON types/models   │
                        └──────────────────────────────┘
```

## Type Hierarchy (shared root package)

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

live.Event (watcher tagged union)
├── Kind: EventAction    → Action *nbalive.Action
├── Kind: EventBoxScore  → BoxScore *nbalive.BoxScoreGame
├── Kind: EventGameOver  → BoxScore *nbalive.BoxScoreGame
└── Kind: EventError     → Err error
```

## Architecture Decisions

### 1. Split by API surface (`live` + `stats`) with shared root types

**Decision**: Keep only model/types in the root package and move API clients into dedicated sub-packages.

**Why**: The CDN and Stats APIs are operationally distinct (different hosts, header requirements, and endpoint patterns) but return overlapping payloads. Splitting clients by API prevents a monolithic root client and avoids conflating transport concerns while preserving one set of model types.

### 2. No unified root facade client

**Decision**: Callers create `live.NewClient()` and `stats.NewClient()` independently.

**Why**: A root facade would re-couple independent API surfaces and complicate configuration. Independent clients keep intent explicit and avoid forcing one configuration model across both backends.

### 3. Single `<-chan Event` With `Kind` enum (live watcher)

**Decision**: `live.Watch()` returns a single `<-chan Event` tagged by `EventKind`.

**Why**: A single channel preserves event ordering and keeps consumer logic simple (`for range` + `switch`), avoiding multi-channel coordination complexity.

### 4. ETag caching only in watcher path

**Decision**: `live.getIfModified()` is used only by watcher polling.

**Why**: One-shot fetch methods always need fresh payloads; ETag logic is only beneficial for repeated poll loops.

### 5. Custom `Duration` type for ISO 8601

**Decision**: Root `Duration` wraps `time.Duration` with custom JSON parsing.

**Why**: NBA payload clocks/minutes use ISO 8601 duration strings (`PT11M58.00S`), which are not directly handled by `time.Duration` JSON unmarshalling.

### 6. Separate `PlayerStats` and `TeamStats`

**Decision**: Keep distinct structs despite overlap.

**Why**: JSON shape and semantics differ; embedding would imply an inheritance relationship that does not exist.

### 7. Minimal constructor options per client

**Decision**: Both clients support only `WithHTTPClient` and `WithBaseURL`.

**Why**: This covers real customization needs while leaving advanced behavior to caller-provided `*http.Client` configuration.

### 8. `orderNumber` dedupe for play-by-play

**Decision**: Watcher tracks highest `orderNumber` and emits only newer actions.

**Why**: CDN returns full action history every poll; high-water-mark filtering prevents duplicate event emission efficiently.

### 9. Buffered watcher channel (64)

**Decision**: `Watch()` uses `make(chan Event, 64)`.

**Why**: A buffer smooths short consumer slowdowns without unbounded memory growth.

### 10. `BoolString` custom type

**Decision**: Root `BoolString` handles string and bool encodings.

**Why**: NBA payloads encode several booleans as `"1"`/`"0"` strings.

### 11. `testing/synctest` for watcher tests

**Decision**: live watcher tests run with fake time using `testing/synctest` and custom round-trippers.

**Why**: Deterministic timing coverage without flaky sleeps or real network I/O.

## File Responsibilities

| File | Responsibility |
|---|---|
| `doc.go` | Root package documentation (shared types + package split) |
| `types.go` | Shared JSON response/model structs, `GameStatus`, `BoolString` |
| `duration.go` | Shared `Duration` type with ISO 8601 marshal/unmarshal |
| `live/client.go` | Live client + options + `get()` + `getIfModified()` |
| `live/scoreboard.go` | CDN `Scoreboard()` and `LiveGames()` |
| `live/playbyplay.go` | CDN `PlayByPlay()` |
| `live/boxscore.go` | CDN `BoxScore()` |
| `live/watcher.go` | `Watch()`, `Event`, `EventKind`, `WatchConfig`, `emit()` |
| `stats/client.go` | Stats client + options + `get()` + date regex |
| `stats/scoreboard.go` | `ScoreboardByDate()` using ScoreboardV3 |

## Constraints

- **Zero dependencies**: stdlib only.
- **No authentication**: public endpoints (CDN and stats endpoint usage here).
- **Stats scope**: ScoreboardV3 by date only.
- **Shared schema contract**: both clients decode into root `nbalive` types.
- **Watcher minimum poll interval**: 5 seconds enforced by `WatchConfig.withDefaults()`.
