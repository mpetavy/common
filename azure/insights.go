package azure

import (
	"fmt"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/mpetavy/common"
	"net/http"
	"time"
)

const (
	FlagNameInsightsInstrumentationkey = "azure.insights.instrumentationkey"
	FlagNameInsightsBatchSize          = "azure.insights.batchsize"
	FlagNameInsightsBatchInterval      = "azure.insights.batchinterval"
)

var (
	FlagInsightsInstrumentationkey = common.SystemFlagString(FlagNameInsightsInstrumentationkey, "", "Azure insights instrumentation key")
	FlagInsightsBatchSize          = common.SystemFlagInt(FlagNameInsightsBatchSize, 8192, "Azure insights batch size")
	FlagInsightsBatchInterval      = common.SystemFlagInt(FlagNameInsightsBatchInterval, 100, "Azure insights batch interval")

	insightClient appinsights.TelemetryClient
)

func init() {
	common.Events.AddListener(common.EventFlagsSet{}, func(event common.Event) {
		if *FlagInsightsInstrumentationkey == "" {
			return
		}

		telemetryConfig := appinsights.NewTelemetryConfiguration(*FlagInsightsInstrumentationkey)
		telemetryConfig.MaxBatchSize = *FlagInsightsBatchSize
		telemetryConfig.MaxBatchInterval = common.MillisecondToDuration(*FlagInsightsBatchInterval)

		insightClient = appinsights.NewTelemetryClientFromConfig(telemetryConfig)
	})

	common.Events.AddListener(common.EventShutdown{}, func(event common.Event) {
		if insightClient == nil {
			return
		}

		select {
		case <-insightClient.Channel().Close(common.MillisecondToDuration(*FlagAzureTimeout)):
			// Ten second timeout for retries.

			// If we got here, then all telemetry was submitted
			// successfully, and we can proceed to exiting.
		case <-time.After(common.MillisecondToDuration(*FlagAzureTimeout)):
			// Thirty second absolute timeout.  This covers any
			// previous telemetry submission that may not have
			// completed before Close was called.

			// There are a number of reasons we could have
			// reached here.  We gave it a go, but telemetry
			// submission failed somewhere.  Perhaps old events
			// were still retrying, or perhaps we're throttled.
			// Either way, we don't want to wait around for it
			// to complete, so let's just exit.
		}
	})

	common.Events.AddListener(common.EventTelemetry{}, func(event common.Event) {
		common.Catch(func() error {
			eventTelemetry := event.(common.EventTelemetry)

			if eventTelemetry.Code == 0 {
				eventTelemetry.Code = http.StatusOK
			}

			req := common.Split(eventTelemetry.Title, " ")
			if len(req) == 1 {
				req = []string{"", req[0]}
			}

			request := appinsights.NewRequestTelemetry(req[0], req[1], 0, http.StatusText(eventTelemetry.Code))

			// Note that the timestamp will be set to time.Now() minus the
			// specified duration.  This can be overridden by either manually
			// setting the Timestamp and Duration fields, or with MarkTime:
			request.MarkTime(eventTelemetry.Start, eventTelemetry.End)

			// Source of request
			request.Source = req[1]

			// Success is normally inferred from the responseCode, but can be overridden:
			request.Success = eventTelemetry.Err == ""

			// Request ID's are randomly generated GUIDs, but this can also be overridden:
			//request.Id = "<id>"

			// Context tags become more useful here as well
			//request.Tags.Session().SetId("<session id>")
			//request.Tags.User().SetAccountId("<user id>")

			// Finally track it
			insightClient.Track(request)

			return nil
		})
	})

	common.Events.AddListener(common.EventLog{}, func(event common.Event) {
		common.Catch(func() error {
			if insightClient == nil {
				return nil
			}

			eventLog := event.(common.EventLog)

			switch eventLog.Entry.Level {
			case common.LevelDebug:
				trace := appinsights.NewTraceTelemetry(eventLog.Entry.Msg, appinsights.Verbose)
				trace.Properties["goroutineid"] = fmt.Sprintf("%d", eventLog.Entry.GoRoutineId)
				trace.Properties["source"] = eventLog.Entry.Source
				trace.Timestamp = eventLog.Entry.Time
				insightClient.Track(trace)
			case common.LevelInfo:
				trace := appinsights.NewTraceTelemetry(eventLog.Entry.Msg, appinsights.Information)
				trace.Properties["goroutineid"] = fmt.Sprintf("%d", eventLog.Entry.GoRoutineId)
				trace.Properties["source"] = eventLog.Entry.Source
				trace.Timestamp = eventLog.Entry.Time
				insightClient.Track(trace)
			case common.LevelWarn:
				trace := appinsights.NewTraceTelemetry(eventLog.Entry.Msg, appinsights.Warning)
				trace.Properties["goroutineid"] = fmt.Sprintf("%d", eventLog.Entry.GoRoutineId)
				trace.Properties["source"] = eventLog.Entry.Source
				trace.Timestamp = eventLog.Entry.Time
				insightClient.Track(trace)
			case common.LevelError:
				trace := appinsights.NewTraceTelemetry(eventLog.Entry.Msg, appinsights.Error)
				trace.Properties["goroutineid"] = fmt.Sprintf("%d", eventLog.Entry.GoRoutineId)
				trace.Properties["source"] = eventLog.Entry.Source
				trace.Timestamp = eventLog.Entry.Time
				insightClient.Track(trace)
			case common.LevelFatal:
				trace := appinsights.NewTraceTelemetry(eventLog.Entry.Msg, appinsights.Critical)
				trace.Properties["goroutineid"] = fmt.Sprintf("%d", eventLog.Entry.GoRoutineId)
				trace.Properties["source"] = eventLog.Entry.Source
				trace.Timestamp = eventLog.Entry.Time
				insightClient.Track(trace)
			}

			return nil
		})
	})
}
