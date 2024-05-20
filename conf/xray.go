package conf

type XrayConfig struct {
	LogConfig          *XrayLogConfig        `json:"Log"`
	AssetPath          string                `json:"AssetPath"`
	DnsConfigPath      string                `json:"DnsConfigPath"`
	RouteConfigPath    string                `json:"RouteConfigPath"`
	ConnectionConfig   *XrayConnectionConfig `json:"XrayConnectionConfig"`
	InboundConfigPath  string                `json:"InboundConfigPath"`
	OutboundConfigPath string                `json:"OutboundConfigPath"`
}

type XrayLogConfig struct {
	Level      string `json:"Level"`
	AccessPath string `json:"AccessPath"`
	ErrorPath  string `json:"ErrorPath"`
}

type XrayConnectionConfig struct {
	Handshake    uint32 `json:"handshake"`
	ConnIdle     uint32 `json:"connIdle"`
	UplinkOnly   uint32 `json:"uplinkOnly"`
	DownlinkOnly uint32 `json:"downlinkOnly"`
	BufferSize   int32  `json:"bufferSize"`
}

func NewXrayConfig() *XrayConfig {
	return &XrayConfig{
		LogConfig: &XrayLogConfig{
			Level:      "warning",
			AccessPath: "",
			ErrorPath:  "",
		},
		AssetPath:          "/etc/rc7.d/V2bX/",
		DnsConfigPath:      "",
		InboundConfigPath:  "",
		OutboundConfigPath: "",
		RouteConfigPath:    "",
		ConnectionConfig: &XrayConnectionConfig{
			Handshake:    4,
			ConnIdle:     30,
			UplinkOnly:   2,
			DownlinkOnly: 4,
			BufferSize:   64,
		},
	}
}

type XrayOptions struct {
	EnableProxyProtocol bool                    `json:"EnableProxyProtocol"`
	EnableDNS           bool                    `json:"EnableDNS"`
	DNSType             string                  `json:"DNSType"`
	EnableUot           bool                    `json:"EnableUot"`
	EnableTFO           bool                    `json:"EnableTFO"`
	DisableIVCheck      bool                    `json:"DisableIVCheck"`
	DisableSniffing     bool                    `json:"DisableSniffing"`
	EnableFallback      bool                    `json:"EnableFallback"`
	FallBackConfigs     []FallBackConfigForXray `json:"FallBackConfigs"`
}

type FallBackConfigForXray struct {
	SNI              string `json:"SNI"`
	Alpn             string `json:"Alpn"`
	Path             string `json:"Path"`
	Dest             string `json:"Dest"`
	ProxyProtocolVer uint64 `json:"ProxyProtocolVer"`
}

func NewXrayOptions() *XrayOptions {
	return &XrayOptions{
		EnableProxyProtocol: false,
		EnableDNS:           false,
		DNSType:             "AsIs",
		EnableUot:           false,
		EnableTFO:           false,
		DisableIVCheck:      false,
		DisableSniffing:     false,
		EnableFallback:      false,
	}
}
