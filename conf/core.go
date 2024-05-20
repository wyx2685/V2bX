package conf

import (
	"encoding/json"
)

type CoreConfig struct {
	Type            string           `json:"Type"`
	Name            string           `json:"Name"`
	XrayConfig      *XrayConfig      `json:"-"`
	SingConfig      *SingConfig      `json:"-"`
	Hysteria2Config *Hysteria2Config `json:"-"`
}

type _CoreConfig CoreConfig

func (c *CoreConfig) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, (*_CoreConfig)(c))
	if err != nil {
		return err
	}
	switch c.Type {
	case "xray":
		c.XrayConfig = NewXrayConfig()
		return json.Unmarshal(b, c.XrayConfig)
	case "sing":
		c.SingConfig = NewSingConfig()
		return json.Unmarshal(b, c.SingConfig)
	case "hysteria2":
		c.Hysteria2Config = NewHysteria2Config()
		return json.Unmarshal(b, c.Hysteria2Config)
	}
	return nil
}
