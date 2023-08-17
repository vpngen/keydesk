package storage

import "strings"

const (
	ConfigWgTypeNative  = "native"
	ConfigWgTypeAmnezia = "amnezia"
	ConfigsWg           = ConfigWgTypeNative + "," + ConfigWgTypeAmnezia

	ConfigOvcTypeAmnezia = "amnezia"
	ConfigsOvc           = ConfigOvcTypeAmnezia

	ConfigIPSecTypeText       = "text"
	ConfigIPSecTypePowerShell = "ps"
	ConfigIPSecTypeMobileConf = "mobileconfig"
	ConfigsIPSec              = ConfigIPSecTypeText + "," + ConfigIPSecTypePowerShell + "," + ConfigIPSecTypeMobileConf
)

type ConfigsImplemented struct {
	Wg    map[string]bool
	Ovc   map[string]bool
	IPSec map[string]bool
}

func NewConfigsImplemented() *ConfigsImplemented {
	return &ConfigsImplemented{
		Wg:    make(map[string]bool),
		Ovc:   make(map[string]bool),
		IPSec: make(map[string]bool),
	}
}

func add(m map[string]bool, s string) {
	for _, v := range strings.Split(s, ",") {
		m[strings.Trim(v, " ")] = true
	}
}

func (c *ConfigsImplemented) AddWg(s string) {
	add(c.Wg, s)
}

func (c *ConfigsImplemented) AddOvc(s string) {
	add(c.Ovc, s)
}

func (c *ConfigsImplemented) AddIPSec(s string) {
	add(c.IPSec, s)
}

func (c *ConfigsImplemented) NewWgConfigs() {
	c.AddWg(ConfigsWg)
}

func (c *ConfigsImplemented) NewOvcConfigs() {
	c.AddOvc(ConfigsOvc)
}

func (c *ConfigsImplemented) NewIPSecConfigs() {
	c.AddIPSec(ConfigsIPSec)
}
