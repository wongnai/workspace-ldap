package main

import (
	"context"
	"github.com/nmcclain/ldap"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wongnai/workspace-ldap/gworkspace"
	"github.com/wongnai/workspace-ldap/prom"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	FlagBind         = kingpin.Flag("bind", "LDAP bind (port)").Short('b').Default(":389").String()
	FlagBaseDn       = kingpin.Flag("base-dn", "Base DN (eg. google.com)").Default("").String()
	FlagMetricsBind  = kingpin.Flag("metrics", "Prometheus bind").String()
	FlagLogLevel     = kingpin.Flag("level", "Log level").Default("info").String()
	FlagLogJson      = kingpin.Flag("json", "JSON Output").Default("true").Bool()
	FlagImpersonate  = kingpin.Flag("impersonate", "Google account to impersonate").String()
	FlagUpdatePeriod = kingpin.Flag("period", "Update period").Default("30m").Duration()
	FlagMaxGroup     = kingpin.Flag("max-groups", "Max groups to load").Int()
)

var ldapServer *ldap.Server
var ldapReadyChan = make(chan bool)
var ldapQuitChan = make(chan bool)
var ldapExitedChan = make(chan bool)
var metricServer *http.Server
var adminSdk *admin.Service
var searcher *gworkspace.WorkspaceSearcher

func startLDAP() {
	searcher = gworkspace.NewSearcher(adminSdk, *FlagBaseDn)
	if FlagMaxGroup != nil {
		searcher.MaxGroups = *FlagMaxGroup
	}

	log.Info().Msgf("Fetching users")
	if err := searcher.Update(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Fail to update workspace data")
	}

	log.Info().Msgf("Starting LDAP server on %s", *FlagBind)
	ldapServer = ldap.NewServer()
	ldapServer.EnforceLDAP = true
	ldapServer.SearchFunc(gworkspace.FqdnToLdap(*FlagBaseDn, "dc"), searcher)
	ldapServer.BindFunc(gworkspace.FqdnToLdap(*FlagBaseDn, "dc"), &gworkspace.WorkspaceBinder{})
	ldapServer.QuitChannel(ldapQuitChan)
	close(ldapReadyChan)
	if err := ldapServer.ListenAndServe(*FlagBind); err != nil {
		log.Error().Err(err).Msg("LDAP server error")
	}
	close(ldapExitedChan)
}

func startRefresher() {
	ticker := time.NewTicker(*FlagUpdatePeriod)
	for {
		<-ticker.C
		if err := searcher.Update(context.Background()); err != nil {
			log.Error().Err(err).Msg("Fail to update workspace data")
		}
	}
}

func startMetric() {
	log.Info().Msgf("Starting metrics server on %s", *FlagMetricsBind)
	prometheus.Register(prometheus.NewGoCollector())
	prometheus.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	prometheus.Register(prom.NewLdapCollector(ldapServer))
	metricServer = &http.Server{
		Addr:         *FlagMetricsBind,
		Handler:      promhttp.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := metricServer.ListenAndServe(); err != nil {
		log.Error().Err(err).Msg("Metric server error")
	}
}

func main() {
	kingpin.CommandLine.DefaultEnvars()
	kingpin.Parse()

	if !*FlagLogJson {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	level, err := zerolog.ParseLevel(*FlagLogLevel)
	if err != nil {
		level = zerolog.TraceLevel
	}
	zerolog.SetGlobalLevel(level)
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	if FlagImpersonate != nil && *FlagImpersonate != "" {
		credsFile, err := ioutil.ReadFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
		if err != nil {
			log.Fatal().Err(err).Msg("Fail to read GOOGLE_APPLICATION_CREDENTIALS")
		}
		creds, err := google.JWTConfigFromJSON(
			credsFile,
			admin.AdminDirectoryUserReadonlyScope,
			admin.AdminDirectoryGroupReadonlyScope,
			admin.AdminDirectoryGroupMemberReadonlyScope,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Fail to parse Google credentails")
		}
		creds.Subject = *FlagImpersonate

		adminSdk, err = admin.NewService(context.Background(), option.WithTokenSource(creds.TokenSource(context.Background())))
		if err != nil {
			log.Fatal().Err(err).Msg("Fail to start Google Admin SDK with impersonation")
		}
	} else {
		adminSdk, err = admin.NewService(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("Fail to start Google Admin SDK")
		}
	}

	go startLDAP()
	<-ldapReadyChan
	if FlagMetricsBind != nil && *FlagMetricsBind != "" {
		go startMetric()
	}
	if *FlagUpdatePeriod > 0 {
		go startRefresher()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// wait for signal
	<-c

	// tell the server to stop
	log.Info().Msg("Stopping server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	close(ldapQuitChan)
	if metricServer != nil {
		log.Info().Msg("Waiting for metric server to stop")
		metricServer.Shutdown(ctx)
	}
	log.Info().Msg("Waiting for LDAP server to stop")
	<-ldapExitedChan

	log.Info().Msg("Gracefully stopped server")
}
