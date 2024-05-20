package xray

import (
	"bytes"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/goccy/go-json"
	log "github.com/sirupsen/logrus"
	coreConf "github.com/xtls/xray-core/infra/conf"
)

func updateDNSConfig(node *panel.NodeInfo) (err error) {
	dnsPath := os.Getenv("XRAY_DNS_PATH")
	if len(node.RawDNS.DNSJson) != 0 {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, node.RawDNS.DNSJson, "", " "); err != nil {
			return err
		}
		err = saveDnsConfig(prettyJSON.Bytes(), dnsPath)
	} else if len(node.RawDNS.DNSMap) != 0 {
		dnsConfig := DNSConfig{
			Servers: []interface{}{
				"1.1.1.1",
				"localhost"},
			Tag: "dns_inbound",
		}
		for _, value := range node.RawDNS.DNSMap {
			address := value["address"].(string)
			if strings.Contains(address, ":") && !strings.Contains(address, "/") {
				host, port, err := net.SplitHostPort(address)
				if err != nil {
					return err
				}
				var uint16Port uint16
				if port, err := strconv.ParseUint(port, 10, 16); err == nil {
					uint16Port = uint16(port)
				}
				value["address"] = host
				value["port"] = uint16Port
			}
			dnsConfig.Servers = append(dnsConfig.Servers, value)

		}
		dnsConfigJSON, err := json.MarshalIndent(dnsConfig, "", "  ")
		if err != nil {
			log.WithField("err", err).Error("Error marshaling dnsConfig to JSON")
			return err
		}
		err = saveDnsConfig(dnsConfigJSON, dnsPath)
	}
	return err
}

func saveDnsConfig(dns []byte, dnsPath string) (err error) {
	currentData, err := os.ReadFile(dnsPath)
	if err != nil {
		log.WithField("err", err).Error("Failed to read XRAY_DNS_PATH")
		return err
	}
	if !bytes.Equal(currentData, dns) {
		coreDnsConfig := &coreConf.DNSConfig{}
		if err = json.Unmarshal(dns, coreDnsConfig); err != nil {
			log.WithField("err", err).Error("Failed to unmarshal DNS config")
		}
		_, err := coreDnsConfig.Build()
		if err != nil {
			log.WithField("err", err).Error("Failed to understand DNS config, Please check: https://xtls.github.io/config/dns.html for help")
			return err
		}
		if err = os.Truncate(dnsPath, 0); err != nil {
			log.WithField("err", err).Error("Failed to clear XRAY DNS PATH file")
		}
		if err = os.WriteFile(dnsPath, dns, 0644); err != nil {
			log.WithField("err", err).Error("Failed to write DNS to XRAY DNS PATH file")
		}
	}
	return err
}
