package xray

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/conf"
	"github.com/InazumaV/V2bX/limiter"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/features/outbound"
	coreConf "github.com/xtls/xray-core/infra/conf"
)

type DNSConfig struct {
	Servers []interface{} `json:"servers"`
	Tag     string        `json:"tag"`
}

func (c *Xray) AddNode(tag string, info *panel.NodeInfo, config *conf.Options) error {
	c.nodeReportMinTrafficBytes[tag] = config.ReportMinTraffic * 1024
	err := updateDNSConfig(info)
	if err != nil {
		return fmt.Errorf("build dns error: %s", err)
	}
	inboundConfig, err := buildInbound(config, info, tag)
	if err != nil {
		return fmt.Errorf("build inbound error: %s", err)
	}
	err = c.addInbound(inboundConfig)
	if err != nil {
		return fmt.Errorf("add inbound error: %s", err)
	}
	outBoundConfig, err := buildOutbound(config, tag)
	if err != nil {
		return fmt.Errorf("build outbound error: %s", err)
	}
	err = c.addOutbound(outBoundConfig)
	if err != nil {
		return fmt.Errorf("add outbound error: %s", err)
	}

	// Process Custom Routes
	l, _ := limiter.GetLimiter(tag)
	userRoutes := make(map[string]string)
	for i := range info.Common.Routes {
		if info.Common.Routes[i].Action == "route_user" {
			// Parse outbound config from action_value
			outboundConf := &coreConf.OutboundDetourConfig{}
			err := json.Unmarshal([]byte(info.Common.Routes[i].ActionValue), outboundConf)
			if err != nil {
				return fmt.Errorf("parse custom outbound error: %s", err)
			}
			// Ensure unique tag from remarks, fallback to route id
			outTag := info.Common.Routes[i].Remarks
			if outTag == "" {
				outTag = fmt.Sprintf("route_user_%d", info.Common.Routes[i].Id)
			}
			outboundConf.Tag = outTag

			// Build and add outbound
			builtOutbound, err := outboundConf.Build()
			if err != nil {
				return fmt.Errorf("build custom outbound error: %s", err)
			}
			err = c.addOutbound(builtOutbound)
			if err != nil {
				// if already exists, it's fine for now as we don't have a good way to update
			}

			// Map UUIDs to this tag
			var uuids []string
			if s, ok := info.Common.Routes[i].Match.(string); ok {
				uuids = []string{s}
			} else if sl, ok := info.Common.Routes[i].Match.([]string); ok {
				uuids = sl
			} else if il, ok := info.Common.Routes[i].Match.([]interface{}); ok {
				for _, item := range il {
					if s, ok := item.(string); ok {
						uuids = append(uuids, s)
					}
				}
			}
			for _, uuid := range uuids {
				userRoutes[uuid] = outTag
			}
		}
	}
	if l != nil {
		l.UpdateUserRoute(userRoutes)
	}

	return nil
}

func (c *Xray) addInbound(config *core.InboundHandlerConfig) error {
	rawHandler, err := core.CreateObject(c.Server, config)
	if err != nil {
		return err
	}
	handler, ok := rawHandler.(inbound.Handler)
	if !ok {
		return fmt.Errorf("not an InboundHandler: %s", err)
	}
	if err := c.ihm.AddHandler(context.Background(), handler); err != nil {
		return err
	}
	return nil
}

func (c *Xray) addOutbound(config *core.OutboundHandlerConfig) error {
	rawHandler, err := core.CreateObject(c.Server, config)
	if err != nil {
		return err
	}
	handler, ok := rawHandler.(outbound.Handler)
	if !ok {
		return fmt.Errorf("not an InboundHandler: %s", err)
	}
	if err := c.ohm.AddHandler(context.Background(), handler); err != nil {
		return err
	}
	return nil
}

func (c *Xray) DelNode(tag string) error {
	err := c.removeInbound(tag)
	if err != nil {
		return fmt.Errorf("remove in error: %s", err)
	}
	err = c.removeOutbound(tag)
	if err != nil {
		return fmt.Errorf("remove out error: %s", err)
	}
	return nil
}

func (c *Xray) removeInbound(tag string) error {
	return c.ihm.RemoveHandler(context.Background(), tag)
}

func (c *Xray) removeOutbound(tag string) error {
	err := c.ohm.RemoveHandler(context.Background(), tag)
	return err
}
