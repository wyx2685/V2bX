package panel

import (
	"fmt"

	"github.com/goccy/go-json"
)

type OnlineUser struct {
	UID int
	IP  string
}

type UserInfo struct {
	Id          int    `json:"id"`
	Uuid        string `json:"uuid"`
	SpeedLimit  int    `json:"speed_limit"`
	DeviceLimit int    `json:"device_limit"`
	AliveIp     int    `json:"alive_ip"`
}

type UserListBody struct {
	//Msg  string `json:"msg"`
	Users []UserInfo `json:"users"`
}

// GetUserList will pull user form sspanel
func (c *Client) GetUserList() (UserList []UserInfo, err error) {
	const path = "/api/v1/server/UniProxy/user"
	r, err := c.client.R().
		SetHeader("If-None-Match", c.userEtag).
		ForceContentType("application/json").
		Get(path)
	if err = c.checkResponse(r, path, err); err != nil {
		return nil, err
	}

	if r != nil {
		defer func() {
			if r.RawBody() != nil {
				r.RawBody().Close()
			}
		}()
		if r.StatusCode() == 304 {
			return nil, nil
		}
	} else {
		return nil, fmt.Errorf("received nil response")
	}
	var userList *UserListBody
	if err != nil {
		return nil, fmt.Errorf("read body error: %s", err)
	}
	if err := json.Unmarshal(r.Body(), &userList); err != nil {
		return nil, fmt.Errorf("unmarshal userlist error: %s", err)
	}
	c.userEtag = r.Header().Get("ETag")

	var userinfos []UserInfo
	var deviceLimit, localDeviceLimit int = 0, 0
	for _, user := range userList.Users {
		// If there is still device available, add the user
		if user.DeviceLimit > 0 && user.AliveIp > 0 {
			lastOnline := 0
			if v, ok := c.LastReportOnline[user.Id]; ok {
				lastOnline = v
			}
			// If there are any available device.
			localDeviceLimit = user.DeviceLimit - user.AliveIp + lastOnline
			if localDeviceLimit > 0 {
				deviceLimit = localDeviceLimit
			} else if lastOnline > 0 {
				deviceLimit = lastOnline
			} else {
				continue
			}
		}
		user.DeviceLimit = deviceLimit
		userinfos = append(userinfos, user)
	}

	return userinfos, nil
}

type UserTraffic struct {
	UID      int
	Upload   int64
	Download int64
}

// ReportUserTraffic reports the user traffic
func (c *Client) ReportUserTraffic(userTraffic []UserTraffic) error {
	data := make(map[int][]int64, len(userTraffic))
	for i := range userTraffic {
		data[userTraffic[i].UID] = []int64{userTraffic[i].Upload, userTraffic[i].Download}
	}
	const path = "/api/v1/server/UniProxy/push"
	r, err := c.client.R().
		SetBody(data).
		ForceContentType("application/json").
		Post(path)
	err = c.checkResponse(r, path, err)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ReportNodeOnlineUsers(data *map[int][]string, reportOnline *map[int]int) error {
	c.LastReportOnline = *reportOnline
	const path = "/api/v1/server/UniProxy/alive"
	r, err := c.client.R().
		SetBody(data).
		ForceContentType("application/json").
		Post(path)
	err = c.checkResponse(r, path, err)

	if err != nil {
		return nil
	}

	return nil
}
