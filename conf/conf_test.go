package conf

import "testing"

func TestConf_LoadFromPath(t *testing.T) {
	c := New()
	t.Log(c.LoadFromPath("../example/config.yml.example"), c.NodesConfig[0].ControllerConfig.LimitConfig.IPLimit)
}
