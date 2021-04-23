package prom

import (
	"github.com/nmcclain/ldap"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DescLdapConns    = prometheus.NewDesc("workspace_ldap_connections", "Number of connections to the Google Workspace LDAP bridge", nil, nil)
	DescLdapBinds    = prometheus.NewDesc("workspace_ldap_binds", "Number of binds to the Google Workspace LDAP bridge", nil, nil)
	DescLdapUnbinds  = prometheus.NewDesc("workspace_ldap_unbinds", "Number of unbinds to the Google Workspace LDAP bridge", nil, nil)
	DescLdapSearches = prometheus.NewDesc("workspace_ldap_searches", "Number of searches to the Google Workspace LDAP bridge", nil, nil)
)

type LdapCollector struct {
	server *ldap.Server
}

func NewLdapCollector(server *ldap.Server) *LdapCollector {
	server.SetStats(true)
	return &LdapCollector{server: server}
}

func (l *LdapCollector) Describe(descs chan<- *prometheus.Desc) {
	descs <- DescLdapConns
	descs <- DescLdapBinds
	descs <- DescLdapUnbinds
	descs <- DescLdapSearches
}

func (l *LdapCollector) Collect(metrics chan<- prometheus.Metric) {
	stats := l.server.GetStats()

	metrics <- prometheus.MustNewConstMetric(DescLdapConns, prometheus.CounterValue, float64(stats.Conns))
	metrics <- prometheus.MustNewConstMetric(DescLdapBinds, prometheus.CounterValue, float64(stats.Binds))
	metrics <- prometheus.MustNewConstMetric(DescLdapUnbinds, prometheus.CounterValue, float64(stats.Unbinds))
	metrics <- prometheus.MustNewConstMetric(DescLdapSearches, prometheus.CounterValue, float64(stats.Searches))
}
