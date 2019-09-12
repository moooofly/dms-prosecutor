package prose

import (
	"errors"
	"sync"
	"time"

	elector "DMS-Elector/client"
	"DMS-Prosecutor-go/version"
	radar "Radar-client-mini/client"

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

type serviceInfo struct {
	upThreshold, downThreshold time.Duration
	lastUpSeen, lastDownSeen   time.Time

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

// Prosecutor defines the prosecutor
type Prosecutor struct {
	radarHost   string // radar server host
	electorHost string // elector request server host
	electorPath string // elector request server unix domain socket path

	checkPeriod     uint
	reconnectPeriod uint

	mode string

	allAppStatus       map[string]*serviceInfo
	checklist          map[elector.Role][]string    // checklist on diff role
	watchAppStatusArgs map[radar.AppStatusArgs]bool // just use map as set

	eleCli   *elector.EleCli
	radarCli *radar.RadarClient

	reconnectC chan struct{}
	stopC      chan struct{}

	logger *logrus.Logger
}

// NewProsecutor returns a prosecutor instance
func NewProsecutor(radarHost, electorHost, electorPath, domainMoid, machineRoomMoid, groupMoid string, checkPeriod,
	reconnectPeriod, serviceUpThreshold, serviceDownThreshold uint, mode string, leaderChecklists,
	followerChecklists map[string]string, logger *logrus.Logger) Prosecutor {
	var p Prosecutor

	p = Prosecutor{radarHost: radarHost, electorHost: electorHost, electorPath: electorPath, checkPeriod: checkPeriod,
		reconnectPeriod: reconnectPeriod, mode: mode}

	p.allAppStatus = map[string]*serviceInfo{}
	p.checklist = map[elector.Role][]string{}
	p.watchAppStatusArgs = map[radar.AppStatusArgs]bool{}

	// NOTE: fd, 20190110
	// there will be some redundant init work, but it is not a big deal, just ignore them
	upTD, downTD := time.Duration(serviceUpThreshold)*time.Second, time.Duration(serviceDownThreshold)*time.Second
	for k, v := range leaderChecklists {
		p.allAppStatus[k] = &serviceInfo{upThreshold: upTD, downThreshold: downTD}
		p.checklist[elector.RoleLeader] = append(p.checklist[elector.RoleLeader], k)

		a := radar.AppStatusArgs{DomainMoid: domainMoid, ResourceMoid: machineRoomMoid,
			GroupMoid: groupMoid, ServerMoid: v, ServerName: k}
		p.watchAppStatusArgs[a] = true
	}
	for k, v := range followerChecklists {
		p.allAppStatus[k] = &serviceInfo{upThreshold: upTD, downThreshold: downTD}
		p.checklist[elector.RoleFollower] = append(p.checklist[elector.RoleFollower], k)

		a := radar.AppStatusArgs{DomainMoid: domainMoid, ResourceMoid: machineRoomMoid,
			GroupMoid: groupMoid, ServerMoid: v, ServerName: k}
		p.watchAppStatusArgs[a] = true
	}

	p.stopC = make(chan struct{})
	p.reconnectC = make(chan struct{})

	p.logger = logger

	return p
}

func (p *Prosecutor) connectRadarServerTillSucceed() error {
	if p.radarCli != nil {
		if p.radarCli.Connected() {
			return errors.New("radar server already connected")
		}

		// try to close old connection
		p.radarCli.Close()
		p.radarCli = nil
	}

	for {
		select {
		case <-p.stopC:
			return errors.New("stopped")

		default:
			p.logger.Infof("[prosecutor] try to connect radar server...")

			c := radar.NewRadarClient(p.radarHost, p.logger)
			err := c.Connect()
			if err != nil {
				p.logger.Errorf("[prosecutor] try to connect radar server failed: %v", err)
			} else {
				p.radarCli = c
				p.logger.Infof("[prosecutor] radar server connected!")
				return nil
			}
		}

		time.Sleep(time.Second * time.Duration(p.reconnectPeriod))
	}
}

func (p *Prosecutor) reconnectRadarServerD(f func()) {
	for {
		select {
		case <-p.stopC:
			return

		case <-p.reconnectC:
			if err := p.connectRadarServerTillSucceed(); err == nil {
				f()
			} else {
				p.logger.Errorf("[prosecutor] reconnectRadarServerD err: %v", err)
			}
		}
	}
}

func (p *Prosecutor) reconnectRadarServer() {
	select {
	case p.reconnectC <- struct{}{}:
		p.logger.Infof("[prosecutor] trigger reconnection...")

	default:
		p.logger.Infof("[prosecutor] reconnection process is ongoing...")
	}
}

func (p *Prosecutor) disconnectRadarServer() {
	if p.radarCli != nil {
		p.radarCli.Close()
	}
}

func (p *Prosecutor) connectEleServerTillSucceed() {
	for {
		select {
		case <-p.stopC:
			return

		default:
			p.logger.Infof("[prosecutor] try to connect elector server...")
			c := elector.NewClient(p.electorHost, p.electorPath, 30)
			if err := c.Connect(); err == nil {
				p.eleCli = c
				p.logger.Infof("[prosecutor] elector server connected!")
				return
			}
		}

		time.Sleep(time.Second * time.Duration(p.reconnectPeriod))
	}
}

func (p *Prosecutor) disconnectEleServer() {
	if p.eleCli != nil {
		p.eleCli.Close()
	}
}

func (p *Prosecutor) updateServiceStatus(service string, status radar.AppStatus) serviceStatus {
	s := p.allAppStatus[service]

	if status == radar.AppStatusRunning {
		s.upSeen()
		p.logger.Infof("[prosecutor] service: %s, up detected! last up seen: %s", service,
			s.lastUpSeen.Format(time.RFC3339))
	} else {
		s.downSeen()
		p.logger.Infof("[prosecutor] service: %s, down detected! last down seen: %s", service,
			s.lastDownSeen.Format(time.RFC3339))
	}

	return s.status()
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

func (p *Prosecutor) checkAllServiceStatus(role elector.Role) bool {
	var down bool

	status := map[serviceStatus][]string{
		serviceStatusUp:        {},
		serviceStatusGoingUp:   {},
		serviceStatusGoingDown: {},
		serviceStatusDown:      {},
	}

	p.logger.Debugf("[prosecutor] radar server connected? %v", p.radarCli != nil && p.radarCli.Connected())

	for _, app := range p.checklist[role] {
		si := p.allAppStatus[app]
		s := si.status()
		status[s] = append(status[s], app)

		// log
		var last string
		var secs int
		if s == serviceStatusGoingDown || s == serviceStatusDown {
			last = si.lastDownSeen.Format(time.RFC3339)
			secs = int(time.Since(si.lastDownSeen).Seconds())
		} else {
			last = si.lastUpSeen.Format(time.RFC3339)
			secs = int(time.Since(si.lastUpSeen).Seconds())
		}
		p.logger.Infof("[prosecutor] %s, %s, since %s, for %d secs", app, s.String(), last, secs)
	}

	if len(status[serviceStatusDown]) > 0 {
		down = true
		p.logger.Warnf("[prosecutor] services down list: %v", status[serviceStatusDown])
	} else {
		p.logger.Infof("[prosecutor] services all good")
	}

	return down
}

func (p *Prosecutor) watcherD() {
	var args []radar.AppStatusArgs
	beginC := make(chan struct{})

	for k := range p.watchAppStatusArgs { // reuse watch args
		args = append(args, k)
	}

	// underground reconnect handler
	go p.reconnectRadarServerD(func() {
		// each time we reconnect radar server
		// reset all the ass watchers
		select {
		case beginC <- struct{}{}:
			p.logger.Infof("[prosecutor] reconnect finished, will restart...")

		default:
			p.logger.Infof("[prosecutor] is already began...")
		}
	})

	go func() { beginC <- struct{}{} }()

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
	case <-p.stopC:
		return

	case <-beginC:
		// step 1.
		for k := range p.watchAppStatusArgs {
			var args = k // do not refactor this line!!!

			p.logger.Debugf("[prosecutor] going to set watcher at %v", args)

			err := p.radarCli.WatchAppStatus(args, func(rpl radar.AppStatusEventReply, err error) {
				if err != nil {
					// NOTE: fd, 20190111
					// if we get any error after watching
					// mark service as failed
					p.updateServiceStatus(args.ServerName, radar.AppStatusFailed)

					if err == radar.ErrRadarServerLost {
						p.logger.Debugf("[prosecutor] should reconnect, triggered by watch callback")
						p.reconnectRadarServer()
					}
					p.logger.Errorf("[prosecutor] error of watcher: %v, w/ args: %v", err, args)
					return
				}

				p.logger.Infof("[prosecutor] watcher event got: %v, w/ args: %v", rpl, args)
				p.logger.Infof("[prosecutor] will confirm current app status...")

				rplp, errp := p.radarCli.GetAppStatus([]radar.AppStatusArgs{args})
				if errp != nil {
					p.logger.Errorf("[prosecutor] GetAppStatus failed: %v", err)
					return
				}
				if len(rplp) != 1 {
					p.logger.Warnf("[prosecutor] GetAppStatus reply len() failed: %v", rplp)
					return
				}
				p.updateServiceStatus(args.ServerName, rplp[0].Status)
			})

			if err != nil {
				p.logger.Errorf("[prosecutor] watch failed: %v, will retry...", err)
				p.updateServiceStatus(args.ServerName, radar.AppStatusFailed)

				time.Sleep(time.Second)
				go func() { beginC <- struct{}{} }()
				goto loop
			}
		}

		// step 2.
		if err := p.getAllServicesStatus(args); err != nil {
			if err == radar.ErrRadarServerLost {
				p.logger.Errorf("[prosecutor] cannot connect radar server, will retry...")
				p.reconnectRadarServer()
			} else {
				p.logger.Errorf("[prosecutor] updateAllServicesStatus failed, will retry...")
			}

			time.Sleep(time.Second)
			go func() { beginC <- struct{}{} }()
		}

		goto loop
	}
}

// Start the prosecutor
func (p *Prosecutor) Start() {
	p.logger.Infof("====================== PROSECUTOR =======================")
	p.logger.Infof("[prosecutor] starting prosecutor, version %s %s", version.Version, version.Revision)
	p.logger.Infof("[prosecutor] w/ radar-cli version %s %s", radar.VERSION, radar.DATE)

	var curRole, prevRole elector.Role
	var ticker = time.NewTicker(time.Duration(p.checkPeriod) * time.Second)

	p.connectRadarServerTillSucceed()
	p.connectEleServerTillSucceed()
	go p.watcherD()

	for range ticker.C {
		select {
		case <-p.stopC:
			return

		default:
			role, err := p.eleCli.Role()
			if err != nil {
				p.logger.Warnf("[prosecutor] request elector role failed: %v", err)
				p.connectEleServerTillSucceed()
				continue
			}

			prevRole, curRole = curRole, role
			p.logger.Infof("[prosecutor] elector current: [%s], last check: [%s]", curRole, prevRole)

			if (curRole != elector.RoleLeader) && (curRole != elector.RoleFollower) {
				continue
			}

			if curRole == elector.RoleLeader && curRole != prevRole {
				// NOTE: fd, 20180717
				// every time the elector become the leader, reset the checklist's
				// to give it a chance to not to abdicate too early, cause it may need a long time
				// to start up services in the new leader side
				p.logger.Infof("[prosecutor] become a leader! will clear service counter")
				for _, v := range p.allAppStatus {
					v.reset()
				}
			}

			if p.checkAllServiceStatus(curRole) {
				switch p.mode {
				case "single-point":
					p.logger.Infof("[prosecutor] nothing to do in single point mode")

				case "master-slave":
					p.logger.Infof("[prosecutor] service failed, telling elector to abdicate")
					ok, err := p.eleCli.Abdicate()
					if err != nil {
						p.logger.Warnf("[prosecutor] abdicate failed: %v", err)
						p.connectEleServerTillSucceed()
						continue
					}
					if !ok {
						p.logger.Infof("[prosecutor] abdication has been rejected")
					}

				case "cluster":
					p.logger.Errorf("[prosecutor] cluster handler unfinished")

				default:
					p.logger.Errorf("[prosecutor] wrong mode: %s", p.mode)
				}
			}
		}
	}
}

// Stop the prosecutor
func (p *Prosecutor) Stop() {
	close(p.stopC)
	p.disconnectEleServer()
	p.disconnectRadarServer()
}
