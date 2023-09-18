package storage

import "strings"

const (
	ConfigWgTypeNative  = "native"
	ConfigWgTypeAmnezia = "amnezia"
	ConfigsWg           = ConfigWgTypeNative + "," + ConfigWgTypeAmnezia

	ConfigOvcTypeAmnezia = "amnezia"
	ConfigsOvc           = ConfigOvcTypeAmnezia

	ConfigIPSecTypeManual     = "manual"
	ConfigIPSecTypePowerShell = "ps"
	ConfigIPSecTypeMobileConf = "mobileconfig"
	ConfigsIPSec              = ConfigIPSecTypeManual + "," + ConfigIPSecTypePowerShell + "," + ConfigIPSecTypeMobileConf

	ConfigOutlineTypeAccesskey = "access_key"
	ConfigsOutline             = ConfigOutlineTypeAccesskey
)

type ConfigsImplemented struct {
	Wg      map[string]bool
	Ovc     map[string]bool
	IPSec   map[string]bool
	Outline map[string]bool
}

func NewConfigsImplemented() *ConfigsImplemented {
	return &ConfigsImplemented{
		Wg:      make(map[string]bool),
		Ovc:     make(map[string]bool),
		IPSec:   make(map[string]bool),
		Outline: make(map[string]bool),
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

func (c *ConfigsImplemented) AddOutline(s string) {
	add(c.Outline, s)
}

func (c *ConfigsImplemented) NewWgConfigs(req map[string]bool) {
	if req == nil {
		c.AddWg(ConfigsWg)

		return
	}

	c.Wg = req
}

func (c *ConfigsImplemented) NewOvcConfigs(req map[string]bool) {
	if req == nil {
		c.AddOvc(ConfigsOvc)

		return
	}

	c.Ovc = req
}

func (c *ConfigsImplemented) NewIPSecConfigs(req map[string]bool) {
	if req == nil {
		c.AddIPSec(ConfigsIPSec)

		return
	}

	c.IPSec = req
}

func (c *ConfigsImplemented) NewOutlineConfigs(req map[string]bool) {
	if req == nil {
		c.AddOutline(ConfigsOutline)

		return
	}

	c.Outline = req
}
