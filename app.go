package common

import (
	"bufio"
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
)

const (
	APACHE string = "https://www.apache.org/licenses/LICENSE-2.0.html"
	GPL2   string = "https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html"
)

//Info information of the application
type application struct {
	// Name of the application
	Name string
	// Version of the application
	Version string
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
	//StartFunc
	StartFunc func() error
	//StopFunc
	StopFunc func() error
	//TickFunc
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

const (
	SERVICE          = "service"
	SERVICE_USERNAME = "service.username"
	SERVICE_PASSWORD = "service.password"
	SERVICE_TIMEOUT  = "service.timeout"
)

type goTesting interface {
	Logf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

var (
	app                     *application
	FlagService             *string
	FlagServiceUser         *string
	FlagServicePassword     *string
	FlagServiceStartTimeout *int
	serviceActions          []string
	FlagUsage               *bool
	NoBanner                bool
	ticker                  *time.Ticker
	appLifecycle            = NewNotice()
	onceBanner              sync.Once
	onceDone                sync.Once
	restart                 bool
	restartCh               = make(chan struct{})
	kbCh                    = make(chan struct{})
	ctrlC                   = make(chan os.Signal, 1)
	gotest                  goTesting
)

func init() {
	app = &application{}

	FlagService = new(string)
	serviceActions = service.ControlAction[:]
	serviceActions = append(serviceActions, "simulate")
	FlagUsage = flag.Bool("?", false, "show usage")
}

func Init(version string, date string, description string, developer string, homepage string, license string, startFunc func() error, stopFunc func() error, runFunc func() error, runTime time.Duration) {
	app.Name = Title()
	app.Version = version
	app.Date = date
	app.Description = description
	app.Developer = developer
	app.License = license
	app.Homepage = homepage
	app.StartFunc = startFunc
	app.StopFunc = stopFunc
	app.RunFunc = runFunc
	app.RunTime = runTime
}

func InitTesting(v goTesting) {
	gotest = v
}

// Run struct for copyright information
func Run(mandatoryFlags []string) {
	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	if app.StopFunc != nil {
		FlagService = flag.String(SERVICE, "", "Service operation ("+strings.Join(serviceActions, ",")+")")
		FlagServiceUser = flag.String(SERVICE_USERNAME, "", "Service user")
		FlagServicePassword = flag.String(SERVICE_PASSWORD, "", "Service password")
		FlagServiceStartTimeout = flag.Int(SERVICE_TIMEOUT, 500, "Server start timeout")
	}

	flag.Parse()

	Events.Emit(EventFlagsParsed{})

	err := initConfiguration()
	if err != nil {
		Fatal(err)

		Exit(1)
	}

	initLog()

	flag.VisitAll(func(fl *flag.Flag) {
		if fl.Value.String() != "" && fl.Value.String() != fl.DefValue {
			v := fmt.Sprintf("%+v", fl.Value)
			if strings.Index(strings.ToLower(fl.Name), "password") != -1 {
				v = strings.Repeat("X", len(v))
			}

			Debug("flag %s = %+v", fl.Name, v)
		}
	})

	if !NoBanner || *FlagUsage {
		showBanner()
	}

	if *FlagUsage {
		flag.Usage()
		Exit(0)
	}

	flagErr := false

	if *FlagService == "" || *FlagService == "install" {
		for _, f := range mandatoryFlags {
			fl := flag.Lookup(f)

			if fl == nil {
				showBanner()
				Error(fmt.Errorf("unknown mandatory flag: %s", f))

				flagErr = true

				continue
			}

			if len(fl.Value.String()) == 0 {
				showBanner()
				Error(fmt.Errorf("mandatory flag needed: \"-%s\" - %s", fl.Name, fl.Usage))

				flagErr = true
			}
		}
	}

	if flagErr {
		Exit(1)
	}

	if IsRunningInteractive() {
		go func() {
			r := bufio.NewReader(os.Stdin)

			var s string

			for len(s) == 0 {
				s, _ = r.ReadString('\n')
			}

			kbCh <- struct{}{}
		}()
	}

	err = run()

	if err == nil || isErrExit(err) {
		Exit(0)
	} else {
		Exit(1)
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
			fmt.Printf("\n")
		}
	})
}

func (app *application) service() error {
	if IsRunningAsService() {
		Info("Service()")
	} else {
		DebugFunc()
	}

	sleep := time.Second

	if app.RunTime > 0 {
		nextTick := time.Now().Truncate(app.RunTime).Add(app.RunTime)
		sleep = nextTick.Sub(time.Now())
	}

	ticker = time.NewTicker(sleep)

	if app.RunTime == 0 {
		ticker.Stop()
	}

	info := func() {
		if app.RunTime > 0 {
			Debug("next tick: %s\n", time.Now().Add(sleep).Truncate(app.RunTime).Format(DateTimeMilliMask))
			Debug("sleep for %v ...", sleep)
		}
	}

	info()

	restart = false

	for {
		select {
		//case <-time.After(time.Second):
		//	Info("Restart on time %d", runtime.NumGoroutine())
		//	fmt.Printf("Restart on time %d\n", runtime.NumGoroutine())
		//	restart = true
		//	return nil
		case <-appLifecycle.Channel():
			Info("Stop on request")
			return nil
		case <-restartCh:
			Info("Restart on request")
			restart = true
			return nil
		case <-kbCh:
			Info("Terminate: keyboard ENTER key pressed")
			return nil
		case <-ctrlC:
			Info("Terminate: CTRL-C pressed")
			return nil
		case <-ticker.C:
			Debug("ticker!")

			ticker.Stop()

			Error(app.RunFunc())

			ti := time.Now()
			ti = ti.Add(app.RunTime)
			ti = TruncateTime(ti, Second)

			sleep = ti.Sub(time.Now())
			ticker = time.NewTicker(sleep)

			info()
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

	if app.StartFunc != nil {
		err := app.StartFunc()
		if Error(err) {
			return err
		}
	}

	return nil
}

func (app *application) loop() error {
	DebugFunc()

	for {
		Error(app.service())

		if restart {
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

			Events.Emit(EventAppRestart{})

			if app.StartFunc != nil {
				err := app.StartFunc()
				if Error(err) {
					return err
				}
			}
		} else {
			break
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

	if app.StopFunc != nil {
		err := app.StopFunc()
		if Error(err) {
			return err
		}
	}

	return nil
}

func run() error {
	if IsRunningAsService() {
		executable, err := os.Executable()
		if err != nil {
			return err
		}

		app.ServiceConfig = &service.Config{
			Name:             Eval(IsWindowsOS(), Capitalize(app.Name), app.Name).(string),
			DisplayName:      Eval(IsWindowsOS(), Capitalize(app.Name), app.Name).(string),
			Description:      Capitalize(app.Description),
			WorkingDirectory: filepath.Dir(executable),
		}

		app.Service, err = service.New(app, app.ServiceConfig)
		if err != nil {
			return err
		}

		if *FlagService != "" && *FlagService != "simulate" {
			args := os.Args[1:]

			for _, item := range []string{SERVICE, SERVICE_USERNAME, SERVICE_PASSWORD} {
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

			if IndexOf(serviceActions, *FlagService) == -1 {
				return fmt.Errorf("invalid service action: %s", *FlagService)
			}

			if *FlagService == "uninstall" {
				status, err := app.Service.Status()
				if err != nil {
					return err
				}

				if status == service.StatusRunning {
					err = service.Control(app.Service, "stop")
					if err != nil {
						return err
					}
				}
			}

			err = service.Control(app.Service, *FlagService)
			if err != nil {
				return err
			}

			switch *FlagService {
			case "install":
				Info(fmt.Sprintf("Service %s successfully installed", app.ServiceConfig.Name))
				return nil
			case "uninstall":
				Info(fmt.Sprintf("Service %s successfully uninstalled", app.ServiceConfig.Name))
				return nil
			case "start":
				Info(fmt.Sprintf("Service %s successfully started", app.ServiceConfig.Name))
				return nil
			case "stop":
				Info(fmt.Sprintf("Service %s successfully stopped", app.ServiceConfig.Name))
				return nil
			case "restart":
				Info(fmt.Sprintf("Service %s successfully restarted", app.ServiceConfig.Name))
				return nil
			}

			return nil
		}
	}

	// run as service

	if IsRunningAsService() {
		if IsRunningInteractive() {
			// simulated service

			err := app.Start(app.Service)
			Error(err)

			if err == nil {
				err = app.loop()
				Error(err)
			}

			stopErr := app.Stop(app.Service)
			if Error(stopErr) && err == nil {
				err = stopErr
			}

			return err
		} else {
			go func() {
				Error(app.loop())
			}()

			// OS service

			return app.Service.Run()
		}
	} else {
		err := app.Start(app.Service)
		if Error(err) {
			return err
		}

		if app.RunFunc != nil {
			go func() {
				err = app.RunFunc()
				appLifecycle.Unset()
			}()

			select {
			case <-appLifecycle.Channel():
			case <-ctrlC:
				Info("Terminate: CTRL-C pressed")
				appLifecycle.Unset()
			}
		}

		err = app.Stop(app.Service)
		if Error(err) {
			return err
		}

		return nil
	}
}

func AppRestart() {
	DebugFunc()

	go func() {
		time.Sleep(time.Second)

		restartCh <- struct{}{}
	}()
}

func IsRunningAsService() bool {
	b := !IsRunningInteractive() || *FlagService == "simulate"

	DebugFunc("%v", b)

	return b
}

func IsRunningAsExecutable() bool {
	path, err := os.Executable()
	if err != nil {
		path = os.Args[0]
	}

	b := !strings.HasPrefix(path, os.TempDir())

	DebugFunc("%v", b)

	return b
}

func IsRunningInteractive() bool {
	return service.Interactive()
}

func App() *application {
	return app
}
