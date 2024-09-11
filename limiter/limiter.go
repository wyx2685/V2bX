package limiter

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/common/format"
	"github.com/InazumaV/V2bX/conf"
	"github.com/juju/ratelimit"
	log "github.com/sirupsen/logrus"
	"github.com/xtls/xray-core/common/task"

)

var limitLock sync.RWMutex
var limiter map[string]*Limiter

func Init() {
	limiter = map[string]*Limiter{}
	c := task.Periodic{
		Interval: time.Minute * 3,
		Execute:  ClearOnlineIP,
	}
	go func() {
		log.WithField("Type", "Limiter").
			Debug("ClearOnlineIP started")
		time.Sleep(time.Minute * 3)
		_ = c.Start()
	}()
}

type Limiter struct {
	DomainRules   []*regexp.Regexp
	ProtocolRules []string
	SpeedLimit    int
	UserOnlineIP  *sync.Map      // Key: Name, value: {Key: Ip, value: Uid}
	UUIDtoUID     map[string]int // Key: UUID, value: UID
	UserLimitInfo *sync.Map      // Key: Uid value: UserLimitInfo
	ConnLimiter   *ConnLimiter   // Key: Uid value: ConnLimiter
	SpeedLimiter  *sync.Map      // key: Uid, value: *ratelimit.Bucket
	AliveList     map[int]int    // Key:Id, value:alive_ip
}

type UserLimitInfo struct {
	UID               int
	SpeedLimit        int
	DeviceLimit       int
	DynamicSpeedLimit int
	ExpireTime        int64
	OverLimit         bool
}

func AddLimiter(tag string, l *conf.LimitConfig, users []panel.UserInfo, aliveList map[int]int) *Limiter {
	info := &Limiter{
		SpeedLimit:    l.SpeedLimit,
		UserOnlineIP:  new(sync.Map),
		UserLimitInfo: new(sync.Map),
		ConnLimiter:   NewConnLimiter(l.ConnLimit, l.IPLimit, l.EnableRealtime),
		SpeedLimiter:  new(sync.Map),
		AliveList:     aliveList,
	}
	uuidmap := make(map[string]int)
	for i := range users {
		uuidmap[users[i].Uuid] = users[i].Id
		userLimit := &UserLimitInfo{}
		userLimit.UID = users[i].Id
		if users[i].SpeedLimit != 0 {
			userLimit.SpeedLimit = users[i].SpeedLimit
		}
		if users[i].DeviceLimit != 0 {
			userLimit.DeviceLimit = users[i].DeviceLimit
		}
		userLimit.OverLimit = false
		info.UserLimitInfo.Store(format.UserTag(tag, users[i].Uuid), userLimit)
	}
	info.UUIDtoUID = uuidmap
	limitLock.Lock()
	limiter[tag] = info
	limitLock.Unlock()
	return info
}

func GetLimiter(tag string) (info *Limiter, err error) {
	limitLock.RLock()
	info, ok := limiter[tag]
	limitLock.RUnlock()
	if !ok {
		return nil, errors.New("not found")
	}
	return info, nil
}

func DeleteLimiter(tag string) {
	limitLock.Lock()
	delete(limiter, tag)
	limitLock.Unlock()
}

func (l *Limiter) UpdateUser(tag string, added []panel.UserInfo, deleted []panel.UserInfo) {
	for i := range deleted {
		l.UserLimitInfo.Delete(format.UserTag(tag, deleted[i].Uuid))
		delete(l.UUIDtoUID, deleted[i].Uuid)
		l.UserOnlineIP.Delete(format.UserTag(tag, deleted[i].Uuid)) // Clear UserOnlineIP
        delete(l.AliveList, deleted[i].Id) // Removed from AliveMap
	}
	for i := range added {
		userLimit := &UserLimitInfo{
			UID: added[i].Id,
		}
		if added[i].SpeedLimit != 0 {
			userLimit.SpeedLimit = added[i].SpeedLimit
			userLimit.ExpireTime = 0
		}
		if added[i].DeviceLimit != 0 {
			userLimit.DeviceLimit = added[i].DeviceLimit
		}
		userLimit.OverLimit = false
		l.UserLimitInfo.Store(format.UserTag(tag, added[i].Uuid), userLimit)
		l.UUIDtoUID[added[i].Uuid] = added[i].Id
	}
}
