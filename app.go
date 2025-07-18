package common

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"golang.org/x/mod/modfile"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	APACHE string = "https://www.apache.org/licenses/LICENSE-2.0.html"

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
	Date time.Time
	// Build label development
	Build string
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

type EventInit struct {
}

type EventShutdown struct {
}

type EventFlags struct {
}

type ErrExit struct {
	ExitCode int
}

func (e *ErrExit) Error() string { return "" }

const (
	FlagNameAppProduct      = "app.product"
	FlagNameAppTicker       = "app.ticker"
	FlagNameAppStartupDelay = "app.startupdelay"
	FlagNameService         = "service"
	FlagNameServiceUsername = "service.username"
	FlagNameServicePassword = "service.password"
	FlagNameServiceTimeout  = "service.timeout"
	FlagNameScriptTimeout   = "script.timeout"
	FlagNameScriptStart     = "script.start"
	FlagNameScriptStop      = "script.stop"
	FlagNameUsage           = "h"
	FlagNameUsageMd         = "hmd"
	FlagNameUsageAll        = "hs"
	FlagNameUsageAllMd      = "hsmd"
	FlagNameNoBanner        = "nb"
	FlagNameVersion         = "v"
)

var (
	FlagService         = SystemFlagString(FlagNameService, "", "Service operation ("+strings.Join([]string{SERVICE_SIMULATE, SERVICE_START, SERVICE_STOP, SERVICE_RESTART, SERVICE_INSTALL, SERVICE_UNINSTALL}, ",")+")")
	FlagServiceUser     = SystemFlagString(FlagNameServiceUsername, "", "Service user")
	FlagServicePassword = SystemFlagString(FlagNameServicePassword, "", "Service password")
	FlagServiceTimeout  = SystemFlagInt(FlagNameServiceTimeout, 1000, "Service timeout")
	FlagScriptTimeout   = SystemFlagInt(FlagNameScriptTimeout, 5000, "Script timeout")
	FlagScriptStart     *string
	FlagScriptStop      *string
	FlagUsage           = flag.Bool(FlagNameUsage, false, "show flags description and usage")
	FlagUsageMd         = flag.Bool(FlagNameUsageMd, false, "show flags description and usage in markdown format")
	FlagUsageAll        = flag.Bool(FlagNameUsageAll, false, "show flags and system flags description and usage")
	FlagUsageAllMd      = flag.Bool(FlagNameUsageAllMd, false, "show flags and system flags description and usage in markdown format")
	FlagVersion         = flag.Bool(FlagNameVersion, false, "show version")
	FlagNoBanner        = flag.Bool(FlagNameNoBanner, false, "no copyright banner")

	app                     *application
	FlagAppProduct          *string
	FlagAppTicker           *int
	FlagAppStartupDelay     *int
	ticker                  *time.Ticker
	appLifecycle            = NewNotice(true, nil)
	onceBanner              sync.Once
	onceRunningAsContainer  sync.Once
	onceRunningAsService    sync.Once
	onceRunningAsExecutable sync.Once
	onceRunningInteractive  sync.Once
	onceRunningInDev        sync.Once
	onceShutdownHooks       sync.Once
	onceTitle               sync.Once
	runningAsContainer      bool
	runningAsService        bool
	runningAsExecutable     bool
	runningInteractive      bool
	runningInDev            bool
	restart                 bool
	restartCh               chan struct{}
	exitCh                  chan error
	banner                  = bytes.Buffer{}
)

func Init(title string, version string, git string, build string, description string, developer string, homepage string, license string, resources *embed.FS, startFunc func() error, stopFunc func() error, runFunc func() error, runTime time.Duration) {
	Panic(initWorkingPath())

	Events.AddListener(EventInit{}, func(ev Event) {
		FlagScriptStart = SystemFlagString(FlagNameScriptStart, fmt.Sprintf("%s%s-start%s", Eval(IsWindows(), "", "./"), strings.ToLower(Title()), Eval(IsWindows(), ".bat", ".sh")), "Service user")
		FlagScriptStop = SystemFlagString(FlagNameScriptStop, fmt.Sprintf("%s%s-stop%s", Eval(IsWindows(), "", "./"), strings.ToLower(Title()), Eval(IsWindows(), ".bat", ".sh")), "Service user")
	})

	if resources != nil {
		ba, err := resources.ReadFile("go.mod")
		Panic(err)

		mf, err := modfile.Parse("go.mod", ba, nil)
		Panic(err)

		if title == "" {
			title = mf.Module.Mod.String()

			p := strings.LastIndex(title, "/")
			if p != -1 {
				title = title[p+1:]
			}
		}
	}

	if title == "" {
		wd, err := os.Getwd()
		Panic(err)

		title = filepath.Base(wd)
	}

	if developer == "" {
		developer = "mpetavy"
	}

	if homepage == "" {
		homepage = fmt.Sprintf("https://github.com/mpetavy/%s", title)
	}

	if license == "" {
		license = APACHE
	}

	if description == "" {
		description = title
	}

	date := time.Now()

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if git != "" {
					continue
				}

				git = setting.Value
			case "vcs.time":
				date, _ = time.Parse(time.RFC3339, setting.Value)

				if version != "" {
					continue
				}

				d := date.Sub(time.Date(date.Year(), 1, 1, 0, 0, 0, 0, time.UTC))

				version = fmt.Sprintf("%d.%d", (date.Year()-22)%100, int(d.Abs().Hours())/24)
			}
		}
	}

	FlagAppProduct = SystemFlagString(FlagNameAppProduct, title, "app product")
	FlagAppTicker = SystemFlagInt(FlagNameAppTicker, int(runTime.Milliseconds()), "app execution ticker")
	FlagAppStartupDelay = SystemFlagInt(FlagNameAppStartupDelay, 0, "app startup delay")

	app = &application{
		Title:         title,
		Version:       version,
		Git:           git,
		Date:          date,
		Build:         build,
		Description:   description,
		Developer:     developer,
		License:       license,
		Homepage:      homepage,
		Resources:     resources,
		StartFunc:     startFunc,
		StopFunc:      stopFunc,
		RunFunc:       runFunc,
		Service:       nil,
		ServiceConfig: nil,
	}

	executable, err := os.Executable()
	Panic(err)

	app.ServiceConfig = &service.Config{
		Name:             Eval(IsWindows(), Capitalize(app.Title), app.Title),
		DisplayName:      Eval(IsWindows(), Capitalize(app.Title), app.Title),
		Description:      Capitalize(app.Description),
		WorkingDirectory: filepath.Dir(executable),
	}

	app.Service, err = service.New(app, app.ServiceConfig)
	Panic(err)
}

func initWorkingPath() error {
	if !IsRunningAsService() || !IsWindows() {
		return nil
	}

	exe, err := os.Executable()
	if Error(err) {
		return err
	}

	exeDir := filepath.Dir(exe)

	wd, err := os.Getwd()
	if Error(err) {
		return err
	}

	if wd != exeDir {
		Warn("change OS working path: %s", exeDir)

		err := os.Chdir(exeDir)
		if Error(err) {
			return err
		}
	}

	return nil
}

func usage() error {
	if *FlagUsageMd || *FlagUsageAllMd {
		dir, err := os.Getwd()
		if Error(err) {
			return err
		}

		st := NewStringTable()

		st.AddCols("Flag name", "ENV name", "Default value", "Only CmdLine", "Description")

		flag.VisitAll(func(fl *flag.Flag) {
			if !*FlagUsageAllMd && slices.Contains(SystemFlagNames, fl.Name) {
				return
			}

			defValue := fl.DefValue
			if strings.HasPrefix(defValue, dir) {
				defValue = fmt.Sprintf("./%s", defValue[len(dir)+1:])
			}

			onlyCmdLine := ""
			if IsCmdlineOnlyFlag(fl.Name) {
				onlyCmdLine = "*"
			}
			st.AddCols(fl.Name, FlagNameAsEnvName(fl.Name), defValue, onlyCmdLine, fl.Usage)
		})

		fmt.Printf("%s\n", st.Markdown())

		return &ErrExit{}
	}

	if *FlagUsage || *FlagUsageAll {
		fmt.Printf("Usage of %s:\n", strings.ToUpper(Title()))

		flag.VisitAll(func(fl *flag.Flag) {
			if !*FlagUsageAll && slices.Contains(SystemFlagNames, fl.Name) {
				return
			}

			ty := fmt.Sprintf("%T", fl.Value)
			ty = strings.TrimPrefix(ty, "*flag.")
			ty = strings.TrimSuffix(ty, "Value")

			def := ""
			if fl.DefValue != "" {
				def = fmt.Sprintf(" (default '%s')", fl.DefValue)
			}

			fmt.Printf("  -%s (%s)\n", fl.Name, ty)
			fmt.Printf("      %s%s\n", fl.Usage, def)
		})

		return &ErrExit{}
	}

	return nil
}

func IsFlagProvided(flagname string) bool {
	flagname0 := "-" + flagname
	flagname1 := "-" + flagname + "="

	for _, arg := range os.Args {
		if arg == flagname0 || strings.HasPrefix(arg, flagname1) {
			return true
		}
	}

	return false
}

func IsFlagAsEnvProvided(flagname string) bool {
	v := os.Getenv(FlagNameAsEnvName(flagname))

	return v != ""
}

func FlagValue(flagname string) string {
	flagname0 := "-" + flagname
	flagname1 := "-" + flagname + "="

	for i := 0; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == flagname0 {
			value := ""
			if i+1 < len(os.Args) {
				if !strings.HasPrefix(os.Args[i+1], "-") {
					value = os.Args[i+1]
				}
			}

			return value
		}
		if strings.HasPrefix(arg, flagname1) {
			splits := Split(arg, "=")

			if len(splits) > 1 {
				return splits[1]
			}
		}
	}

	return ""
}

func installService() error {
	args := os.Args[1:]

	for _, item := range []string{FlagNameService, FlagNameServiceUsername, FlagNameServicePassword} {
		for i := range args {
			if args[i] == "-"+item {
				args = slices.Delete(args, i, i+2)
				break
			}

			if strings.HasPrefix(args[i], "-"+item) {
				args = slices.Delete(args, i, i+1)
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
	run(mandatoryFlags, nil)
}

func RunTests(m *testing.M) {
	run(nil, m)
}

func run(mandatoryFlags []string, m *testing.M) {
	defer done()

	if app == nil {
		Init("", "", "", "", "", "", "", "", nil, nil, nil, nil, 0)
	}

	Events.Emit(EventInit{}, false)

	err := func() error {
		flag.Parse()

		if !*FlagNoBanner && !*FlagUsageMd {
			showBanner()
		}

		if *FlagVersion {
			return nil
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

		err = initTemp()
		if Error(err) {
			return err
		}

		err = InitLog()
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

		defer func() {
			if FileExists(*FlagScriptStop) {
				_, err := RunScript(MillisecondToDuration(*FlagScriptTimeout), *FlagScriptStop)
				Error(err)
			}
		}()

		if IsRunningAsContainer() && *FlagAppStartupDelay > 0 {
			d := MillisecondToDuration(*FlagAppStartupDelay)

			Debug("Startup delay: %v", d)

			Sleep(d)
		}

		if FileExists(*FlagScriptStart) {
			_, err = RunScript(MillisecondToDuration(*FlagScriptTimeout), *FlagScriptStart)
			if Error(err) {
				return err
			}
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

		ctrlC := make(chan os.Signal, 1)

		// can kill with "kill -SIGINT <pid>"
		signal.Notify(ctrlC, os.Interrupt)

		go func() {
			defer UnregisterGoRoutine(RegisterGoRoutine(1))

			<-ctrlC

			fmt.Println()

			Info("Terminate: CTRL-C pressed")

			Exit(130) // 128 + 2 (SIGINT)
		}()

		err = app.applicationLoop(m)
		if Error(err) {
			return err
		}

		return nil
	}()
	if err != nil {
		if IsErrExit(err) {
			errExit := err.(*ErrExit)

			Exit(errExit.ExitCode)
		} else {
			Error(err)

			Exit(1)
		}
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
			version := ""
			if app.Version != "" {
				version = fmt.Sprintf(" %s", app.Version)
			}

			banner.WriteString(fmt.Sprintf("\n"))
			banner.WriteString(fmt.Sprintf("%s%s - %s\n", strings.ToUpper(app.Title), version, app.Description))
			banner.WriteString(fmt.Sprintf("\n"))
			banner.WriteString(fmt.Sprintf("Copyright: © %d %s\n", app.Date.Year(), app.Developer))
			banner.WriteString(fmt.Sprintf("Homepage:  %s\n", app.Homepage))
			banner.WriteString(fmt.Sprintf("License:   %s\n", app.License))
			if app.Build != "" {
				banner.WriteString(fmt.Sprintf("Build:     %s\n", app.Build))
			}
			if app.Git != "" {
				banner.WriteString(fmt.Sprintf("Git:       %s\n", app.Git))
			}
			if !app.Date.IsZero() {
				banner.WriteString(fmt.Sprintf("Time:      %s\n", app.Date.Format(time.RFC822)))
			}
			banner.WriteString(fmt.Sprintf("PID:       %d\n", os.Getpid()))

			banner.WriteString(fmt.Sprintf("\n"))

			fmt.Printf("%s", banner.String())
		}
	})
}

func (app *application) applicationRun() error {
	if IsRunningAsService() {
		Info("Service()")
	} else {
		DebugFunc()
	}

	var tickerCh <-chan time.Time

	if *FlagAppTicker != 0 {
		ticker = time.NewTicker(MillisecondToDuration(*FlagAppTicker))
		defer func() {
			ticker.Stop()
		}()

		tickerCh = ticker.C
	}

	errCh := make(chan error)

	if app.RunFunc != nil && *FlagAppTicker == 0 {
		go func() {
			defer UnregisterGoRoutine(RegisterGoRoutine(1))

			err := app.RunFunc()

			Error(err)

			errCh <- err
		}()
	}

	restartCh = make(chan struct{}, 1)
	exitCh = make(chan error, 1)

	defer func() {
		close(restartCh)
		restartCh = nil

		close(exitCh)
		exitCh = nil
	}()

	restart = false

	for {
		select {
		//case <-time.After(time.Second * 3):
		//	Info("CTRL-C simulated shutdown")
		//	return nil
		//case <-time.After(time.Second):
		//	Info("Restart on time %d", runtime.NumGoroutine())
		//	fmt.Printf("Restart on time %d\n", runtime.NumGoroutine())
		//	restart = true
		//	return nil
		case err := <-errCh:
			return err
		case <-appLifecycle.Channel():
			Info("Stop on request")
			return nil
		case <-restartCh:
			Info("Restart on request")
			restart = true
			return nil
		case err := <-exitCh:
			Info("Shutdown on request")
			return err
		case <-tickerCh:
			Debug("ticker!")

			err := app.RunFunc()
			if Error(err) {
				return err
			}
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

			Error(app.applicationLoop(nil))
		}()
	} else {
		return app.applicationLoop(nil)
	}

	return nil
}

func (app *application) applicationLoop(m *testing.M) error {
	DebugFunc()

	appLifecycle.Set()
	defer func() {
		appLifecycle.Unset()
	}()

	for {
		if app.StartFunc != nil {
			err := app.StartFunc()

			if Error(err) {
				return err
			}
		}

		var appErr error

		if m != nil {
			appErr = &ErrExit{ExitCode: m.Run()}
		} else {
			appErr = app.applicationRun()
		}

		if app.StopFunc != nil {
			Error(app.StopFunc())
		}

		if !restart || m != nil {
			return appErr
		}

		err := initConfiguration()
		if Error(err) {
			return err
		}
	}
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

	return nil
}

func Restart() {
	DebugFunc()

	if appLifecycle.IsSet() {
		restartCh <- struct{}{}
	}
}

func Exit(exitCode int) {
	DebugFunc()

	if !IsRunningInTest() && exitCh != nil {
		exitCh <- &ErrExit{
			ExitCode: exitCode,
		}
	} else {
		done()

		os.Exit(exitCode)
	}
}

func IsRunningAsContainer() bool {
	onceRunningAsContainer.Do(func() {
		runningAsContainer = os.Getpid() == 1 && app.StartFunc != nil
	})

	DebugFunc(runningAsContainer)

	return runningAsContainer
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

func IsRunningInTest() bool {
	runningInTest := testing.Testing()

	DebugFunc(runningInTest)

	return runningInTest
}

func IsRunningInteractive() bool {
	onceRunningInteractive.Do(func() {
		runningInteractive = service.Interactive()
	})

	DebugFunc(runningInteractive)

	return runningInteractive
}

func IsRunningInDev() bool {
	onceRunningInDev.Do(func() {
		workDir, err := os.Getwd()
		if err != nil {
			runningInDev = false
		}

		runningInDev = FileExists(filepath.Join(workDir, ".git"))
	})

	DebugFunc(runningInDev)

	return runningInDev
}

func App() *application {
	return app
}

func done() {
	DebugFunc()

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
	if app == nil {
		return ""
	}

	onceTitle.Do(func() {
		DebugFunc(app.Title)
	})

	return app.Title
}

func Version(major bool, minor bool, patch bool) string {
	s := Split(app.Version, ".")
	dos := []bool{major, minor, patch}

	sb := strings.Builder{}

	for i := 0; i < 3; i++ {
		if dos[i] {
			if sb.Len() > 0 {
				sb.WriteString(".")
			}

			if i < len(s) {
				sb.WriteString(s[i])
			} else {
				sb.WriteString("0")
			}
		}
	}

	return sb.String()
}

func TitleVersion(major bool, minor bool, patch bool) string {
	return Title() + "-" + Version(major, minor, patch)
}
