package common

import (
	"embed"
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"os"
	"os/signal"
	"path/filepath"
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

//Info information of the application
type application struct {
	// IsService
	CanRunAsService bool
	// Name of the application
	Name string
	// Version of the application
	Version string
	// Git label development
	Git string
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
	//TickTime
	RunTime time.Duration
	//Service
	Service service.Service
	//ServiceConfig
	ServiceConfig *service.Config
}

type EventFlagsParsed struct {
}

type EventFlagsSet struct {
}

type EventRestart struct {
}

type ErrExit struct {
}

func (e *ErrExit) Error() string { return "" }

const (
	FlagNameService         = "service"
	FlagNameServiceUsername = "service.username"
	FlagNameServicePassword = "service.password"
	FlagNameServiceTimeout  = "service.timeout"
	FlagNameUsage           = "?"
	FlagNameNoBanner        = "nb"
)

var (
	app                     *application
	FlagService             *string
	FlagServiceUser         *string
	FlagServicePassword     *string
	FlagServiceStartTimeout *int
	FlagUsage               *bool
	FlagNoBanner            *bool
	ticker                  *time.Ticker
	appLifecycle            = NewNotice()
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
	shutdownHooks           = make([]func(), 0)
	restartCh               = make(chan struct{})
	ctrlC                   = make(chan os.Signal, 1)
)

func init() {
	FlagService = new(string)
	FlagServiceUser = new(string)
	FlagServicePassword = new(string)
	FlagServiceStartTimeout = new(int)
	FlagUsage = flag.Bool(FlagNameUsage, false, "show flags description and usage")
	FlagNoBanner = flag.Bool(FlagNameNoBanner, false, "no copyright banner")
}

func Init(isService bool, version string, git string, date string, description string, developer string, homepage string, license string, resources *embed.FS, startFunc func() error, stopFunc func() error, runFunc func() error, runTime time.Duration) {
	app = &application{
		CanRunAsService: isService,
		Name:            Title(),
		Version:         version,
		Git:             git,
		Date:            date,
		Description:     description,
		Developer:       developer,
		License:         license,
		Homepage:        homepage,
		Resources:       resources,
		StartFunc:       startFunc,
		StopFunc:        stopFunc,
		RunFunc:         runFunc,
		RunTime:         runTime,
	}

	executable, err := os.Executable()
	Panic(err)

	app.ServiceConfig = &service.Config{
		Name:             Eval(IsWindowsOS(), Capitalize(app.Name), app.Name).(string),
		DisplayName:      Eval(IsWindowsOS(), Capitalize(app.Name), app.Name).(string),
		Description:      Capitalize(app.Description),
		WorkingDirectory: filepath.Dir(executable),
	}

	app.Service, err = service.New(app, app.ServiceConfig)
	Panic(err)
}

func Run(mandatoryFlags []string) {
	if app.CanRunAsService {
		FlagService = flag.String(FlagNameService, "", "Service operation ("+strings.Join([]string{SERVICE_SIMULATE, SERVICE_START, SERVICE_STOP, SERVICE_RESTART, SERVICE_INSTALL, SERVICE_UNINSTALL}, ",")+")")
		FlagServiceUser = flag.String(FlagNameServiceUsername, "", "Service user")
		FlagServicePassword = flag.String(FlagNameServicePassword, "", "Service password")
		FlagServiceStartTimeout = flag.Int(FlagNameServiceTimeout, 1000, "Service timeout")
	}

	flag.Parse()

	flag.VisitAll(func(f *flag.Flag) {
		envName := strings.ReplaceAll(fmt.Sprintf("%s.%s", Title(), f.Name), ".", "_")
		envValue := strings.ToLower(os.Getenv(envName))
		if envValue == "" {
			envValue = strings.ToUpper(os.Getenv(envName))
		}

		if envValue != "" {
			fl := flag.Lookup(f.Name)
			if fl != nil && fl.Value.String() == fl.DefValue {
				Error(flag.Set(f.Name, envValue))
			}
		}
	})

	Events.Emit(EventFlagsParsed{})

	if !*FlagNoBanner || *FlagUsage {
		showBanner()
	}

	Panic(InitConfiguration())

	initLog()

	if flag.NArg() > 0 {
		Panic(fmt.Errorf("superfluous flags provided: %s", strings.Join(os.Args[1:], " ")))
	}

	flag.VisitAll(func(fl *flag.Flag) {
		v := fmt.Sprintf("%+v", fl.Value)
		if strings.Contains(strings.ToLower(fl.Name), "password") {
			v = strings.Repeat("X", len(v))
		}

		Debug("flag %s = %+v", fl.Name, v)
	})

	if *FlagUsage {
		flag.Usage()
		Exit(0)
	}

	if mandatoryFlags != nil {
		for _, mf := range mandatoryFlags {
			alreadyOne := false

			choices := strings.Split(mf, "|")
			for i := 0; i < len(choices); i++ {
				choices[i] = "\"-" + choices[i] + "\""
			}

			allChoices := strings.Join(choices, " or ")

			for _, flagName := range strings.Split(mf, "|") {
				fl := flag.Lookup(flagName)

				if fl == nil {
					Panic(fmt.Errorf("unknown mandatory flag: \"%s\"", flagName))
				}

				if alreadyOne && fl.Value.String() != "" {
					Panic(fmt.Errorf("only one mandatory flag allowed: %s", allChoices))
				}

				alreadyOne = alreadyOne || fl.Value.String() != ""
			}

			if !alreadyOne {
				Panic(fmt.Errorf("none mandatory flags is defined: %s", allChoices))
			}
		}
	}

	err := run()

	if err == nil || IsErrExit(err) {
		Exit(0)
	} else {
		Panic(err)
	}
}

func Exit(code int) {
	Done()

	os.Exit(code)
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
			fmt.Printf("%s %s %s\n", strings.ToUpper(app.Name), app.Version, app.Description)
			fmt.Printf("\n")
			fmt.Printf("Copyright: © %s %s\n", date, app.Developer)
			fmt.Printf("Homepage:  %s\n", app.Homepage)
			fmt.Printf("License:   %s\n", app.License)
			if app.Git != "" {
				fmt.Printf("Git:       %s\n", app.Git)
			}
			fmt.Printf("\n")
		}
	})
}

func nextTicker() *time.Ticker {
	tickerSleep := time.Second

	if app.RunTime > 0 {
		nextTick := time.Now().Truncate(app.RunTime).Add(app.RunTime)
		tickerSleep = nextTick.Sub(time.Now())
	}

	newTicker := time.NewTicker(tickerSleep)

	if app.RunTime == 0 {
		newTicker.Stop()
	} else {
		Debug("next tick: %s sleep: %v\n", time.Now().Add(tickerSleep).Truncate(app.RunTime).Format(DateTimeMilliMask), tickerSleep)
	}

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

	if app.RunFunc != nil && app.RunTime == 0 {
		go func() {
			err := app.RunFunc()

			errCh <- err
		}()
	}

	lifecycleCh := appLifecycle.NewChannel()

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

			Error(app.RunFunc())

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

	appLifecycle.Set()

	if !IsRunningInteractive() {
		go func() {
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
		if app.StartFunc != nil {
			err := app.StartFunc()
			if Error(err) {
				return err
			}
		}

		if AppLifecycle().IsSet() {
			Error(app.applicationRun())
		}

		if !restart {
			break
		}

		if app.StopFunc != nil {
			err := app.StopFunc()
			if Error(err) {
				return err
			}
		}

		err := InitConfiguration()
		if Error(err) {
			return err
		}

		Events.Emit(EventRestart{})
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

	if app.StopFunc != nil {
		err := app.StopFunc()
		if Error(err) {
			return err
		}
	}

	return nil
}

func run() error {
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
			return nil
		case SERVICE_UNINSTALL:
			Info(fmt.Sprintf("Service %s successfully uninstalled", app.ServiceConfig.Name))
			return nil
		case SERVICE_START:
			Info(fmt.Sprintf("Service %s successfully started", app.ServiceConfig.Name))
			return nil
		case SERVICE_STOP:
			Info(fmt.Sprintf("Service %s successfully stopped", app.ServiceConfig.Name))
			return nil
		case SERVICE_RESTART:
			Info(fmt.Sprintf("Service %s successfully restarted", app.ServiceConfig.Name))
			return nil
		default:
			return fmt.Errorf("unknown service action: %s", *FlagService)
		}
	}

	// run as real OS daemon

	if !IsRunningInteractive() {
		return app.Service.Run()
	}

	// simple app or simulated service

	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	startErr := app.Start(app.Service)
	Error(startErr)

	stopErr := app.Stop(app.Service)
	if Error(stopErr) && startErr == nil {
		startErr = stopErr
	}

	return startErr
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

		runningAsExecutable = !strings.HasPrefix(path, os.TempDir())
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

func Done() {
	onceShutdownHooks.Do(func() {
		for _, f := range shutdownHooks {
			f()
		}
	})

	closeLogFile()
}

func AddShutdownHook(f func()) {
	shutdownHooks = append(shutdownHooks, nil)
	copy(shutdownHooks[1:], shutdownHooks[0:])
	shutdownHooks[0] = f
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
		for i := len(runes) - 1; i >= 0; i-- {
			if unicode.IsLetter(runes[i]) {
				title = string(runes[i]) + title
			} else {
				break
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
