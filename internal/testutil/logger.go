// Package testutil provides shared test helpers for NetVantage packages.
package testutil

import "go.uber.org/zap"

// Logger returns a development Zap logger for use in tests.
// Panics on construction failure (should never happen in tests).
func Logger() *zap.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic("testutil.Logger: " + err.Error())
	}
	return l
}
