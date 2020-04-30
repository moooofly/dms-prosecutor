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
	RadarIp   string `ini:"radar_ip"`
	RadarPort string `ini:"radar_port"`

	ElectorRoleServiceIp   string `ini:"elector_ip"`
	ElectorRoleServicePort string `ini:"elector_port"`

	ElectorRoleServiceUnixPath string `ini:"elector_unix_path"`

	LogFile  string `ini:"log_file"`
	LogLevel string `ini:"log_level"`

	DomainMoid      string `ini:"domain_moid"`
	MachineRoomMoid string `ini:"machine_room_moid"`
	GroupMoid       string `ini:"group_moid"`

	CheckPeriod     uint `ini:"check_period"`
	ReconnectPeriod uint `ini:"reconnect_period"`

	ServiceUpThreshold   uint `ini:"service_up_threshold"`
	ServiceDownThreshold uint `ini:"service_down_threshold"`

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

func Load(confFile string) {
	var err error
	cfg, err = ini.LoadSources(ini.LoadOptions{
		IgnoreInlineComment: true,
	}, confFile)
	if err != nil {
		logrus.Fatalf("Fail to parse '%s': %v", confFile, err)
	}

	mapTo("prosecutor", ProsecutorSetting)

	switch ProsecutorSetting.Mode {
	case "single-point", "master-slave", "cluster":
		//logrus.Infof("mode setting => [%s]", ProsecutorSetting.Mode)
	default:
		logrus.Fatalf("mode '%v' dose not match any of [single-point|master-slave|cluster].", ProsecutorSetting.Mode)
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
