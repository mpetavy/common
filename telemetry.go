package common

import (
	"context"
	"time"
)

type EventTelemetry struct {
	IsTelemetryRequest bool
	Ctx                context.Context
	Title              string
	Start              time.Time
	End                time.Time
	Err                string
	Code               int
}

func (et EventTelemetry) IsSuccess() bool {
	return et.Code/100 == 2
}
