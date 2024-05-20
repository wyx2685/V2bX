package hy2

import (
	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/conf"
	"github.com/apernet/hysteria/core/server"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Hysteria2node struct {
	Hy2server     server.Server
	Tag           string
	Logger        *zap.Logger
	EventLogger   server.EventLogger
	TrafficLogger server.TrafficLogger
}

func (h *Hysteria2) AddNode(tag string, info *panel.NodeInfo, config *conf.Options) error {
	var err error
	hyconfig := &server.Config{}
	var c serverConfig
	v := viper.New()
	if len(config.Hysteria2ConfigPath) != 0 {
		v.SetConfigFile(config.Hysteria2ConfigPath)
		if err := v.ReadInConfig(); err != nil {
			h.Logger.Fatal("failed to read server config", zap.Error(err))
		}
		if err := v.Unmarshal(&c); err != nil {
			h.Logger.Fatal("failed to parse server config", zap.Error(err))
		}
	}
	n := Hysteria2node{
		Tag:    tag,
		Logger: h.Logger,
		EventLogger: &serverLogger{
			Tag:    tag,
			logger: h.Logger,
		},
		TrafficLogger: &HookServer{
			Tag: tag,
		},
	}

	hyconfig, err = n.getHyConfig(info, config, &c)
	if err != nil {
		return err
	}
	hyconfig.Authenticator = h.Auth
	s, err := server.NewServer(hyconfig)
	if err != nil {
		return err
	}
	n.Hy2server = s
	h.Hy2nodes[tag] = n
	go func() {
		if err := s.Serve(); err != nil {
			h.Logger.Error("Server Error", zap.Error(err))
		}
	}()
	return nil
}

func (h *Hysteria2) DelNode(tag string) error {
	err := h.Hy2nodes[tag].Hy2server.Close()
	if err != nil {
		return err
	}
	delete(h.Hy2nodes, tag)
	return nil
}
