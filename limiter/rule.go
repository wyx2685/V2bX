package limiter

import (
	"regexp"

	"github.com/InazumaV/V2bX/api/panel"
)

func (l *Limiter) CheckDomainRule(destination string) (reject bool) {
	// have rule
	for i := range l.DomainRules {
		if l.DomainRules[i].MatchString(destination) {
			reject = true
			break
		}
	}
	return
}

func (l *Limiter) CheckProtocolRule(protocol string) (reject bool) {
	for i := range l.ProtocolRules {
		if l.ProtocolRules[i] == protocol {
			reject = true
			break
		}
	}
	return
}

func (l *Limiter) UpdateRule(rule *panel.Rules) error {
	l.DomainRules = make([]*regexp.Regexp, len(rule.Regexp))
	for i := range rule.Regexp {
		l.DomainRules[i] = regexp.MustCompile(rule.Regexp[i])
	}
	l.ProtocolRules = rule.Protocol
	return nil
}

func (l *Limiter) GetUserRoute(uuid string) string {
	return l.UserRoutes[uuid]
}

func (l *Limiter) UpdateUserRoute(routes map[string]string) {
	l.UserRoutes = routes
}
