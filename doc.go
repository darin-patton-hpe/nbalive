// Package nbalive provides shared response and model types used by the
// sub-packages in this module.
//
// Use github.com/darin-patton-hpe/nbalive/live for NBA CDN live data and
// github.com/darin-patton-hpe/nbalive/stats for the NBA Stats API.
// Callers create clients independently with live.NewClient() and
// stats.NewClient(), then decode into the shared types defined in this
// package.
package nbalive
