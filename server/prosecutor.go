package server

import (
	"time"
	"unsafe"

	"github.com/moooofly/dms-prosecutor/pkg/parser"
	pb "github.com/moooofly/dms-prosecutor/proto"

	"google.golang.org/grpc"

	radar "github.com/moooofly/radar-go-client"
)

// Prosecutor defines the prosecutor
type Prosecutor struct {
	radarIp   string
	radarPort string

	rsIp   string
	rsPort string

	rsUnixHost string

	checkPeriod     uint
	reconnectPeriod uint

	mode string

	allAppStatus       map[string]*serviceInfo
	checklist          map[pb.EnumRole][]string // checklist on diff role
	watchAppStatusArgs map[radar.ReqArgs]bool   // just use map as set

	rsClientConn *grpc.ClientConn // connection to remote elector
	rsClient     pb.RoleServiceClient

	radarCli *radar.RadarClient

	stopCh chan struct{}

	disconnectedRadarCh chan struct{}
	connectedRadarCh    chan struct{}

	disconnectedElectorCh chan struct{}
	connectedElectorCh    chan struct{}

	lastConnectErrPtr unsafe.Pointer
}

// NewProsecutor returns a prosecutor instance
func NewProsecutor() *Prosecutor {

	p := &Prosecutor{
		radarIp:   parser.ProsecutorSetting.RadarIp,
		radarPort: parser.ProsecutorSetting.RadarPort,

		rsIp:   parser.ProsecutorSetting.ElectorRoleServiceIp,
		rsPort: parser.ProsecutorSetting.ElectorRoleServicePort,

		rsUnixHost: parser.ProsecutorSetting.ElectorRoleServiceUnixPath,

		checkPeriod:     parser.ProsecutorSetting.CheckPeriod,
		reconnectPeriod: parser.ProsecutorSetting.ReconnectPeriod,

		mode: parser.ProsecutorSetting.Mode,
	}

	p.allAppStatus = map[string]*serviceInfo{}
	p.checklist = map[pb.EnumRole][]string{}
	p.watchAppStatusArgs = map[radar.ReqArgs]bool{}

	srvUpThres := time.Duration(parser.ProsecutorSetting.ServiceUpThreshold) * time.Second
	srvDownThres := time.Duration(parser.ProsecutorSetting.ServiceDownThreshold) * time.Second

	for name, moid := range parser.LeaderChecklistSetting.List {
		p.allAppStatus[name] = &serviceInfo{
			upThreshold:   srvUpThres,
			downThreshold: srvDownThres,
		}
		p.checklist[pb.EnumRole_Leader] = append(p.checklist[pb.EnumRole_Leader], name)

		a := radar.ReqArgs{
			DomainMoid:   parser.ProsecutorSetting.DomainMoid,
			ResourceMoid: parser.ProsecutorSetting.MachineRoomMoid,
			GroupMoid:    parser.ProsecutorSetting.GroupMoid,
			ServerName:   name,
			ServerMoid:   moid,
		}
		p.watchAppStatusArgs[a] = true
	}

	for name, moid := range parser.FollowerChecklistSetting.List {
		p.allAppStatus[name] = &serviceInfo{
			upThreshold:   srvUpThres,
			downThreshold: srvDownThres,
		}
		p.checklist[pb.EnumRole_Follower] = append(p.checklist[pb.EnumRole_Follower], name)

		a := radar.ReqArgs{
			DomainMoid:   parser.ProsecutorSetting.DomainMoid,
			ResourceMoid: parser.ProsecutorSetting.MachineRoomMoid,
			GroupMoid:    parser.ProsecutorSetting.GroupMoid,
			ServerName:   name,
			ServerMoid:   moid,
		}
		p.watchAppStatusArgs[a] = true
	}

	p.disconnectedRadarCh = make(chan struct{}, 0)
	p.disconnectedElectorCh = make(chan struct{}, 1)

	p.connectedRadarCh = make(chan struct{}, 1)
	p.connectedElectorCh = make(chan struct{}, 1)

	p.stopCh = make(chan struct{})

	return p
}

// Start the prosecutor
func (p *Prosecutor) Start() error {

	go p.radarLoop()
	go p.electorLoop()

	return nil
}

// Stop the prosecutor
func (p *Prosecutor) Stop() {
	close(p.stopCh)

	p.disconnectElector()
	p.disconnectRadar()
}
