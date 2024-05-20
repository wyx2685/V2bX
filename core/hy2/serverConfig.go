package hy2

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/apernet/hysteria/extras/outbounds"
)

type serverConfig struct {
	Listen                string                      `mapstructure:"listen"`
	Obfs                  serverConfigObfs            `mapstructure:"obfs"`
	TLS                   *serverConfigTLS            `mapstructure:"tls"`
	ACME                  *serverConfigACME           `mapstructure:"acme"`
	QUIC                  serverConfigQUIC            `mapstructure:"quic"`
	Bandwidth             serverConfigBandwidth       `mapstructure:"bandwidth"`
	IgnoreClientBandwidth bool                        `mapstructure:"ignoreClientBandwidth"`
	DisableUDP            bool                        `mapstructure:"disableUDP"`
	UDPIdleTimeout        time.Duration               `mapstructure:"udpIdleTimeout"`
	Auth                  serverConfigAuth            `mapstructure:"auth"`
	Resolver              serverConfigResolver        `mapstructure:"resolver"`
	ACL                   serverConfigACL             `mapstructure:"acl"`
	Outbounds             []serverConfigOutboundEntry `mapstructure:"outbounds"`
	TrafficStats          serverConfigTrafficStats    `mapstructure:"trafficStats"`
	Masquerade            serverConfigMasquerade      `mapstructure:"masquerade"`
}

type serverConfigObfsSalamander struct {
	Password string `mapstructure:"password"`
}

type serverConfigObfs struct {
	Type       string                     `mapstructure:"type"`
	Salamander serverConfigObfsSalamander `mapstructure:"salamander"`
}

type serverConfigTLS struct {
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type serverConfigACME struct {
	Domains        []string `mapstructure:"domains"`
	Email          string   `mapstructure:"email"`
	CA             string   `mapstructure:"ca"`
	DisableHTTP    bool     `mapstructure:"disableHTTP"`
	DisableTLSALPN bool     `mapstructure:"disableTLSALPN"`
	AltHTTPPort    int      `mapstructure:"altHTTPPort"`
	AltTLSALPNPort int      `mapstructure:"altTLSALPNPort"`
	Dir            string   `mapstructure:"dir"`
}

type serverConfigQUIC struct {
	InitStreamReceiveWindow     uint64        `mapstructure:"initStreamReceiveWindow"`
	MaxStreamReceiveWindow      uint64        `mapstructure:"maxStreamReceiveWindow"`
	InitConnectionReceiveWindow uint64        `mapstructure:"initConnReceiveWindow"`
	MaxConnectionReceiveWindow  uint64        `mapstructure:"maxConnReceiveWindow"`
	MaxIdleTimeout              time.Duration `mapstructure:"maxIdleTimeout"`
	MaxIncomingStreams          int64         `mapstructure:"maxIncomingStreams"`
	DisablePathMTUDiscovery     bool          `mapstructure:"disablePathMTUDiscovery"`
}

type serverConfigBandwidth struct {
	Up   string `mapstructure:"up"`
	Down string `mapstructure:"down"`
}

type serverConfigAuthHTTP struct {
	URL      string `mapstructure:"url"`
	Insecure bool   `mapstructure:"insecure"`
}

type serverConfigAuth struct {
	Type     string               `mapstructure:"type"`
	Password string               `mapstructure:"password"`
	UserPass map[string]string    `mapstructure:"userpass"`
	HTTP     serverConfigAuthHTTP `mapstructure:"http"`
	Command  string               `mapstructure:"command"`
}

type serverConfigResolverTCP struct {
	Addr    string        `mapstructure:"addr"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type serverConfigResolverUDP struct {
	Addr    string        `mapstructure:"addr"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type serverConfigResolverTLS struct {
	Addr     string        `mapstructure:"addr"`
	Timeout  time.Duration `mapstructure:"timeout"`
	SNI      string        `mapstructure:"sni"`
	Insecure bool          `mapstructure:"insecure"`
}

type serverConfigResolverHTTPS struct {
	Addr     string        `mapstructure:"addr"`
	Timeout  time.Duration `mapstructure:"timeout"`
	SNI      string        `mapstructure:"sni"`
	Insecure bool          `mapstructure:"insecure"`
}

type serverConfigResolver struct {
	Type  string                    `mapstructure:"type"`
	TCP   serverConfigResolverTCP   `mapstructure:"tcp"`
	UDP   serverConfigResolverUDP   `mapstructure:"udp"`
	TLS   serverConfigResolverTLS   `mapstructure:"tls"`
	HTTPS serverConfigResolverHTTPS `mapstructure:"https"`
}

type serverConfigACL struct {
	File              string        `mapstructure:"file"`
	Inline            []string      `mapstructure:"inline"`
	GeoIP             string        `mapstructure:"geoip"`
	GeoSite           string        `mapstructure:"geosite"`
	GeoUpdateInterval time.Duration `mapstructure:"geoUpdateInterval"`
}

type serverConfigOutboundDirect struct {
	Mode       string `mapstructure:"mode"`
	BindIPv4   string `mapstructure:"bindIPv4"`
	BindIPv6   string `mapstructure:"bindIPv6"`
	BindDevice string `mapstructure:"bindDevice"`
}

type serverConfigOutboundSOCKS5 struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type serverConfigOutboundHTTP struct {
	URL      string `mapstructure:"url"`
	Insecure bool   `mapstructure:"insecure"`
}

type serverConfigOutboundEntry struct {
	Name   string                     `mapstructure:"name"`
	Type   string                     `mapstructure:"type"`
	Direct serverConfigOutboundDirect `mapstructure:"direct"`
	SOCKS5 serverConfigOutboundSOCKS5 `mapstructure:"socks5"`
	HTTP   serverConfigOutboundHTTP   `mapstructure:"http"`
}

type serverConfigTrafficStats struct {
	Listen string `mapstructure:"listen"`
	Secret string `mapstructure:"secret"`
}

type serverConfigMasqueradeFile struct {
	Dir string `mapstructure:"dir"`
}

type serverConfigMasqueradeProxy struct {
	URL         string `mapstructure:"url"`
	RewriteHost bool   `mapstructure:"rewriteHost"`
}

type serverConfigMasqueradeString struct {
	Content    string            `mapstructure:"content"`
	Headers    map[string]string `mapstructure:"headers"`
	StatusCode int               `mapstructure:"statusCode"`
}

type serverConfigMasquerade struct {
	Type        string                       `mapstructure:"type"`
	File        serverConfigMasqueradeFile   `mapstructure:"file"`
	Proxy       serverConfigMasqueradeProxy  `mapstructure:"proxy"`
	String      serverConfigMasqueradeString `mapstructure:"string"`
	ListenHTTP  string                       `mapstructure:"listenHTTP"`
	ListenHTTPS string                       `mapstructure:"listenHTTPS"`
	ForceHTTPS  bool                         `mapstructure:"forceHTTPS"`
}

func serverConfigOutboundDirectToOutbound(c serverConfigOutboundDirect) (outbounds.PluggableOutbound, error) {
	var mode outbounds.DirectOutboundMode
	switch strings.ToLower(c.Mode) {
	case "", "auto":
		mode = outbounds.DirectOutboundModeAuto
	case "64":
		mode = outbounds.DirectOutboundMode64
	case "46":
		mode = outbounds.DirectOutboundMode46
	case "6":
		mode = outbounds.DirectOutboundMode6
	case "4":
		mode = outbounds.DirectOutboundMode4
	default:
		return nil, fmt.Errorf("outbounds.direct.mode unsupported mode")
	}
	bindIP := len(c.BindIPv4) > 0 || len(c.BindIPv6) > 0
	bindDevice := len(c.BindDevice) > 0
	if bindIP && bindDevice {
		return nil, fmt.Errorf("outbounds.direct cannot bind both IP and device")
	}
	if bindIP {
		ip4, ip6 := net.ParseIP(c.BindIPv4), net.ParseIP(c.BindIPv6)
		if len(c.BindIPv4) > 0 && ip4 == nil {
			return nil, fmt.Errorf("outbounds.direct.bindIPv4 invalid IPv4 address")
		}
		if len(c.BindIPv6) > 0 && ip6 == nil {
			return nil, fmt.Errorf("outbounds.direct.bindIPv6 invalid IPv6 address")
		}
		return outbounds.NewDirectOutboundBindToIPs(mode, ip4, ip6)
	}
	if bindDevice {
		return outbounds.NewDirectOutboundBindToDevice(mode, c.BindDevice)
	}
	return outbounds.NewDirectOutboundSimple(mode), nil
}

func serverConfigOutboundSOCKS5ToOutbound(c serverConfigOutboundSOCKS5) (outbounds.PluggableOutbound, error) {
	if c.Addr == "" {
		return nil, fmt.Errorf("outbounds.socks5.addr empty socks5 address")
	}
	return outbounds.NewSOCKS5Outbound(c.Addr, c.Username, c.Password), nil
}

func serverConfigOutboundHTTPToOutbound(c serverConfigOutboundHTTP) (outbounds.PluggableOutbound, error) {
	if c.URL == "" {
		return nil, fmt.Errorf("outbounds.http.url empty http address")
	}
	return outbounds.NewHTTPOutbound(c.URL, c.Insecure)
}
