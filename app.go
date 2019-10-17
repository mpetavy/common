package common

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/kardianos/service"
)

const (
	// Apache license
	APACHE string = "https://www.apache.org/licenses/LICENSE-2.0.html"
	// GPL2 license
	//GPL2 string = "https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html"
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
	SERVICE_USERNAME = "service.username"
	SERVICE_PASSWORD = "service.password"
	SERVICE_TIMEOUT  = "service.timeout"
)

var (
	app                 *App
	serviceFlag         *string
	serviceUser         *string
	servicePassword     *string
	ServiceStartTimeout *int
	serviceActions      []string
	usage               *bool
	NoBanner            bool
	ticker              *time.Ticker
	appDeath            = NewNotice()
	onceBanner          sync.Once
	onceDone            sync.Once
)

func init() {
	app = &App{}

	serviceFlag = new(string)
	serviceActions = service.ControlAction[:]
	serviceActions = append(serviceActions, "simulate")
	usage = flag.Bool("?", false, "show usage")
}

func Init(Version string, Date string, Description string, Developer string, License string, IsService bool, StartFunc func() error, StopFunc func() error, TickFunc func() error, TickTime time.Duration) {
	app.Name = Title()
	app.Version = Version
	app.Date = Date
	app.Description = Description
	app.Developer = Developer
	app.License = License
	app.Homepage = fmt.Sprintf("https://github.com/mpetavy/%s", Title())
	app.IsService = IsService
	app.StartFunc = StartFunc
	app.StopFunc = StopFunc
	app.TickFunc = TickFunc
	app.TickTime = TickTime
}

// Run struct for copyright information
func Run(mandatoryFlags []string) {
	if app.IsService {
		serviceFlag = flag.String(SERVICE, "", "Service operation ("+strings.Join(serviceActions, ",")+")")
		serviceUser = flag.String(SERVICE_USERNAME, "", "Service user")
		servicePassword = flag.String(SERVICE_PASSWORD, "", "Service password")
		ServiceStartTimeout = flag.Int(SERVICE_TIMEOUT, 1000, "Server start timeout")
	}

	flag.Parse()

	err := initConfiguration()
	if err != nil {
		Fatal(err)
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

	if !NoBanner || *usage {
		showBanner()
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

	err = run()
	if err != nil {
		Fatal(err)
	}
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
	path, err := os.Executable()
	if err != nil {
		path = os.Args[0]
	}

	path = filepath.Base(path)
	path = path[0:(len(path) - len(filepath.Ext(path)))]

	runes := []rune(path)
	for len(runes) > 0 && !unicode.IsLetter(runes[0]) {
		runes = runes[1:]
	}

	title := string(runes)

	DebugFunc(title)

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

func Exit(code int) {
	Done()

	os.Exit(code)
}

func ErrExitOrError(err error) error {
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
			fmt.Printf("%s %s - %s\n", strings.ToUpper(app.Name), app.Version, app.Description)
			fmt.Printf("\n")
			fmt.Printf("Copyright: © %s %s\n", date, app.Developer)
			fmt.Printf("Homepage:  %s\n", app.Homepage)
			fmt.Printf("License:   %s\n", app.License)
			fmt.Printf("\n")
		}
	})
}

func (app *App) service() error {
	if IsRunningAsService() {
		Info("Service()")
	}

	sleep := time.Second

	if app.TickTime > 0 {
		nextTick := time.Now().Truncate(app.TickTime).Add(app.TickTime)
		sleep = nextTick.Sub(time.Now())
	}

	ticker = time.NewTicker(sleep)

	if app.TickTime == 0 {
		ticker.Stop()
	}

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

			ti := time.Now()
			ti = ti.Add(app.TickTime)
			ti = TruncateTime(ti, Second)

			sleep = ti.Sub(time.Now())
			ticker = time.NewTicker(sleep)

			info()
		}
	}
}

func (app *App) Start(s service.Service) error {
	if IsRunningAsService() {
		Info("Start()")
	}

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

		time.Sleep(time.Duration(*ServiceStartTimeout) * time.Millisecond)
	}

	return err
}

func AppDeath() *Notice {
	return appDeath
}

func (app *App) Stop(s service.Service) error {
	if IsRunningAsService() {
		Info("Stop()")
	}

	appDeath.Set()

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

	if IsRunningAsService() {
		if err := app.service(); err != nil {
			return err
		}
	}

	if err := app.Stop(s); err != nil {
		return err
	}

	return nil
}

func IsRunningAsService() bool {
	return app.IsService && (!service.Interactive() || *serviceFlag == "simulate")
}

func IsRunningAsExecutable() bool {
	path, err := os.Executable()
	if err != nil {
		path = os.Args[0]
	}

	return !strings.HasPrefix(path, os.TempDir())
}
