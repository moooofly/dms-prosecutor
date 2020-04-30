package servitization

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime/debug"
	"time"

	"github.com/moooofly/dms-prosecutor/pkg/parser"
	"github.com/moooofly/dms-prosecutor/pkg/version"
	"github.com/moooofly/dms-prosecutor/server"

	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// CLI facilities
var (
	app *kingpin.Application
	cmd *exec.Cmd
)

var p *server.Prosecutor

var dbg *bool
var prof *bool
var Output io.Writer

func Init() (err error) {

	// 定制 logrus 日志格式
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006/01/02 - 15:04:05",
		FullTimestamp:   true,
	})

	app = kingpin.New("prosecutor", "This is a component of dms called prosecutor.")
	app.Author("moooofly").Version(version.Version)

	// global settings
	dbg = app.Flag("debug", "debug log output").Default("false").Bool()
	level := app.Flag("level", "control the log level for debug output only").Default("debug").String()
	prof = app.Flag("prof", "generate all kinds of profile into files").Default("false").Bool()
	daemon := app.Flag("daemon", "run prosecutor in background").Default("false").Bool()
	forever := app.Flag("forever", "run prosecutor in forever, fail and retry").Default("false").Bool()
	logfile := app.Flag("log", "log file, e.g. '/opt/log/dms/prosecutor.log'").Default("").String()
	confFile := app.Flag("conf", "config file path, e.g. '/opt/config/dms/prosecutor.ini'").Default("conf/prosecutor.ini").String()
	nolog := app.Flag("nolog", "turn off logging").Default("false").Bool()

	_ = kingpin.MustParse(app.Parse(os.Args[1:]))

	// ini 配置解析
	parser.Load(*confFile)

	// log setting
	if *dbg {
		Output = os.Stdout
		logrus.SetOutput(Output)

		if *level != "" {
			switch *level {
			case "debug":
				logrus.SetLevel(logrus.DebugLevel)
			case "info":
				logrus.SetLevel(logrus.InfoLevel)
			case "warn":
				logrus.SetLevel(logrus.WarnLevel)
			case "error":
				logrus.SetLevel(logrus.ErrorLevel)
			default:
				logrus.Warnf("wrong level setting, defaut to [debug]")
				logrus.SetLevel(logrus.DebugLevel)
			}
		} else {
			logrus.SetLevel(logrus.DebugLevel)
		}
	} else {
		// log setting
		if *nolog {
			Output = ioutil.Discard
			logrus.SetOutput(Output)
		} else if *logfile != "" {
			f, err := os.OpenFile(*logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				logrus.Fatal(err)
			}
			Output = f
			logrus.SetOutput(Output)
		} else if parser.ProsecutorSetting.LogFile != "" {
			f, err := os.OpenFile(parser.ProsecutorSetting.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				logrus.Fatal(err)
			}
			Output = f
			logrus.SetOutput(Output)
		} else {
			Output = os.Stdout
			logrus.SetOutput(Output)
		}
		l, err := logrus.ParseLevel(parser.ProsecutorSetting.LogLevel)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Warningf("Update output log level to [%s]", parser.ProsecutorSetting.LogLevel)
		logrus.SetLevel(l)
	}

	// pprof setting
	if *prof {
		startProfiling()
	}

	// daemon setting
	if *daemon {
		args := []string{}
		for _, arg := range os.Args[1:] {
			if arg != "--daemon" {
				args = append(args, arg)
			}
		}
		cmd = exec.Command(os.Args[0], args...)
		cmd.Start()
		f := ""
		if *forever {
			f = "forever "
		}
		logrus.Printf("%s%s [PID] %d running...\n", f, os.Args[0], cmd.Process.Pid)
		os.Exit(0)
	}
	if *forever {
		args := []string{}
		for _, arg := range os.Args[1:] {
			if arg != "--forever" {
				args = append(args, arg)
			}
		}
		go func() {
			defer func() {
				if e := recover(); e != nil {
					fmt.Printf("crashed, err: %s\nstack:%s", e, string(debug.Stack()))
				}
			}()
			for {
				if cmd != nil {
					cmd.Process.Kill()
					time.Sleep(time.Second * 5)
				}
				cmd = exec.Command(os.Args[0], args...)
				cmdReaderStderr, err := cmd.StderrPipe()
				if err != nil {
					logrus.Printf("ERR: %s, restarting...\n", err)
					continue
				}
				cmdReader, err := cmd.StdoutPipe()
				if err != nil {
					logrus.Printf("ERR: %s, restarting...\n", err)
					continue
				}
				scanner := bufio.NewScanner(cmdReader)
				scannerStdErr := bufio.NewScanner(cmdReaderStderr)
				go func() {
					defer func() {
						if e := recover(); e != nil {
							fmt.Printf("crashed, err: %s\nstack:%s", e, string(debug.Stack()))
						}
					}()
					for scanner.Scan() {
						fmt.Println(scanner.Text())
					}
				}()
				go func() {
					defer func() {
						if e := recover(); e != nil {
							fmt.Printf("crashed, err: %s\nstack:%s", e, string(debug.Stack()))
						}
					}()
					for scannerStdErr.Scan() {
						fmt.Println(scannerStdErr.Text())
					}
				}()
				if err := cmd.Start(); err != nil {
					logrus.Printf("ERR: %s, restarting...\n", err)
					continue
				}
				pid := cmd.Process.Pid
				logrus.Printf("worker %s [PID] %d running...\n", os.Args[0], pid)
				if err := cmd.Wait(); err != nil {
					logrus.Printf("ERR: %s, restarting...", err)
					continue
				}
				logrus.Printf("worker %s [PID] %d unexpected exited, restarting...\n", os.Args[0], pid)
			}
		}()
		return
	}

	if *dbg {
		logrus.Infof("============================= prosecutor ==============================")
		logrus.Infof("version: %s", version.Version)
		logrus.Infof("============================= ========== ==============================")
	} else {
		logrus.Infof("============================= prosecutor ==============================")
	}

	p = server.NewProsecutor()

	go func() {
		if err := p.Start(); err != nil {
			logrus.Fatal(err)
		}
	}()

	return nil
}

func Teardown() {
	if cmd != nil {
		logrus.Infof("clean process %d", cmd.Process.Pid)
		cmd.Process.Kill()
	} else {
		p.Stop()
	}
	if *prof {
		saveProfiling()
	}
}
