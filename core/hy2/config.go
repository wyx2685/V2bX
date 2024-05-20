package hy2

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/conf"
	"github.com/apernet/hysteria/core/server"
	"github.com/apernet/hysteria/extras/correctnet"
	"github.com/apernet/hysteria/extras/masq"
	"github.com/apernet/hysteria/extras/obfs"
	"github.com/apernet/hysteria/extras/outbounds"
	"go.uber.org/zap"
)

type masqHandlerLogWrapper struct {
	H      http.Handler
	QUIC   bool
	Logger *zap.Logger
}

func (m *masqHandlerLogWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Logger.Debug("masquerade request",
		zap.String("addr", r.RemoteAddr),
		zap.String("method", r.Method),
		zap.String("host", r.Host),
		zap.String("url", r.URL.String()),
		zap.Bool("quic", m.QUIC))
	m.H.ServeHTTP(w, r)
}

const (
	Byte     = 1
	Kilobyte = Byte * 1000
	Megabyte = Kilobyte * 1000
	Gigabyte = Megabyte * 1000
	Terabyte = Gigabyte * 1000
)

const (
	defaultStreamReceiveWindow = 8388608                            // 8MB
	defaultConnReceiveWindow   = defaultStreamReceiveWindow * 5 / 2 // 20MB
	defaultMaxIdleTimeout      = 30 * time.Second
	defaultMaxIncomingStreams  = 4096
	defaultUDPIdleTimeout      = 60 * time.Second
)

func (n *Hysteria2node) getTLSConfig(config *conf.Options) (*server.TLSConfig, error) {
	if config.CertConfig == nil {
		return nil, fmt.Errorf("the CertConfig is not vail")
	}
	switch config.CertConfig.CertMode {
	case "none", "":
		return nil, fmt.Errorf("the CertMode cannot be none")
	default:
		var certs []tls.Certificate
		cert, err := tls.LoadX509KeyPair(config.CertConfig.CertFile, config.CertConfig.KeyFile)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
		return &server.TLSConfig{
			Certificates: certs,
			GetCertificate: func(tlsinfo *tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert, err := tls.LoadX509KeyPair(config.CertConfig.CertFile, config.CertConfig.KeyFile)
				return &cert, err
			},
		}, nil
	}
}

func (n *Hysteria2node) getQUICConfig(config *serverConfig) (*server.QUICConfig, error) {
	quic := &server.QUICConfig{}
	if config.QUIC.InitStreamReceiveWindow == 0 {
		quic.InitialStreamReceiveWindow = defaultStreamReceiveWindow
	} else if config.QUIC.InitStreamReceiveWindow < 16384 {
		return nil, fmt.Errorf("QUICConfig.InitialStreamReceiveWindowf must be at least 16384")
	} else {
		quic.InitialConnectionReceiveWindow = config.QUIC.InitConnectionReceiveWindow
	}
	if config.QUIC.MaxStreamReceiveWindow == 0 {
		quic.MaxStreamReceiveWindow = defaultStreamReceiveWindow
	} else if config.QUIC.MaxStreamReceiveWindow < 16384 {
		return nil, fmt.Errorf("QUICConfig.MaxStreamReceiveWindowf must be at least 16384")
	} else {
		quic.MaxStreamReceiveWindow = config.QUIC.MaxStreamReceiveWindow
	}
	if config.QUIC.InitConnectionReceiveWindow == 0 {
		quic.InitialConnectionReceiveWindow = defaultConnReceiveWindow
	} else if config.QUIC.InitConnectionReceiveWindow < 16384 {
		return nil, fmt.Errorf("QUICConfig.InitialConnectionReceiveWindowf must be at least 16384")
	} else {
		quic.InitialConnectionReceiveWindow = config.QUIC.InitConnectionReceiveWindow
	}
	if config.QUIC.MaxConnectionReceiveWindow == 0 {
		quic.MaxConnectionReceiveWindow = defaultConnReceiveWindow
	} else if config.QUIC.MaxConnectionReceiveWindow < 16384 {
		return nil, fmt.Errorf("QUICConfig.MaxConnectionReceiveWindowf must be at least 16384")
	} else {
		quic.MaxConnectionReceiveWindow = config.QUIC.MaxConnectionReceiveWindow
	}
	if config.QUIC.MaxIdleTimeout == 0 {
		quic.MaxIdleTimeout = defaultMaxIdleTimeout
	} else if config.QUIC.MaxIdleTimeout < 4*time.Second || config.QUIC.MaxIdleTimeout > 120*time.Second {
		return nil, fmt.Errorf("QUICConfig.MaxIdleTimeoutf must be between 4s and 120s")
	} else {
		quic.MaxIdleTimeout = config.QUIC.MaxIdleTimeout
	}
	if config.QUIC.MaxIncomingStreams == 0 {
		quic.MaxIncomingStreams = defaultMaxIncomingStreams
	} else if config.QUIC.MaxIncomingStreams < 8 {
		return nil, fmt.Errorf("QUICConfig.MaxIncomingStreamsf must be at least 8")
	} else {
		quic.MaxIncomingStreams = config.QUIC.MaxIncomingStreams
	}
	// todo fix !linux && !windows && !darwin
	quic.DisablePathMTUDiscovery = false

	return quic, nil
}
func (n *Hysteria2node) getConn(info *panel.NodeInfo, config *conf.Options) (net.PacketConn, error) {
	uAddr, err := net.ResolveUDPAddr("udp", formatAddress(config.ListenIP, info.Common.ServerPort))
	if err != nil {
		return nil, err
	}
	conn, err := correctnet.ListenUDP("udp", uAddr)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(info.Hysteria2.ObfsType) {
	case "", "plain":
		return conn, nil
	case "salamander":
		ob, err := obfs.NewSalamanderObfuscator([]byte(info.Hysteria2.ObfsPassword))
		if err != nil {
			return nil, err
		}
		return obfs.WrapPacketConn(conn, ob), nil
	default:
		return nil, fmt.Errorf("unsupported obfuscation type")
	}
}

func (n *Hysteria2node) getBandwidthConfig(info *panel.NodeInfo) *server.BandwidthConfig {
	band := &server.BandwidthConfig{}
	if info.Hysteria2.UpMbps != 0 {
		band.MaxTx = (uint64)(info.Hysteria2.UpMbps * Megabyte / 8)
	}
	if info.Hysteria2.DownMbps != 0 {
		band.MaxRx = (uint64)(info.Hysteria2.DownMbps * Megabyte / 8)

	}
	return band
}

func (n *Hysteria2node) getOutboundConfig(c *serverConfig) (server.Outbound, error) {
	// Resolver, ACL, actual outbound are all implemented through the Outbound interface.
	// Depending on the config, we build a chain like this:
	// Resolver(ACL(Outbounds...))

	// Outbounds
	var obs []outbounds.OutboundEntry
	if len(c.Outbounds) == 0 {
		// Guarantee we have at least one outbound
		obs = []outbounds.OutboundEntry{{
			Name:     "default",
			Outbound: outbounds.NewDirectOutboundSimple(outbounds.DirectOutboundModeAuto),
		}}
	} else {
		obs = make([]outbounds.OutboundEntry, len(c.Outbounds))
		for i, entry := range c.Outbounds {
			if entry.Name == "" {
				return nil, fmt.Errorf("empty outbound name")
			}
			var ob outbounds.PluggableOutbound
			var err error
			switch strings.ToLower(entry.Type) {
			case "direct":
				ob, err = serverConfigOutboundDirectToOutbound(entry.Direct)
			case "socks5":
				ob, err = serverConfigOutboundSOCKS5ToOutbound(entry.SOCKS5)
			case "http":
				ob, err = serverConfigOutboundHTTPToOutbound(entry.HTTP)
			default:
				err = fmt.Errorf("outbounds.type unsupported outbound type")
			}
			if err != nil {
				return nil, err
			}
			obs[i] = outbounds.OutboundEntry{Name: entry.Name, Outbound: ob}
		}
	}
	var uOb outbounds.PluggableOutbound // "unified" outbound

	// ACL
	hasACL := false
	if c.ACL.File != "" && len(c.ACL.Inline) > 0 {
		return nil, fmt.Errorf("cannot set both acl.file and acl.inline")
	}
	gLoader := &GeoLoader{
		GeoIPFilename:   c.ACL.GeoIP,
		GeoSiteFilename: c.ACL.GeoSite,
		UpdateInterval:  c.ACL.GeoUpdateInterval,
		Logger:          n.Logger,
	}

	if c.ACL.File != "" {
		hasACL = true
		acl, err := outbounds.NewACLEngineFromFile(c.ACL.File, obs, gLoader)
		if err != nil {
			return nil, err
		}
		uOb = acl
	} else if len(c.ACL.Inline) > 0 {
		n.Logger.Debug("found ACL Inline:", zap.Strings("Inline", c.ACL.Inline))
		hasACL = true
		acl, err := outbounds.NewACLEngineFromString(strings.Join(c.ACL.Inline, "\n"), obs, gLoader)
		if err != nil {
			return nil, err
		}
		uOb = acl
	} else {
		// No ACL, use the first outbound
		uOb = obs[0].Outbound
	}

	switch strings.ToLower(c.Resolver.Type) {
	case "", "system":
		if hasACL {
			// If the user uses ACL, we must put a resolver in front of it,
			// for IP rules to work on domain requests.
			uOb = outbounds.NewSystemResolver(uOb)
		}
		// Otherwise we can just rely on outbound handling on its own.
	case "tcp":
		if c.Resolver.TCP.Addr == "" {
			return nil, fmt.Errorf("empty resolver address")
		}
		uOb = outbounds.NewStandardResolverTCP(c.Resolver.TCP.Addr, c.Resolver.TCP.Timeout, uOb)
	case "udp":
		if c.Resolver.UDP.Addr == "" {
			return nil, fmt.Errorf("empty resolver address")
		}
		uOb = outbounds.NewStandardResolverUDP(c.Resolver.UDP.Addr, c.Resolver.UDP.Timeout, uOb)
	case "tls", "tcp-tls":
		if c.Resolver.TLS.Addr == "" {
			return nil, fmt.Errorf("empty resolver address")
		}
		uOb = outbounds.NewStandardResolverTLS(c.Resolver.TLS.Addr, c.Resolver.TLS.Timeout, c.Resolver.TLS.SNI, c.Resolver.TLS.Insecure, uOb)
	case "https", "http":
		if c.Resolver.HTTPS.Addr == "" {
			return nil, fmt.Errorf("empty resolver address")
		}
		uOb = outbounds.NewDoHResolver(c.Resolver.HTTPS.Addr, c.Resolver.HTTPS.Timeout, c.Resolver.HTTPS.SNI, c.Resolver.HTTPS.Insecure, uOb)
	default:
		return nil, fmt.Errorf("unsupported resolver type")
	}
	Outbound := &outbounds.PluggableOutboundAdapter{PluggableOutbound: uOb}

	return Outbound, nil
}

func (n *Hysteria2node) getMasqHandler(tlsconfig *server.TLSConfig, conn net.PacketConn, c *serverConfig) (http.Handler, error) {
	var handler http.Handler
	switch strings.ToLower(c.Masquerade.Type) {
	case "", "404":
		handler = http.NotFoundHandler()
	case "file":
		if c.Masquerade.File.Dir == "" {
			return nil, fmt.Errorf("masquerade.file.dir empty file directory")
		}
		handler = http.FileServer(http.Dir(c.Masquerade.File.Dir))
	case "proxy":
		if c.Masquerade.Proxy.URL == "" {
			return nil, fmt.Errorf("masquerade.proxy.url empty proxy url")
		}
		u, err := url.Parse(c.Masquerade.Proxy.URL)
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("masquerade.proxy.url %s", err))
		}
		handler = &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetURL(u)
				// SetURL rewrites the Host header,
				// but we don't want that if rewriteHost is false
				if !c.Masquerade.Proxy.RewriteHost {
					r.Out.Host = r.In.Host
				}
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				n.Logger.Error("HTTP reverse proxy error", zap.Error(err))
				w.WriteHeader(http.StatusBadGateway)
			},
		}
	case "string":
		if c.Masquerade.String.Content == "" {
			return nil, fmt.Errorf("masquerade.string.content empty string content")
		}
		if c.Masquerade.String.StatusCode != 0 &&
			(c.Masquerade.String.StatusCode < 200 ||
				c.Masquerade.String.StatusCode > 599 ||
				c.Masquerade.String.StatusCode == 233) {
			// 233 is reserved for Hysteria authentication
			return nil, fmt.Errorf("masquerade.string.statusCode invalid status code (must be 200-599, except 233)")
		}
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for k, v := range c.Masquerade.String.Headers {
				w.Header().Set(k, v)
			}
			if c.Masquerade.String.StatusCode != 0 {
				w.WriteHeader(c.Masquerade.String.StatusCode)
			} else {
				w.WriteHeader(http.StatusOK) // Use 200 OK by default
			}
			_, _ = w.Write([]byte(c.Masquerade.String.Content))
		})
	default:
		return nil, fmt.Errorf("masquerade.type unsupported masquerade type")
	}
	MasqHandler := &masqHandlerLogWrapper{H: handler, QUIC: true, Logger: n.Logger}

	if c.Masquerade.ListenHTTP != "" || c.Masquerade.ListenHTTPS != "" {
		if c.Masquerade.ListenHTTP != "" && c.Masquerade.ListenHTTPS == "" {
			return nil, fmt.Errorf("masquerade.listenHTTPS having only HTTP server without HTTPS is not supported")
		}
		s := masq.MasqTCPServer{
			QUICPort:  extractPortFromAddr(conn.LocalAddr().String()),
			HTTPSPort: extractPortFromAddr(c.Masquerade.ListenHTTPS),
			Handler:   &masqHandlerLogWrapper{H: handler, QUIC: false},
			TLSConfig: &tls.Config{
				Certificates:   tlsconfig.Certificates,
				GetCertificate: tlsconfig.GetCertificate,
			},
			ForceHTTPS: c.Masquerade.ForceHTTPS,
		}
		go runMasqTCPServer(&s, c.Masquerade.ListenHTTP, c.Masquerade.ListenHTTPS, n.Logger)
	}

	return MasqHandler, nil
}

func (n *Hysteria2node) getHyConfig(info *panel.NodeInfo, config *conf.Options, c *serverConfig) (*server.Config, error) {
	tls, err := n.getTLSConfig(config)
	if err != nil {
		return nil, err
	}
	quic, err := n.getQUICConfig(c)
	if err != nil {
		return nil, err
	}
	conn, err := n.getConn(info, config)
	if err != nil {
		return nil, err
	}
	Outbound, err := n.getOutboundConfig(c)
	if err != nil {
		return nil, err
	}
	Masq, err := n.getMasqHandler(tls, conn, c)
	if err != nil {
		return nil, err
	}
	return &server.Config{
		TLSConfig:             *tls,
		QUICConfig:            *quic,
		Conn:                  conn,
		Outbound:              Outbound,
		BandwidthConfig:       *n.getBandwidthConfig(info),
		IgnoreClientBandwidth: c.IgnoreClientBandwidth,
		DisableUDP:            c.DisableUDP,
		UDPIdleTimeout:        c.UDPIdleTimeout,
		EventLogger:           n.EventLogger,
		TrafficLogger:         n.TrafficLogger,
		MasqHandler:           Masq,
	}, nil
}

func runMasqTCPServer(s *masq.MasqTCPServer, httpAddr, httpsAddr string, logger *zap.Logger) {
	errChan := make(chan error, 2)
	if httpAddr != "" {
		go func() {
			logger.Info("masquerade HTTP server up and running", zap.String("listen", httpAddr))
			errChan <- s.ListenAndServeHTTP(httpAddr)
		}()
	}
	if httpsAddr != "" {
		go func() {
			logger.Info("masquerade HTTPS server up and running", zap.String("listen", httpsAddr))
			errChan <- s.ListenAndServeHTTPS(httpsAddr)
		}()
	}
	err := <-errChan
	if err != nil {
		logger.Fatal("failed to serve masquerade HTTP(S)", zap.Error(err))
	}
}

func extractPortFromAddr(addr string) int {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return port
}

func formatAddress(ip string, port int) string {
	if strings.Contains(ip, ":") {
		return fmt.Sprintf("[%s]:%d", ip, port)
	}
	return fmt.Sprintf("%s:%d", ip, port)
}
