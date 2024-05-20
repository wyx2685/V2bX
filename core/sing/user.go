package sing

import (
	"encoding/base64"
	"errors"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/common/counter"
	"github.com/InazumaV/V2bX/core"
	"github.com/sagernet/sing-box/inbound"
	"github.com/sagernet/sing-box/option"
)

func (b *Sing) AddUsers(p *core.AddUsersParams) (added int, err error) {
	switch p.NodeInfo.Type {
	case "vmess", "vless":
		if p.NodeInfo.Type == "vless" {
			us := make([]option.VLESSUser, len(p.Users))
			for i := range p.Users {
				us[i] = option.VLESSUser{
					Name: p.Users[i].Uuid,
					Flow: p.VAllss.Flow,
					UUID: p.Users[i].Uuid,
				}
			}
			err = b.inbounds[p.Tag].(*inbound.VLESS).AddUsers(us)
		} else {
			us := make([]option.VMessUser, len(p.Users))
			for i := range p.Users {
				us[i] = option.VMessUser{
					Name: p.Users[i].Uuid,
					UUID: p.Users[i].Uuid,
				}
			}
			err = b.inbounds[p.Tag].(*inbound.VMess).AddUsers(us)
		}
	case "shadowsocks":
		us := make([]option.ShadowsocksUser, len(p.Users))
		for i := range p.Users {
			var password = p.Users[i].Uuid
			switch p.Shadowsocks.Cipher {
			case "2022-blake3-aes-128-gcm":
				password = base64.StdEncoding.EncodeToString([]byte(password[:16]))
			case "2022-blake3-aes-256-gcm":
				password = base64.StdEncoding.EncodeToString([]byte(password[:32]))
			}
			us[i] = option.ShadowsocksUser{
				Name:     p.Users[i].Uuid,
				Password: password,
			}
		}
		err = b.inbounds[p.Tag].(*inbound.ShadowsocksMulti).AddUsers(us)
	case "trojan":
		us := make([]option.TrojanUser, len(p.Users))
		for i := range p.Users {
			us[i] = option.TrojanUser{
				Name:     p.Users[i].Uuid,
				Password: p.Users[i].Uuid,
			}
		}
		err = b.inbounds[p.Tag].(*inbound.Trojan).AddUsers(us)
	case "hysteria":
		us := make([]option.HysteriaUser, len(p.Users))
		for i := range p.Users {
			us[i] = option.HysteriaUser{
				Name:       p.Users[i].Uuid,
				AuthString: p.Users[i].Uuid,
			}
		}
		err = b.inbounds[p.Tag].(*inbound.Hysteria).AddUsers(us)
	case "hysteria2":
		us := make([]option.Hysteria2User, len(p.Users))
		id := make([]int, len(p.Users))
		for i := range p.Users {
			us[i] = option.Hysteria2User{
				Name:     p.Users[i].Uuid,
				Password: p.Users[i].Uuid,
			}
			id[i] = p.Users[i].Id
		}
		err = b.inbounds[p.Tag].(*inbound.Hysteria2).AddUsers(us, id)
	}
	if err != nil {
		return 0, err
	}
	return len(p.Users), err
}

func (b *Sing) GetUserTraffic(tag, uuid string, reset bool) (up int64, down int64) {
	if v, ok := b.hookServer.counter.Load(tag); ok {
		c := v.(*counter.TrafficCounter)
		up = c.GetUpCount(uuid)
		down = c.GetDownCount(uuid)
		if reset {
			c.Reset(uuid)
		}
		return
	}
	return 0, 0
}

type UserDeleter interface {
	DelUsers(uuid []string) error
}

func (b *Sing) DelUsers(users []panel.UserInfo, tag string) error {
	var del UserDeleter
	if i, ok := b.inbounds[tag]; ok {
		switch i.Type() {
		case "vmess":
			del = i.(*inbound.VMess)
		case "vless":
			del = i.(*inbound.VLESS)
		case "shadowsocks":
			del = i.(*inbound.ShadowsocksMulti)
		case "trojan":
			del = i.(*inbound.Trojan)
		case "hysteria":
			del = i.(*inbound.Hysteria)
		case "hysteria2":
			del = i.(*inbound.Hysteria2)
		}
	} else {
		return errors.New("the inbound not found")
	}
	uuids := make([]string, len(users))
	for i := range users {
		b.hookServer.ClearConn(tag, users[i].Uuid)
		uuids[i] = users[i].Uuid
	}
	err := del.DelUsers(uuids)
	if err != nil {
		return err
	}
	return nil
}
