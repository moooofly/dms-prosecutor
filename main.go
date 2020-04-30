package main // import "github.com/moooofly/dms-prosecutor"

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	srv "github.com/moooofly/dms-prosecutor/pkg/servitization"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := srv.Init(); err != nil {
		logrus.Fatalf("err : %s", err)
	}

	done := make(chan bool)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		defer func() {
			if e := recover(); e != nil {
				fmt.Printf("crashed, err: %s\nstack:\n%s", e, string(debug.Stack()))
			}
		}()
		for sig := range signalChan {
			logrus.Infof("Recv an Unix Signal (%v), stopping...", sig)
			srv.Teardown()
			done <- true
		}
	}()

	<-done

	os.Exit(0)
}
