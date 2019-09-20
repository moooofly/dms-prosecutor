package server

import (
	"sync"
	"time"

	pb "github.com/moooofly/dms-prosecutor/proto"
	radar "github.com/moooofly/radar-go-client"
	"github.com/sirupsen/logrus"
)

type serviceStatus int

const (
	serviceStatusDown serviceStatus = iota
	serviceStatusGoingDown
	serviceStatusGoingUp
	serviceStatusUp
)

var state2str = map[serviceStatus]string{
	serviceStatusDown:      "DOWN",
	serviceStatusGoingDown: "GOING-DOWN",
	serviceStatusGoingUp:   "GOING-UP",
	serviceStatusUp:        "UP",
}

func (s serviceStatus) String() string {
	return state2str[s]
}
func (p *Prosecutor) checkAllServiceStatus(role pb.EnumRole) (ok bool) {

	status := map[serviceStatus][]string{
		serviceStatusUp:        {},
		serviceStatusGoingUp:   {},
		serviceStatusGoingDown: {},
		serviceStatusDown:      {},
	}

	logrus.Debugf("[prosecutor] radar server connected? %v", p.radarCli != nil && p.radarCli.Connected())

	for _, app := range p.checklist[role] {
		si := p.allAppStatus[app]
		s := si.status()
		status[s] = append(status[s], app)

		// FIXME: secs 显示值在初始状态下似乎不对
		var last string
		var secs int
		if s == serviceStatusGoingDown || s == serviceStatusDown {
			last = si.lastDownSeen.Format(time.RFC3339)
			secs = int(time.Since(si.lastDownSeen).Seconds())
		} else {
			last = si.lastUpSeen.Format(time.RFC3339)
			secs = int(time.Since(si.lastUpSeen).Seconds())
		}
		logrus.Infof("[prosecutor] %s, %s, since %s, for %d secs", app, s.String(), last, secs)
	}

	if len(status[serviceStatusDown]) > 0 {
		logrus.Warnf("[prosecutor] service down list: %v", status[serviceStatusDown])
		ok = false
	} else {
		logrus.Infof("[prosecutor] services all good")
		ok = true
	}

	return
}

type serviceInfo struct {
	upThreshold   time.Duration
	downThreshold time.Duration

	lastUpSeen   time.Time
	lastDownSeen time.Time

	s serviceStatus

	sync.Mutex
}

func (si *serviceInfo) status() serviceStatus {
	si.Lock()
	defer si.Unlock()

	if si.s == serviceStatusGoingUp || si.s == serviceStatusUp {
		if time.Since(si.lastUpSeen) > si.upThreshold {
			si.s = serviceStatusUp
		} else {
			si.s = serviceStatusGoingUp
		}
	} else {
		if time.Since(si.lastDownSeen) > si.downThreshold {
			si.s = serviceStatusDown
		} else {
			si.s = serviceStatusGoingDown
		}
	}

	return si.s
}

func (si *serviceInfo) upSeen() {
	si.Lock()
	defer si.Unlock()

	if si.s == serviceStatusDown || si.s == serviceStatusGoingDown {
		si.s = serviceStatusGoingUp
	}
	si.lastUpSeen = time.Now()
}

func (si *serviceInfo) downSeen() {
	si.Lock()
	defer si.Unlock()

	if si.s == serviceStatusUp || si.s == serviceStatusGoingUp {
		si.s = serviceStatusGoingDown
	}
	si.lastDownSeen = time.Now()
}

func (si *serviceInfo) reset() {
	si.Lock()
	defer si.Unlock()

	now := time.Now()

	si.lastDownSeen = now
	si.lastUpSeen = now
}

// get all services s from radar server
func (p *Prosecutor) getAllServicesStatus(args []radar.AppStatusArgs) error {
	if p.radarCli == nil {
		return radar.ErrRadarServerLost
	}

	rpl, err := p.radarCli.GetAppStatus(args)
	if err != nil {
		return err
	}

	for _, r := range rpl {
		p.updateServiceStatus(r.ServerName, r.Status)
	}

	return nil
}

func (p *Prosecutor) updateServiceStatus(service string, status radar.AppStatus) serviceStatus {
	s := p.allAppStatus[service]

	if status == radar.AppStatusRunning {
		s.upSeen()
		logrus.Infof("[prosecutor] service [%s], up detected! last up seen: %s", service,
			s.lastUpSeen.Format(time.RFC3339))
	} else {
		s.downSeen()
		logrus.Infof("[prosecutor] service [%s], down detected! last down seen: %s", service,
			s.lastDownSeen.Format(time.RFC3339))
	}

	return s.status()
}

func (p *Prosecutor) radarLoop() {
	logrus.Debug("====> enter radarLoop")
	defer logrus.Debug("====> leave radarLoop")

	go p.backgroundConnectRadar()
	p.radarReconnectTrigger()

	var args []radar.AppStatusArgs
	for k := range p.watchAppStatusArgs { // reuse watch args
		args = append(args, k)
	}

	// steps:
	// 1. set the watchers;
	// 2. get the s, mark them as the current app s;
	// 3. for connection lost:
	//	  repeat 1-2.

	// NOTE: fd, 20190116
	// DO NOT first call GetAppStatus(), then WatchAppStatus(), cuz any even happened
	// between them will be missed, which may make us failed to abdicate in time.
	//
	// Correctness of first set-watches then get-app-status proved as:
	// Case 1: event happened between set-watches and get-app-status
	// 			no matter callback of watchers, or get-app-status both reply the newest current app status;
	// Case 2: event happened after get-app-status
	//			the first arrived reply comes from get-app-status, which will be updated by the reply
	//			of watcher's callback
	//
	// Callback: make sure no matter what kind of event happened, use GetAppStatus() to get the newest status
loop:
	select {
	case <-p.stopCh:
		return

	case <-p.connectedRadarCh:

		// step 1.
		for k := range p.watchAppStatusArgs {
			var args = k // do not refactor this line!!!

			logrus.Debugf("[prosecutor] set watcher on => %v", args)

			err := p.radarCli.WatchAppStatus(args, func(rpl radar.AppStatusEventReply, err error) {
				if err != nil {
					// NOTE: fd, 20190111
					// if we get any error after watching
					// mark service as failed
					p.updateServiceStatus(args.ServerName, radar.AppStatusFailed)

					if err == radar.ErrRadarServerLost {
						logrus.Debugf("[prosecutor] should reconnect, triggered by watch callback")
						p.radarReconnectTrigger()
					}
					logrus.Errorf("[prosecutor] error of watcher: %v, w/ args: %v", err, args)
					return
				}

				logrus.Infof("[prosecutor] watcher event got: %v, w/ args: %v", rpl, args)
				logrus.Infof("[prosecutor] will confirm current app status...")

				rplp, errp := p.radarCli.GetAppStatus([]radar.AppStatusArgs{args})
				if errp != nil {
					logrus.Errorf("[prosecutor] GetAppStatus failed: %v", err)
					return
				}
				if len(rplp) != 1 {
					logrus.Warnf("[prosecutor] GetAppStatus reply len() failed: %v", rplp)
					return
				}
				p.updateServiceStatus(args.ServerName, rplp[0].Status)
			})

			if err != nil {
				logrus.Errorf("[prosecutor] watch failed: %v, will retry...", err)
				p.updateServiceStatus(args.ServerName, radar.AppStatusFailed)

				// FIXME: 为啥要休息？
				time.Sleep(time.Second)

				// FIXME: 为啥 err 了就要回 loop 起始位置重来？
				go func() { p.connectedRadarCh <- struct{}{} }()
				goto loop
			}
		}

		// step 2.
		if err := p.getAllServicesStatus(args); err != nil {
			if err == radar.ErrRadarServerLost {
				logrus.Errorf("[prosecutor] cannot connect radar server, will retry...")
				p.radarReconnectTrigger()
			} else {
				logrus.Errorf("[prosecutor] updateAllServicesStatus failed, will retry...")
			}

			time.Sleep(time.Second)
			go func() { p.connectedRadarCh <- struct{}{} }()
		}

		goto loop
	}
}

func (p *Prosecutor) backgroundConnectRadar() {
	for {
		select {
		case <-p.stopCh:
			return

		case <-p.disconnectedRadarCh:
		}

		for {

			logrus.Infof("[prosecutor] --> try to connect radar[%s]", p.radarHost)

			c := radar.NewRadarClient(p.radarHost, logrus.StandardLogger())

			// NOTE: block + timeout
			if err := c.Connect(); err != nil {
				logrus.Warnf("[prosecutor] connect radar failed, reason: %v", err)
			} else {
				logrus.Infof("[prosecutor] connect radar success")

				// NOTE: 顺序不能变
				p.radarCli = c
				p.connectedRadarCh <- struct{}{}

				break
			}

			time.Sleep(time.Second * time.Duration(p.reconnectPeriod))
		}
	}
}

func (p *Prosecutor) radarReconnectTrigger() {
	select {
	case p.disconnectedRadarCh <- struct{}{}:
		logrus.Debugf("[prosecutor] trigger connection to [radar]")
	default:
		logrus.Debugf("[prosecutor] connection process is ongoing")
	}
}

func (p *Prosecutor) disconnectRadar() {
	if p.radarCli != nil {
		p.radarCli.Close()
	}
}
