package common

import (
	"embed"
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
)

const (
	APACHE string = "https://www.apache.org/licenses/LICENSE-2.0.html"
	GPL2   string = "https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html"

	SERVICE_SIMULATE  = "simulate"
	SERVICE_START     = "start"
	SERVICE_STOP      = "stop"
	SERVICE_RESTART   = "restart"
	SERVICE_INSTALL   = "install"
	SERVICE_UNINSTALL = "uninstall"
)

// Info information of the application
type application struct {
	// Name of the application
	Title string
	// Version of the application
	Version string
	// Git label development
	Git string
	// Time label development
	Time time.Time
	// Build label development
	Build string
	// Date of development
	Date string
	// Description of the application
	Description string
	// Developer of the application
	Developer string
	// License of the application
	License string
	// Homepage of the application
	Homepage string
	//Resources
	Resources *embed.FS
	//StartFunc
	StartFunc func() error
	//StopFunc
	StopFunc func() error
	//RunFunc
	RunFunc func() error
	//Service
	Service service.Service
	//ServiceConfig
	ServiceConfig *service.Config
}

type EventShutdown struct {
}

type EventFlagsSet struct {
}

type EventFlagsParsed struct {
}

type ErrExit struct {
}

func (e *ErrExit) Error() string { return "" }

const (
	FlagNameAppProduct      = "app.product"
	FlagNameAppTicker       = "app.ticker"
	FlagNameService         = "service"
	FlagNameServiceUsername = "service.username"
	FlagNameServicePassword = "service.password"
	FlagNameServiceTimeout  = "service.timeout"
	FlagNameUsage           = "h"
	FlagNameUsageMd         = "hmd"
	FlagNameNoBanner        = "nb"
)

var (
	FlagService             = flag.String(FlagNameService, "", "Service operation ("+strings.Join([]string{SERVICE_SIMULATE, SERVICE_START, SERVICE_STOP, SERVICE_RESTART, SERVICE_INSTALL, SERVICE_UNINSTALL}, ",")+")")
	FlagServiceUser         = flag.String(FlagNameServiceUsername, "", "Service user")
	FlagServicePassword     = flag.String(FlagNameServicePassword, "", "Service password")
	FlagServiceStartTimeout = flag.Int(FlagNameServiceTimeout, 1000, "Service timeout")
	FlagUsage               = flag.Bool(FlagNameUsage, false, "show flags description and usage")
	FlagUsageMd             = flag.Bool(FlagNameUsageMd, false, "show flags description and usage in markdown format")
	FlagNoBanner            = flag.Bool(FlagNameNoBanner, false, "no copyright banner")

	app                     *application
	FlagAppProduct          *string
	FlagAppTicker           *int
	ticker                  *time.Ticker
	appLifecycle            = NewNotice(true)
	onceBanner              sync.Once
	onceRunningAsService    sync.Once
	onceRunningAsExecutable sync.Once
	onceRunningInteractive  sync.Once
	onceShutdownHooks       sync.Once
	onceTitle               sync.Once
	title                   string
	runningAsService        bool
	runningAsExecutable     bool
	runningInteractive      bool
	restart                 bool
	restartCh               = make(chan struct{})
	ctrlC                   = make(chan os.Signal, 1)
	isFirstTicker           = true
)

func Init(title string, version string, git string, build string, date string, description string, developer string, homepage string, license string, resources *embed.FS, startFunc func() error, stopFunc func() error, runFunc func() error, runTime time.Duration) {
	FlagAppProduct = flag.String(FlagNameAppProduct, Title(), "app product")
	FlagAppTicker = flag.Int(FlagNameAppTicker, int(runTime.Milliseconds()), "app execution ticker")

	var ti time.Time

	if git == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					git = setting.Value
				case "vcs.time":
					if version != "" {
						continue
					}
					ti, _ = time.Parse(time.RFC3339, setting.Value)
					d := ti.Sub(time.Date(ti.Year(), 1, 1, 0, 0, 0, 0, time.UTC))

					version = fmt.Sprintf("%d.%d", (ti.Year()-22)%100, int(d.Abs().Hours())/24)
				}
			}
		}
	}

	app = &application{
		Title:       title,
		Version:     version,
		Git:         git,
		Time:        ti,
		Build:       build,
		Date:        date,
		Description: description,
		Developer:   developer,
		License:     license,
		Homepage:    homepage,
		Resources:   resources,
		StartFunc:   startFunc,
		StopFunc:    stopFunc,
		RunFunc:     runFunc,
	}

	executable, err := os.Executable()
	Panic(err)

	app.ServiceConfig = &service.Config{
		Name:             Eval(IsWindowsOS(), Capitalize(app.Title), app.Title).(string),
		DisplayName:      Eval(IsWindowsOS(), Capitalize(app.Title), app.Title).(string),
		Description:      Capitalize(app.Description),
		WorkingDirectory: filepath.Dir(executable),
	}

	app.Service, err = service.New(app, app.ServiceConfig)
	Panic(err)
}

func usage() error {
	if *FlagUsageMd {
		dir, err := os.Getwd()
		if Error(err) {
			return err
		}

		fmt.Printf("Parameter | Default value | Only CmdLine | Description\n")
		fmt.Printf("------------ | ------------- | ------------- | -------------\n")

		flag.VisitAll(func(fl *flag.Flag) {
			defValue := fl.DefValue
			if strings.HasPrefix(defValue, dir) {
				defValue = fmt.Sprintf("./%s", defValue[len(dir)+1:])
			}

			onlyCmdLine := ""
			if IsCmdlineOnlyFlag(fl.Name) {
				onlyCmdLine = "*"
			}

			fmt.Printf("%s | %s | %s | %s\n", fl.Name, defValue, onlyCmdLine, fl.Usage)
		})

		return &ErrExit{}
	}

	if *FlagUsage {
		flag.Usage()

		return &ErrExit{}
	}

	return nil
}

func checkMandatoryFlags(flags []string) error {
	if *FlagService == SERVICE_UNINSTALL {
		return nil
	}

	if flags != nil {
		notDefined := strings.Builder{}

		for _, mf := range flags {
			choices := strings.Split(mf, "|")
			for i := 0; i < len(choices); i++ {
				choices[i] = "\"-" + choices[i] + "\""
			}

			allChoices := strings.Join(choices, " or ")
			defined := 0

			for _, flagName := range strings.Split(mf, "|") {
				fl := flag.Lookup(flagName)

				if fl == nil || fl.Value == nil {
					return fmt.Errorf("unknown mandatory flag: \"%s\"", flagName)
				}

				if fl.Value.String() != "" {
					defined++
				}
			}

			switch {
			case defined == 0:
				if notDefined.Len() > 0 {
					notDefined.WriteString(", ")
				}
				notDefined.WriteString(allChoices)
			case defined > 1:
				return TraceError(fmt.Errorf("only one mandatory flag allowed: %s", allChoices))
			}
		}

		if notDefined.Len() > 0 {
			return TraceError(fmt.Errorf("mandatory flag not defined: %s", notDefined.String()))
		}
	}

	return nil
}

func installService() error {
	args := os.Args[1:]

	for _, item := range []string{FlagNameService, FlagNameServiceUsername, FlagNameServicePassword} {
		for i := range args {
			if args[i] == "-"+item {
				args = append(args[:i], args[i+2:]...)
				break
			}

			if strings.HasPrefix(args[i], "-"+item) {
				args = append(args[:i], args[i+1:]...)
				break
			}
		}
	}

	if len(args) > 0 {
		app.ServiceConfig.Arguments = args
	}

	if *FlagServiceUser != "" {
		app.ServiceConfig.UserName = *FlagServiceUser
	}

	if *FlagServicePassword != "" {
		option := service.KeyValue{}
		option["Password"] = *FlagServicePassword

		app.ServiceConfig.Option = option
	}

	if *FlagService != "" && *FlagService != SERVICE_SIMULATE {
		if *FlagService == SERVICE_UNINSTALL {
			status, err := app.Service.Status()
			if Error(err) {
				return err
			}

			if status == service.StatusRunning {
				err = service.Control(app.Service, SERVICE_STOP)
				if Error(err) {
					return err
				}
			}
		}

		err := service.Control(app.Service, *FlagService)
		if Error(err) {
			return err
		}

		switch *FlagService {
		case SERVICE_INSTALL:
			Info(fmt.Sprintf("Service %s successfully installed", app.ServiceConfig.Name))
		case SERVICE_UNINSTALL:
			Info(fmt.Sprintf("Service %s successfully uninstalled", app.ServiceConfig.Name))
		case SERVICE_START:
			Info(fmt.Sprintf("Service %s successfully started", app.ServiceConfig.Name))
		case SERVICE_STOP:
			Info(fmt.Sprintf("Service %s successfully stopped", app.ServiceConfig.Name))
		case SERVICE_RESTART:
			Info(fmt.Sprintf("Service %s successfully restarted", app.ServiceConfig.Name))
		default:
			return fmt.Errorf("unknown service action: %s", *FlagService)
		}

		return &ErrExit{}
	}

	return nil
}

func Run(mandatoryFlags []string) {
	defer done()

	run := func() error {
		flag.Parse()

		Events.Emit(EventFlagsParsed{}, false)

		if !*FlagNoBanner && !*FlagUsageMd {
			showBanner()
		}

		if flag.NArg() > 0 {
			return TraceError(fmt.Errorf("superfluous flags provided: %s", strings.Join(os.Args[1:], " ")))
		}

		err := usage()
		if Error(err) {
			return err
		}

		err = initConfiguration()
		if Error(err) {
			return err
		}

		err = initLog()
		if Error(err) {
			return err
		}

		err = checkMandatoryFlags(mandatoryFlags)
		if Error(err) {
			return err
		}

		err = installService()
		if Error(err) {
			return err
		}

		// run as real OS daemon

		if !IsRunningInteractive() {
			err := app.Service.Run()
			if Error(err) {
				return err
			}

			return nil
		}

		// simple app or simulated service

		signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

		err = app.applicationLoop()
		if Error(err) {
			return err
		}

		return nil
	}

	err := run()
	if err != nil && !IsErrExit(err) {
		Error(err)
	}
}

func ExitOrError(err error) error {
	if err != nil {
		return err
	} else {
		return &ErrExit{}
	}
}

func showBanner() {
	onceBanner.Do(func() {
		if app != nil {
			date := strconv.Itoa(time.Now().Year())

			if app.Date != date {
				date = app.Date + "-" + date
			}

			fmt.Printf("\n")
			fmt.Printf("%s %s - %s\n", strings.ToUpper(app.Title), app.Version, app.Description)
			fmt.Printf("\n")
			fmt.Printf("Copyright: © %s %s\n", date, app.Developer)
			fmt.Printf("Homepage:  %s\n", app.Homepage)
			fmt.Printf("License:   %s\n", app.License)
			if app.Build != "" {
				fmt.Printf("Build:     %s\n", app.Build)
			}
			if app.Git != "" {
				fmt.Printf("Git:       %s\n", app.Git)
			}
			if !app.Time.IsZero() {
				fmt.Printf("Time:      %s\n", app.Time.Format(time.RFC822))
			}
			fmt.Printf("\n")
		}
	})
}

func nextTicker() *time.Ticker {
	tickerSleep := time.Second

	if isFirstTicker {
		tickerSleep = time.Millisecond
	} else {
		if *FlagAppTicker > 0 {
			nextTick := time.Now().Truncate(MillisecondToDuration(*FlagAppTicker)).Add(MillisecondToDuration(*FlagAppTicker))
			tickerSleep = nextTick.Sub(time.Now())
		}
	}

	newTicker := time.NewTicker(tickerSleep)

	if *FlagAppTicker == 0 {
		newTicker.Stop()
	} else {
		if !isFirstTicker {
			Debug("next tick: %s sleep: %v\n", time.Now().Add(tickerSleep).Truncate(MillisecondToDuration(*FlagAppTicker)).Format(DateTimeMilliMask), tickerSleep)
		}
	}

	isFirstTicker = false

	return newTicker
}

func (app *application) applicationRun() error {
	if IsRunningAsService() {
		Info("Service()")
	} else {
		DebugFunc()
	}

	ticker = nextTicker()

	errCh := make(chan error)

	if app.RunFunc != nil && *FlagAppTicker == 0 {
		go func() {
			defer UnregisterGoRoutine(RegisterGoRoutine(1))

			err := app.RunFunc()

			errCh <- err
		}()
	}

	lifecycleCh := appLifecycle.NewChannel()
	defer appLifecycle.RemoveChannel(lifecycleCh)

	restart = false

	for {
		select {
		//case <-time.After(time.Second):
		//	Info("Restart on time %d", runtime.NumGoroutine())
		//	fmt.Printf("Restart on time %d\n", runtime.NumGoroutine())
		//	restart = true
		//	return nil
		case err := <-errCh:
			return err
		case <-lifecycleCh:
			Info("Stop on request")
			return nil
		case <-restartCh:
			Info("Restart on request")
			restart = true
			return nil
		case <-ctrlC:
			Info("")
			Info("Terminate: CTRL-C pressed")
			return nil
		case <-ticker.C:
			Debug("ticker!")

			ticker.Stop()

			err := app.RunFunc()
			if Error(err) {
				return err
			}

			ticker = nextTicker()
		}
	}
}

func (app *application) Start(s service.Service) error {
	if IsRunningAsService() {
		Info("Start()")
	} else {
		DebugFunc()
	}

	if !IsRunningInteractive() {
		go func() {
			defer UnregisterGoRoutine(RegisterGoRoutine(1))

			Error(app.applicationLoop())
		}()
	} else {
		return app.applicationLoop()
	}

	return nil
}

func (app *application) applicationLoop() error {
	DebugFunc()

	for {
		appLifecycle.Set()

		if app.StartFunc != nil {
			err := app.StartFunc()
			if Error(err) {
				return err
			}
		}

		Error(app.applicationRun())

		if !restart {
			break
		}

		if app.StopFunc != nil {
			err := app.StopFunc()
			if Error(err) {
				return err
			}
		}

		err := initConfiguration()
		if Error(err) {
			return err
		}
	}

	appLifecycle.Unset()

	if app.StopFunc != nil {
		err := app.StopFunc()
		if Error(err) {
			return err
		}
	}

	return nil
}

func AppLifecycle() *Notice {
	return appLifecycle
}

func (app *application) Stop(s service.Service) error {
	if IsRunningAsService() {
		Info("Stop()")
	} else {
		DebugFunc()
	}

	appLifecycle.Unset()

	if ticker != nil {
		ticker.Stop()
	}

	return nil
}

func AppRestart() {
	DebugFunc()

	restartCh <- struct{}{}
}

func IsRunningAsService() bool {
	onceRunningAsService.Do(func() {
		isFlagServiceSimulated := false

		flag.Visit(func(f *flag.Flag) {
			isFlagServiceSimulated = isFlagServiceSimulated || (f.Name == FlagNameService && f.Value.String() == SERVICE_SIMULATE)
		})

		runningAsService = !service.Interactive() || isFlagServiceSimulated
	})

	DebugFunc(runningAsService)

	return runningAsService
}

func IsRunningAsExecutable() bool {
	onceRunningAsExecutable.Do(func() {
		path, err := os.Executable()
		if err != nil {
			path = os.Args[0]
		}

		path = strings.ToLower(path)

		runningAsExecutable = !strings.Contains(path, "temp") && !strings.Contains(path, "tmp")
	})

	DebugFunc(runningAsExecutable)

	return runningAsExecutable
}

func IsRunningInteractive() bool {
	onceRunningInteractive.Do(func() {
		runningInteractive = service.Interactive()
	})

	DebugFunc(runningInteractive)

	return runningInteractive
}

func App() *application {
	return app
}

func done() {
	onceShutdownHooks.Do(func() {
		Events.Emit(EventShutdown{}, true)
	})
}

func AppFilename(newExt string) string {
	filename := Title()
	ext := filepath.Ext(filename)

	if len(ext) > 0 {
		filename = string(filename[:len(filename)-len(ext)])
	}

	return filename + newExt
}

func Title() string {
	onceTitle.Do(func() {
		path, err := os.Executable()
		if err != nil {
			path = os.Args[0]
		}

		path = filepath.Base(path)
		path = path[0:(len(path) - len(filepath.Ext(path)))]

		title = ""

		runes := []rune(path)
		for i := 0; i < len(runes); i++ {
			if string(runes[i]) == "-" {
				break
			}

			if unicode.IsLetter(runes[i]) {
				title = title + string(runes[i])
			}
		}

		DebugFunc(title)
	})

	return title
}

func Version(major bool, minor bool, patch bool) string {
	if strings.Count(app.Version, ".") == 2 {
		s := strings.Split(app.Version, ".")

		sb := strings.Builder{}

		if major {
			sb.WriteString(s[0])
		}

		if minor {
			if sb.Len() > 0 {
				sb.WriteString(".")
			}

			sb.WriteString(s[1])
		}

		if patch {
			if sb.Len() > 0 {
				sb.WriteString(".")
			}

			sb.WriteString(s[2])
		}

		return sb.String()
	}

	return ""
}

func TitleVersion(major bool, minor bool, patch bool) string {
	return Title() + "-" + Version(major, minor, patch)
}
