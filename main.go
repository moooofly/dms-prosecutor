package main

import (
	"fmt"
	"os"

	"DMS-Prosecutor-go/prose"
	"DMS-Prosecutor-go/version"

	"Radar-client-mini/client"

	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
}

func configLogger(f *os.File, logLevel string) {
	logger.Out = f
	logger.Level = map[string]logrus.Level{
		"debug": logrus.DebugLevel, "info": logrus.InfoLevel, "warn": logrus.WarnLevel}[logLevel]
}

type conf struct {
	radarHost   string
	electorHost string
	electorPath string // unix domain

	domainMoid      string
	machineRoomMoid string
	groupMoid       string

	checkPeriod          uint
	reconnectPeriod      uint
	serviceUpThreshold   uint
	serviceDownThreshold uint

	logPath  string
	logLevel string

	mode string

	leaderChecklists   map[string]string
	followerChecklists map[string]string
}

func parseInit(path string) (conf, error) {
	var cfg = conf{leaderChecklists: make(map[string]string), followerChecklists: make(map[string]string)}

	f, err := ini.Load(path)
	if err != nil {
		return cfg, err
	}

	cfg.radarHost = f.Section("PROSECUTOR").Key("radar_host").String()
	cfg.electorHost = f.Section("PROSECUTOR").Key("elector_host").String()
	cfg.electorPath = f.Section("PROSECUTOR").Key("elector_path").String()
	cfg.domainMoid = f.Section("PROSECUTOR").Key("domain_moid").String()
	cfg.machineRoomMoid = f.Section("PROSECUTOR").Key("machine_room_moid").String()
	cfg.groupMoid = f.Section("PROSECUTOR").Key("group_moid").String()
	cfg.checkPeriod, _ = f.Section("PROSECUTOR").Key("check_period").Uint()
	cfg.reconnectPeriod, _ = f.Section("PROSECUTOR").Key("reconnect_period").Uint()
	cfg.serviceUpThreshold, _ = f.Section("PROSECUTOR").Key("service_up_threshold").Uint()
	cfg.serviceDownThreshold, _ = f.Section("PROSECUTOR").Key("service_down_threshold").Uint()
	cfg.logPath = f.Section("PROSECUTOR").Key("log_path").String()
	cfg.logLevel = f.Section("PROSECUTOR").Key("log_level").String()
	cfg.mode = f.Section("PROSECUTOR").Key("mode").String()

	for _, v := range f.Section("LEADER-CHECKLIST").Keys() {
		cfg.leaderChecklists[v.Name()] = v.Value()
	}

	for _, v := range f.Section("FOLLOWER-CHECKLIST").Keys() {
		cfg.followerChecklists[v.Name()] = v.Value()
	}

	return cfg, nil
}

func main() {
	if os.Args[1] == "--version" || os.Args[1] == "-v" {
		fmt.Printf("prosecutor %s\n", version.Version+" "+version.Revision)
		fmt.Printf("radar-cli-go %s\n", client.VERSION+" "+client.DATE)
		return
	}

	cfg, err := parseInit(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// config logger
	logFile, err := os.OpenFile(cfg.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer logFile.Close()
	configLogger(logFile, cfg.logLevel)

	p := prose.NewProsecutor(cfg.radarHost, cfg.electorHost, cfg.electorPath, cfg.domainMoid, cfg.machineRoomMoid,
		cfg.groupMoid, cfg.checkPeriod, cfg.reconnectPeriod, cfg.serviceUpThreshold, cfg.serviceDownThreshold,
		cfg.mode, cfg.leaderChecklists, cfg.followerChecklists, logger)

	p.Start()
}
