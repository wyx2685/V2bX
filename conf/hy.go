package conf

type Hysteria2Config struct {
	LogConfig Hysteria2LogConfig `json:"Log"`
}

type Hysteria2LogConfig struct {
	Level string `json:"Level"`
}

func NewHysteria2Config() *Hysteria2Config {
	return &Hysteria2Config{
		LogConfig: Hysteria2LogConfig{
			Level: "error",
		},
	}
}
