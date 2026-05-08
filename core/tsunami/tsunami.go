package tsunami

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/conf"
	vCore "github.com/InazumaV/V2bX/core"
	"github.com/tsunami-protocol/tsunami/pkg/control"
	"github.com/tsunami-protocol/tsunami/pkg/protocol"
	"github.com/tsunami-protocol/tsunami/pkg/server"
	"github.com/tsunami-protocol/tsunami/pkg/transport"
	log "github.com/sirupsen/logrus"
)

var _ vCore.Core = (*Tsunami)(nil)

// tsunamiNode holds per-tag server state.
type tsunamiNode struct {
	server  *server.Server
	tracker *control.UsageTracker
	cancel  context.CancelFunc
}

// Tsunami implements core.Core for the TSUNAMI protocol.
type Tsunami struct {
	nodes map[string]*tsunamiNode
	auth  *v2bxAuth
	mu    sync.RWMutex
}

func init() {
	vCore.RegisterCore("tsunami", New)
}

func New(c *conf.CoreConfig) (vCore.Core, error) {
	return &Tsunami{
		nodes: make(map[string]*tsunamiNode),
		auth: &v2bxAuth{
			users:     make(map[[protocol.AuthHashLen]byte]*v2bxUserEntry),
			uuidToUID: make(map[string]int),
		},
	}, nil
}

func (t *Tsunami) Protocols() []string {
	return []string{"tsunami"}
}

func (t *Tsunami) Start() error {
	return nil
}

func (t *Tsunami) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	for tag, n := range t.nodes {
		n.cancel()
		if err := n.server.Close(); err != nil {
			log.WithField("tag", tag).Errorf("tsunami: close node error: %s", err)
		}
	}
	t.nodes = make(map[string]*tsunamiNode)
	return nil
}

func (t *Tsunami) Type() string {
	return "tsunami"
}

func (t *Tsunami) AddNode(tag string, info *panel.NodeInfo, config *conf.Options) error {
	if info.Tsunami == nil {
		return nil
	}

	tracker := control.NewUsageTracker()

	// Build padding scheme text
	paddingScheme := ""
	if len(info.Tsunami.PaddingScheme) > 0 {
		paddingScheme = strings.Join(info.Tsunami.PaddingScheme, "\n")
	}

	// Determine listen address
	listenAddr := config.ListenIP + ":" + itoa(info.Common.ServerPort)

	// Build TLS config
	tlsCfg := transport.TLSConfig{
		ALPN: []string{"h2"},
	}
	if config.CertConfig != nil && config.CertConfig.CertMode != "none" && config.CertConfig.CertMode != "" {
		tlsCfg.CertFile = config.CertConfig.CertFile
		tlsCfg.KeyFile = config.CertConfig.KeyFile
	}
	if info.Common.ServerName != "" {
		tlsCfg.ServerName = info.Common.ServerName
	}

	// Build server config with limiter and usage tracker
	srvConfig := server.Config{
		Listen:         listenAddr,
		TLS:            tlsCfg,
		Authenticator:  t.auth,
		SurgeMode:      info.Tsunami.SurgeMode,
		MaxConnections: info.Tsunami.MaxConnections,
		SurgeThreshold: info.Tsunami.SurgeThreshold,
		FallbackAddr:   info.Tsunami.FallbackAddr,
		PaddingScheme:  paddingScheme,
		Traffic: control.TrafficPolicy{
			Usage:   tracker,
			Limiter: control.NewUserLimiter(), // per-user speed limiting
		},
	}

	srv, err := server.New(srvConfig)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx

	t.mu.Lock()
	t.nodes[tag] = &tsunamiNode{
		server:  srv,
		tracker: tracker,
		cancel:  cancel,
	}
	t.mu.Unlock()

	go func() {
		if err := srv.Start(); err != nil {
			if !strings.Contains(err.Error(), "use of closed") {
				log.WithField("tag", tag).Errorf("tsunami server error: %s", err)
			}
		}
	}()

	return nil
}

func (t *Tsunami) DelNode(tag string) error {
	t.mu.Lock()
	n, ok := t.nodes[tag]
	if ok {
		delete(t.nodes, tag)
	}
	t.mu.Unlock()

	if ok {
		n.cancel()
		return n.server.Close()
	}
	return nil
}

func (t *Tsunami) AddUsers(p *vCore.AddUsersParams) (added int, err error) {
	t.auth.mu.Lock()
	defer t.auth.mu.Unlock()

	for _, user := range p.Users {
		hash := sha256.Sum256([]byte(user.Uuid))
		t.auth.users[hash] = &v2bxUserEntry{
			uuid:        user.Uuid,
			uid:         user.Id,
			speedLimit:  user.SpeedLimit,
			deviceLimit: user.DeviceLimit,
		}
		t.auth.uuidToUID[user.Uuid] = user.Id
	}

	return len(p.Users), nil
}

func (t *Tsunami) DelUsers(users []panel.UserInfo, tag string, _ *panel.NodeInfo) error {
	t.auth.mu.Lock()
	defer t.auth.mu.Unlock()

	for _, user := range users {
		hash := sha256.Sum256([]byte(user.Uuid))
		delete(t.auth.users, hash)
		delete(t.auth.uuidToUID, user.Uuid)
	}
	return nil
}

func (t *Tsunami) GetUserTrafficSlice(tag string, reset bool) ([]panel.UserTraffic, error) {
	t.mu.RLock()
	n, ok := t.nodes[tag]
	t.mu.RUnlock()
	if !ok {
		return nil, nil
	}

	var deltas []control.UsageDelta
	if reset {
		deltas = n.tracker.SnapshotAndReset()
	} else {
		deltas = n.tracker.Snapshot()
	}

	if len(deltas) == 0 {
		return nil, nil
	}

	result := make([]panel.UserTraffic, 0, len(deltas))
	t.auth.mu.RLock()
	defer t.auth.mu.RUnlock()

	for _, delta := range deltas {
		uid := t.auth.uuidToUID[delta.UserID]
		if uid == 0 {
			continue
		}
		result = append(result, panel.UserTraffic{
			UID:      uid,
			Upload:   delta.UploadBytes,
			Download: delta.DownloadBytes,
		})
	}

	return result, nil
}

// v2bxUserEntry stores per-user metadata from the panel.
type v2bxUserEntry struct {
	uuid        string
	uid         int
	speedLimit  int // Mbps, 0 = unlimited
	deviceLimit int // 0 = unlimited
}

// v2bxAuth implements server.UserAuthenticator for V2bX panel users.
type v2bxAuth struct {
	users     map[[protocol.AuthHashLen]byte]*v2bxUserEntry
	uuidToUID map[string]int // uuid -> panel UID
	mu        sync.RWMutex
}

func (a *v2bxAuth) Authenticate(hash [protocol.AuthHashLen]byte) *protocol.UserInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, ok := a.users[hash]
	if !ok {
		return nil
	}

	info := &protocol.UserInfo{
		ID:   entry.uuid,
		Name: entry.uuid,
	}

	// Pass speed limit to TSUNAMI's built-in UserLimiter
	if entry.speedLimit > 0 {
		info.SpeedLimitBps = int64(entry.speedLimit) * 1000000 / 8 // Mbps -> Bytes/s
	}

	// Pass device limit for TSUNAMI session management
	if entry.deviceLimit > 0 {
		info.MaxDevices = entry.deviceLimit
	}

	return info
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
