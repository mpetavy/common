package opentelemetry

import (
	"context"
	"errors"
	"fmt"
	"github.com/mpetavy/common"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	tracer "go.opentelemetry.io/otel/trace"
)

const (
	FlagNameOpentelemetryEnabled = "opentelemetry.enabled"
)

var (
	FlagOpentelemetryEnabled = common.SystemFlagBool(FlagNameOpentelemetryEnabled, true, "OpenTelemetry enabled")
	Telemetry                *OpenTelemetry
)

func init() {
	common.Events.AddListener(common.EventFlagsSet{}, func(event common.Event) {
		var err error

		Telemetry, err = NewTelemetry(context.Background())
		common.Panic(err)
	})

	common.Events.AddListener(common.EventShutdown{}, func(event common.Event) {
		if Telemetry == nil {
			return
		}

		err := Telemetry.Shutdown()
		common.Panic(err)

		Telemetry = nil
	})

	common.Events.AddListener(common.EventTelemetry{}, func(event common.Event) {
		common.Catch(func() error {
			if Telemetry == nil {
				return nil
			}

			eventTelemetry := event.(common.EventTelemetry)

			t := otel.Tracer(common.Title())

			_, span := t.Start(
				context.Background(),
				eventTelemetry.Title,
				tracer.WithTimestamp(eventTelemetry.Start),
			)

			span.End()

			return nil
		})
	})

	common.Events.AddListener(common.EventLog{}, func(event common.Event) {
		common.Catch(func() error {
			if Telemetry == nil {
				return nil
			}

			eventLog := event.(common.EventLog)

			switch eventLog.Entry.Level {
			case common.LevelDebug:
				Telemetry.Logger.Debug(eventLog.Entry.Msg)
			case common.LevelInfo:
				Telemetry.Logger.Info(eventLog.Entry.Msg)
			case common.LevelWarn:
				Telemetry.Logger.Warn(eventLog.Entry.Msg)
			case common.LevelError:
				Telemetry.Logger.Error(eventLog.Entry.Msg)
			case common.LevelFatal:
				Telemetry.Logger.Error(fmt.Sprintf("FATAL: %s", eventLog.Entry.Msg))
			}

			return nil
		})
	})
}

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// Set up logger provider.
	loggerProvider, err := newLoggerProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider() (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
	)
	return traceProvider, nil
}

func newMeterProvider() (*metric.MeterProvider, error) {
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 3s for demonstrative purposes.
			metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}

func newLoggerProvider() (*log.LoggerProvider, error) {
	logExporter, err := stdoutlog.New()
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}

type OpenTelemetry struct {
	Tracer       tracer.Tracer
	Ctx          context.Context
	Logger       *slog.Logger
	shutdownFunc func(context.Context) error
}

type OpenTelemetrySpan struct {
	telemery *OpenTelemetry
	ctx      context.Context
	span     tracer.Span
}

func NewTelemetry(ctx context.Context) (*OpenTelemetry, error) {
	if !*FlagOpentelemetryEnabled {
		return &OpenTelemetry{}, nil
	}

	err := os.Setenv("OTEL_RESOURCE_ATTRIBUTES", fmt.Sprintf("service.name=%s,service.version=%s", common.Title(), common.Version(true, true, true)))
	if common.Error(err) {
		return nil, err
	}

	shutdownFunc, err := setupOTelSDK(ctx)
	if common.Error(err) {
		return nil, err
	}

	telemetry := &OpenTelemetry{
		Tracer:       otel.Tracer(common.Title()),
		Ctx:          ctx,
		Logger:       otelslog.NewLogger(common.Title()),
		shutdownFunc: shutdownFunc,
	}

	return telemetry, nil
}

func (t *OpenTelemetry) Shutdown() error {
	if !*FlagOpentelemetryEnabled {
		return nil
	}

	err := t.shutdownFunc(t.Ctx)
	if common.Error(err) {
		return err
	}

	return nil
}
