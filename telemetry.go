package common

import (
	"context"
	"time"
)

type EventTelemetry struct {
	Ctx   context.Context
	Title string
	Start time.Time
	End   time.Time
	Err   string
	Code  int
}
