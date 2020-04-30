package server

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

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

	if p.radarCli != nil && p.radarCli.Connected() {
		logrus.Infof("[prosecutor] radar server connected? true")
	} else {
		logrus.Infof("[prosecutor] radar server connected? false           -- trigger reconnect")
		p.setDisconnectedState(errors.New("trigger by electorLoop"))
	}

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
		logrus.Infof("[prosecutor] %-15s, %s, since %s, for %d secs", app, s.String(), last, secs)
	}

	if len(status[serviceStatusDown]) > 0 {
		logrus.Warnf("[prosecutor] service down list: %v", status[serviceStatusDown])
		ok = false
	} else {
		//logrus.Infof("[prosecutor] services all good")
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
func (p *Prosecutor) getAllServicesStatus(args []radar.ReqArgs) error {
	if p.radarCli == nil {
		return radar.ErrRadarServerLost
	}

	logrus.Infof("[prosecutor] =====")
	logrus.Infof("[prosecutor] ===== getAllServicesStatus -> GetAppStatus()")
	logrus.Infof("[prosecutor] =====")
	rsp, err := p.radarCli.GetAppStatus(args)
	if err != nil {
		logrus.Infof("[prosecutor] =====          -> GetAppStatus() failed, %v", err)
		return err
	}
	for _, r := range rsp {
		logrus.Infof("[prosecutor] =====  --> %v", r)
	}

	logrus.Infof("[prosecutor] =====")
	logrus.Infof("[prosecutor] ===== getAllServicesStatus -> updateServiceStatus()")
	logrus.Infof("[prosecutor] =====")
	for _, r := range rsp {
		p.updateServiceStatus(r.ServerName, r.Status)
	}
	logrus.Infof("[prosecutor] =====")

	return nil
}

func (p *Prosecutor) updateServiceStatus(service string, status radar.AppStatus) serviceStatus {
	s := p.allAppStatus[service]

	if status == radar.AppStatusRunning {
		s.upSeen()
		logrus.Infof("[prosecutor]    [  %-12s  UP   ] detected! last up seen: %s", service,
			s.lastUpSeen.Format(time.RFC3339))
	} else {
		s.downSeen()
		logrus.Infof("[prosecutor]    [  %-12s  DOWN ] detected! last down seen: %s", service,
			s.lastDownSeen.Format(time.RFC3339))
	}

	return s.status()
}

func (p *Prosecutor) radarLoop() {
	logrus.Debug("[prosecutor] ====> enter radarLoop")
	defer logrus.Debug("[prosecutor] ====> leave radarLoop")

	go p.backgroundConnectRadar()
	p.setDisconnectedState(errors.New("bootstrap"))

	var args []radar.ReqArgs
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
	for {
		select {
		case <-p.stopCh:
			return

		case <-p.connectedRadarCh:

			for {

				if !p.connected() {
					break
				}

				var errWatch error

				// step 1. setup watch
				for k := range p.watchAppStatusArgs {
					var args = k // do not refactor this line!!!

					logrus.Debugf("[prosecutor] ##### set watch on -- %s", args.ServerName)
					logrus.Debugf("[prosecutor] ##### %v", args)

					errWatch = p.radarCli.WatchAppStatus(args, func(rpl radar.EventRsp, err error) {
						if err != nil {
							// NOTE: any error after watching, will be marked as service failed
							p.updateServiceStatus(args.ServerName, radar.AppStatusFailed)

							// NOTE: do not reconnect here, as it will trigger multiple reconnection
							if err == radar.ErrRadarServerLost {
								logrus.Debugf("[prosecutor]  --> find radar connection lost in watch callback ")
							}
							return
						}

						//logrus.Infof("[prosecutor]")
						//logrus.Infof("[prosecutor] ------------------------- WatchAppStatus() callback -------------------------")
						//logrus.Infof("[prosecutor] watch args  => %v", args)
						//logrus.Infof("[prosecutor] recv watch event => %v ", rpl)
						//logrus.Infof("[prosecutor]")

						rsp, errp := p.radarCli.GetAppStatus([]radar.ReqArgs{args})
						if errp != nil {
							logrus.Errorf("[prosecutor] +++++ GetAppStatus() in WatchAppStatus() callback failed, %v", errp)
							return
						}
						if len(rsp) != 1 {
							logrus.Warnf("[prosecutor] +++++ GetAppStatus() in WatchAppStatus() callback failed, (len(rsp) != 1), rsp => %v", rsp)
							return
						}

						logrus.Infof("[prosecutor] +++++ GetAppStatus() in WatchAppStatus() callback, rsp => %v", rsp)
						p.updateServiceStatus(args.ServerName, rsp[0].Status)
						//logrus.Infof("[prosecutor] ------------------------- ------------------------- -------------------------")
					})

					// FIXME: 这里的逻辑是，任意一个 watch 失败都会重新 watch 一遍，是否合理？
					//        如果出现重复 watch 是否会有问题？
					if errWatch != nil {
						logrus.Errorf("[prosecutor] ##### watch '%s' failed: %v, will retry", args.ServerName, errWatch)
						p.updateServiceStatus(args.ServerName, radar.AppStatusFailed)

						break
					}
				}

				if errWatch != nil {
					time.Sleep(1 * time.Second)
					continue
				}

				// step 2. get and update services status
				logrus.Debugf("[prosecutor] ##### radarCli -> getAllServicesStatus (GetAppStatus + updateServiceStatus)")

				if err := p.getAllServicesStatus(args); err != nil {
					if err == radar.ErrRadarServerLost {
						logrus.Errorf("[prosecutor] getAllServicesStatus() failed, '%v' -- reconnect", err)
						p.setDisconnectedState(err)
					} else {
						logrus.Errorf("[prosecutor] getAllServicesStatus() failed, '%v'", err)
					}
				}

				break
			}
		}
	}
}

func (p *Prosecutor) lastConnectError() error {
	errPtr := (*error)(atomic.LoadPointer(&p.lastConnectErrPtr))
	if errPtr == nil {
		return nil
	}
	return *errPtr
}

func (p *Prosecutor) saveLastConnectError(err error) {
	var errPtr *error
	if err != nil {
		errPtr = &err
	}
	atomic.StorePointer(&p.lastConnectErrPtr, unsafe.Pointer(errPtr))
}

func (p *Prosecutor) setDisconnectedState(err error) {
	p.saveLastConnectError(err)

	select {
	case p.disconnectedRadarCh <- struct{}{}:
	default:
	}
}

func (p *Prosecutor) setConnectedState() {
	p.saveLastConnectError(nil)

	select {
	case p.connectedRadarCh <- struct{}{}:
	default:
	}
}

func (p *Prosecutor) connected() bool {
	return p.lastConnectError() == nil
}

func (p *Prosecutor) backgroundConnectRadar() {
	for {
		select {
		case <-p.stopCh:
			return

		case <-p.disconnectedRadarCh:
		}

		for {
			logrus.Infof("[prosecutor]      --> try to connect radar '%s:%s'", p.radarIp, p.radarPort)

			// NOTE: block + timeout
			c := radar.NewRadarClient(p.radarIp, p.radarPort, logrus.StandardLogger())
			if err := c.Connect(); err != nil {
				logrus.Warnf("[prosecutor]      <-- connect radar failed, %v", err)
			} else {
				logrus.Infof("[prosecutor]      <-- connect radar success")

				// NOTE: do not change order of these two line
				p.radarCli = c
				p.setConnectedState()

				break
			}

			logrus.Infof("[prosecutor]      --- try again after %v", time.Second*time.Duration(p.reconnectPeriod))
			time.Sleep(time.Second * time.Duration(p.reconnectPeriod))
		}
	}
}

func (p *Prosecutor) disconnectRadar() {
	if p.radarCli != nil {
		p.radarCli.Disconnect()
	}
}
