package parser

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

var (
	cfg *ini.File

	ProsecutorSetting = &prosecutor{}

	LeaderChecklistSetting = &leaderChecklist{
		List: make(map[string]string),
	}
	FollowerChecklistSetting = &followerChecklist{
		List: make(map[string]string),
	}
)

// [prosecutor] section in .ini
type prosecutor struct {
	RadarHost string `ini:"radar-server-host"`

	ElectorRoleServiceTcpHost  string `ini:"elector-role-service-host"`
	ElectorRoleServiceUnixHost string `ini:"elector-role-service-path"`

	LogPath  string `ini:"log-path"`
	LogLevel string `ini:"log-level"`

	DomainMoid      string `ini:"domain-moid"`
	MachineRoomMoid string `ini:"machine-room-moid"`
	GroupMoid       string `ini:"group-moid"`

	CheckPeriod     uint `ini:"check-period"`
	ReconnectPeriod uint `ini:"reconnect-period"`

	ServiceUpThreshold   uint `ini:"service-up-threshold"`
	ServiceDownThreshold uint `ini:"service-down-threshold"`

	Mode string `ini:"mode"`
}

// [leader-checklist] section in .ini
type leaderChecklist struct {
	List map[string]string
}

// [follower-checklist] section in .ini
type followerChecklist struct {
	List map[string]string
}

func Load() {
	// TODO: 路径问题
	var err error
	cfg, err = ini.Load("conf/prosecutor.ini")
	if err != nil {
		logrus.Fatalf("Fail to parse 'conf/prosecutor.ini': %v", err)
	}

	mapTo("prosecutor", ProsecutorSetting)

	switch ProsecutorSetting.Mode {
	case "single-point":
		logrus.Infof("mode => [%s]", ProsecutorSetting.Mode)
		//mapTo("single-point", SinglePointSetting)
	case "master-slave":
		logrus.Infof("mode => [%s]", ProsecutorSetting.Mode)
		//mapTo("master-slave", MasterSlaveSetting)
	case "cluster":
		logrus.Infof("mode => [%s]", ProsecutorSetting.Mode)
		//mapTo("cluster", ClusterSetting)
	default:
		logrus.Fatal("not match any of [single-point|master-slave|cluster].")
	}

	for _, k := range cfg.Section("leader-checklist").Keys() {
		LeaderChecklistSetting.List[k.Name()] = k.Value()
	}
	for _, k := range cfg.Section("follower-checklist").Keys() {
		FollowerChecklistSetting.List[k.Name()] = k.Value()
	}
}

func mapTo(section string, v interface{}) {
	err := cfg.Section(section).MapTo(v)
	if err != nil {
		logrus.Fatalf("mapto err: %v", err)
	}
}
