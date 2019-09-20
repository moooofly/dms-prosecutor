package server

import (
	"time"

	"github.com/moooofly/dms-prosecutor/pkg/parser"
	"google.golang.org/grpc"

	radar "github.com/moooofly/radar-go-client"

	pb "github.com/moooofly/dms-prosecutor/proto"
)

// Prosecutor defines the prosecutor
type Prosecutor struct {
	radarHost  string // radar server tcp host
	rsTcpHost  string // role service tcp host
	rsUnixHost string // role service unix host

	checkPeriod     uint
	reconnectPeriod uint

	mode string

	allAppStatus       map[string]*serviceInfo
	checklist          map[pb.EnumRole][]string     // checklist on diff role
	watchAppStatusArgs map[radar.AppStatusArgs]bool // just use map as set

	rsClientConn *grpc.ClientConn // connection to remote elector
	rsClient     pb.RoleServiceClient

	radarCli *radar.RadarClient

	disconnectedRadarCh   chan struct{}
	disconnectedElectorCh chan struct{}

	connectedRadarCh   chan struct{}
	connectedElectorCh chan struct{}

	stopCh chan struct{}
}

// NewProsecutor returns a prosecutor instance
func NewProsecutor() *Prosecutor {

	p := &Prosecutor{
		radarHost: parser.ProsecutorSetting.RadarHost,

		rsTcpHost:  parser.ProsecutorSetting.ElectorRoleServiceTcpHost,
		rsUnixHost: parser.ProsecutorSetting.ElectorRoleServiceUnixHost,

		checkPeriod:     parser.ProsecutorSetting.CheckPeriod,
		reconnectPeriod: parser.ProsecutorSetting.ReconnectPeriod,

		mode: parser.ProsecutorSetting.Mode,
	}

	p.allAppStatus = map[string]*serviceInfo{}
	p.checklist = map[pb.EnumRole][]string{}
	p.watchAppStatusArgs = map[radar.AppStatusArgs]bool{}

	srvUpThres := time.Duration(parser.ProsecutorSetting.ServiceUpThreshold) * time.Second
	srvDownThres := time.Duration(parser.ProsecutorSetting.ServiceDownThreshold) * time.Second

	for name, moid := range parser.LeaderChecklistSetting.List {
		p.allAppStatus[name] = &serviceInfo{
			upThreshold:   srvUpThres,
			downThreshold: srvDownThres,
		}
		p.checklist[pb.EnumRole_Leader] = append(p.checklist[pb.EnumRole_Leader], name)

		a := radar.AppStatusArgs{
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

		a := radar.AppStatusArgs{
			DomainMoid:   parser.ProsecutorSetting.DomainMoid,
			ResourceMoid: parser.ProsecutorSetting.MachineRoomMoid,
			GroupMoid:    parser.ProsecutorSetting.GroupMoid,
			ServerName:   name,
			ServerMoid:   moid,
		}
		p.watchAppStatusArgs[a] = true
	}

	p.disconnectedRadarCh = make(chan struct{}, 1)
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
