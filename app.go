package common

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-ini/ini"

	"github.com/kardianos/service"
)

const (
	// Apache license
	APACHE string = "https://www.apache.org/licenses/LICENSE-2.0.html"
	// GPL2 license
	GPL2 string = "https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html"
)

//Info information of the application
type App struct {
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
	//IsService
	IsService bool
	//PrepareFunc
	PrepareFunc func() error
	//StartFunc
	StartFunc func() error
	//StopFunc
	StopFunc func() error
	//TickFunc
	TickFunc func() error
	//TickTime
	TickTime time.Duration
}

const (
	SERVICE          = "service"
	SERVICE_USER     = "service-user"
	SERVICE_PASSWORD = "service-password"
)

var (
	app                 *App
	serviceFlag         *string
	serviceUser         *string
	servicePassword     *string
	serviceActions      []string
	serviceStartTimeout *int
	usage               *bool
	NoBanner            bool
	ticker              *time.Ticker
	profile             *string
	stopped             bool
)

func init() {
	serviceFlag = new(string)
	serviceActions = service.ControlAction[:]
	serviceActions = append(serviceActions, "simulate")
	usage = flag.Bool("?", false, "show usage")
	profile = flag.String("profile", "", "flag profile in configuration logFile")
}

// New struct for copyright information
func New(application *App, mandatoryFlags []string) {
	app = application

	if app.IsService {
		serviceFlag = flag.String(SERVICE, "", "Service operation ("+strings.Join(serviceActions, ",")+")")
		serviceUser = flag.String(SERVICE_USER, "", "Service user")
		servicePassword = flag.String(SERVICE_PASSWORD, "", "Service password")
		serviceStartTimeout = flag.Int("service-timeout", 1000, "Server start timeout")
	}

	flag.Parse()

	parseCfgFile()

	if app.PrepareFunc != nil {
		err := app.PrepareFunc()

		if err != nil {
			Fatal(err)
		}
	}

	if !NoBanner || *usage {
		ShowBanner()
	}

	if *usage {
		flag.Usage()
		Exit(0)
	}

	flagErr := false

	if *serviceFlag == "" || *serviceFlag == "install" {
		for _, f := range mandatoryFlags {
			fl := flag.Lookup(f)

			if fl == nil {
				Error(fmt.Errorf("unknown mandatory flag: %s", f))

				flagErr = true

				continue
			}

			if len(fl.Value.String()) == 0 {
				Error(fmt.Errorf("mandatory flag needed: \"-%s\" - %s", fl.Name, fl.Usage))

				flagErr = true
			}
		}
	}

	if flagErr {
		Exit(1)
	}
}

func GetApp() *App {
	return app
}

func Test() {
	New(&App{"test", "0.0.0", "2018", "test", "mpetavy", APACHE, "https://github.com/golang/mpetavy/golang/tresor", true, nil, nil, nil, nil, time.Duration(0)}, nil)
}

func parseCfgFile() {
	fn := app.Name + ".cfg"

	f, err := ini.Load(fn)
	if err == nil {
		for _, k := range f.Section(*profile).Keys() {
			name := strings.ToLower(k.Name())
			value := k.String()

			p := strings.Index(name, "@")
			system := runtime.GOOS

			if p != -1 {
				system = name[p+1:]
				name = name[:p]
			}

			if runtime.GOOS == system {
				if flag.Lookup("-"+name) == nil {
					err := flag.Set(name, value)
					if err != nil {
						Error(err)
					}
				}
			}
		}
	}
}

func Exit(code int) {
	runShutdownHooks()
	os.Exit(code)
}

func ShowBanner() {
	if app != nil {
		date := strconv.Itoa(time.Now().Year())

		if app.Date != date {
			date = app.Date + "-" + date
		}

		fmt.Printf("\n")
		fmt.Printf("%s %s - %s\n", strings.ToUpper(app.Name), app.Version, app.Description)
		fmt.Printf("\n")
		fmt.Printf("Copyright: © %s %s\n", date, app.Developer)
		fmt.Printf("Homepage:  %s\n", app.Homepage)
		fmt.Printf("License:   %s\n", app.License)
		fmt.Printf("\n")
	}
}

func (app *App) service() error {
	DebugFunc()

	sleep := time.Duration(1000000) * time.Hour

	if app.TickTime > 0 {
		nextTick := time.Now().Truncate(app.TickTime).Add(app.TickTime)
		sleep = nextTick.Sub(time.Now())
	}
	ticker = time.NewTicker(sleep)

	info := func() {
		if app.TickTime > 0 {
			Debug("next tick: %s\n", time.Now().Add(sleep).Truncate(app.TickTime).Format(DateTimeMilliMask))
			Debug("sleep for %v ...", sleep)
		}
	}

	info()

	ctrlC := make(chan os.Signal, 1)
	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	kbCh := make(chan struct{})

	if service.Interactive() {
		go func() {
			r := bufio.NewReader(os.Stdin)

			var s string

			for len(s) == 0 {
				s, _ = r.ReadString('\n')
			}

			kbCh <- struct{}{}
		}()
	}

	for {
		select {
		case <-kbCh:
			Info("Terminate: keyboard ENTER key pressed")
			return nil
		case <-ctrlC:
			Info("Terminate: CTRL-C pressed")
			return nil
		case <-ticker.C:
			Debug("ticker!")

			ticker.Stop()

			if err := app.TickFunc(); err != nil {
				Error(err)
			}

			sleep = app.TickTime
			ticker = time.NewTicker(app.TickTime)

			info()
		}
	}
}

func (app *App) Start(s service.Service) error {
	DebugFunc()

	if app.StartFunc != nil {
		if err := app.StartFunc(); err != nil {
			return err
		}
	}

	var err error

	if !service.Interactive() {
		go func(err *error) {
			*err = app.service()
		}(&err)

		time.Sleep(time.Duration(*serviceStartTimeout) * time.Millisecond)
	}

	return err
}

func (app *App) Stopped() bool {
	return stopped
}

func (app *App) Stop(s service.Service) error {
	DebugFunc()

	stopped = true

	if ticker != nil {
		ticker.Stop()
	}

	if app.StopFunc != nil {
		if err := app.StopFunc(); err != nil {
			return err
		}
	}

	return nil
}

func run() error {
	svcConfig := &service.Config{
		Name:        Eval(IsWindowsOS(), Capitalize(app.Name), app.Name).(string),
		DisplayName: Eval(IsWindowsOS(), Capitalize(app.Name), app.Name).(string),
		Description: app.Description,
	}

	s, err := service.New(app, svcConfig)
	if err != nil {
		return err
	}

	if *serviceFlag != "" && *serviceFlag != "simulate" {
		args := os.Args[1:]

		for _, item := range []string{SERVICE, SERVICE_USER, SERVICE_PASSWORD} {
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
			svcConfig.Arguments = args
		}

		if *serviceUser != "" {
			svcConfig.UserName = *serviceUser
		}

		if *servicePassword != "" {
			option := service.KeyValue{}
			option["Password"] = *servicePassword

			svcConfig.Option = option
		}

		i, err := IndexOf(serviceActions, *serviceFlag)
		if err != nil {
			return err
		}

		if i == -1 {
			return fmt.Errorf("invalid service action: %s", *serviceFlag)
		}

		if *serviceFlag == "uninstall" {
			status, err := s.Status()
			if err != nil {
				return err
			}

			if status == service.StatusRunning {
				err = service.Control(s, "stop")
				if err != nil {
					return err
				}
			}
		}

		err = service.Control(s, *serviceFlag)
		if err != nil {
			return err
		}

		switch *serviceFlag {
		case "install":
			Info(fmt.Sprintf("Service %s successfully installed", svcConfig.Name))
			return nil
		case "uninstall":
			Info(fmt.Sprintf("Service %s successfully uninstalled", svcConfig.Name))
			return nil
		case "start":
			Info(fmt.Sprintf("Service %s successfully started", svcConfig.Name))
			return nil
		case "stop":
			Info(fmt.Sprintf("Service %s successfully stopped", svcConfig.Name))
			return nil
		case "restart":
			Info(fmt.Sprintf("Service %s successfully restarted", svcConfig.Name))
			return nil
		}

		return nil
	}

	// run as service

	if app.IsService && !service.Interactive() {
		return s.Run()
	}

	// run as app

	if err := app.Start(s); err != nil {
		return err
	}

	if app.TickFunc != nil {
		if err := app.TickFunc(); err != nil {
			return err
		}
	}

	if app.IsService && (!service.Interactive() || *serviceFlag == "simulate") {
		if err := app.service(); err != nil {
			return err
		}
	}

	if err := app.Stop(s); err != nil {
		return err
	}

	return nil
}

func Run() {
	err := run()

	if err != nil {
		Fatal(err)
	}
}
