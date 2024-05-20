package sing

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sagernet/sing-box/log"

	"github.com/InazumaV/V2bX/conf"
	vCore "github.com/InazumaV/V2bX/core"
	"github.com/goccy/go-json"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
)

var _ vCore.Core = (*Sing)(nil)

type DNSConfig struct {
	Servers []map[string]interface{} `json:"servers"`
	Rules   []map[string]interface{} `json:"rules"`
}

type Sing struct {
	box        *box.Box
	ctx        context.Context
	hookServer *HookServer
	router     adapter.Router
	logFactory log.Factory
	inbounds   map[string]adapter.Inbound
}

func init() {
	vCore.RegisterCore("sing", New)
}

func New(c *conf.CoreConfig) (vCore.Core, error) {
	options := option.Options{}
	if len(c.SingConfig.OriginalPath) != 0 {
		data, err := os.ReadFile(c.SingConfig.OriginalPath)
		if err != nil {
			return nil, fmt.Errorf("read original config error: %s", err)
		}
		err = json.Unmarshal(data, &options)
		if err != nil {
			return nil, fmt.Errorf("unmarshal original config error: %s", err)
		}
	}
	options.Log = &option.LogOptions{
		Disabled:  c.SingConfig.LogConfig.Disabled,
		Level:     c.SingConfig.LogConfig.Level,
		Timestamp: c.SingConfig.LogConfig.Timestamp,
		Output:    c.SingConfig.LogConfig.Output,
	}
	options.NTP = &option.NTPOptions{
		Enabled:       c.SingConfig.NtpConfig.Enable,
		WriteToSystem: true,
		Server:        c.SingConfig.NtpConfig.Server,
		ServerPort:    c.SingConfig.NtpConfig.ServerPort,
	}
	os.Setenv("SING_DNS_PATH", "")
	options.DNS = &option.DNSOptions{}
	if c.SingConfig.DnsConfigPath != "" {
		f, err := os.OpenFile(c.SingConfig.DnsConfigPath, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to open or create sing dns config file: %s", err)
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			log.Warn(fmt.Sprintf(
				"Failed to read sing dns config from file '%v': %v. Using default DNS options",
				f.Name(), err))
			options.DNS = &option.DNSOptions{}
		} else {
			if err := json.Unmarshal(data, options.DNS); err != nil {
				log.Warn(fmt.Sprintf(
					"Failed to unmarshal sing dns config from file '%v': %v. Using default DNS options",
					f.Name(), err))
				options.DNS = &option.DNSOptions{}
			}
		}
		os.Setenv("SING_DNS_PATH", c.SingConfig.DnsConfigPath)
	}
	b, err := box.New(box.Options{
		Context: context.Background(),
		Options: options,
	})
	if err != nil {
		return nil, err
	}
	hs := NewHookServer(c.SingConfig.EnableConnClear)
	b.Router().SetClashServer(hs)
	return &Sing{
		ctx:        b.Router().GetCtx(),
		box:        b,
		hookServer: hs,
		router:     b.Router(),
		logFactory: b.LogFactory(),
		inbounds:   make(map[string]adapter.Inbound),
	}, nil
}

func (b *Sing) Start() error {
	return b.box.Start()
}

func (b *Sing) Close() error {
	return b.box.Close()
}

func (b *Sing) Protocols() []string {
	return []string{
		"vmess",
		"vless",
		"shadowsocks",
		"trojan",
		"hysteria",
		"hysteria2",
	}
}

func (b *Sing) Type() string {
	return "sing"
}
