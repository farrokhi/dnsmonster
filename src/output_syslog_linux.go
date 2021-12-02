package main

import (
	"net/url"
	"time"

	syslog "log/syslog"

	"github.com/mosajjal/dnsmonster/types"
	log "github.com/sirupsen/logrus"
)

var syslogstats = outputStats{"Syslog", 0, 0}

func connectSyslogRetry(sysConfig syslogConfig) *syslog.Writer {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if sysConfig.syslogOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectSyslog(sysConfig)
		if err == nil {
			return conn
		} else {
			log.Info(err)
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-types.GlobalExitChannel:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectSyslog(sysConfig syslogConfig) (*syslog.Writer, error) {
	u, _ := url.Parse(sysConfig.syslogOutputEndpoint)
	log.Infof("Connecting to syslog server %v with protocol %v", u.Host, u.Scheme)
	sysLog, err := syslog.Dial(u.Scheme, u.Host, syslog.LOG_WARNING|syslog.LOG_DAEMON, sysConfig.general.serverName)
	if err != nil {
		return nil, err
	}
	return sysLog, err
}

func syslogOutput(sysConfig syslogConfig) {

	writer := connectSyslogRetry(sysConfig)

	printStatsTicker := time.Tick(sysConfig.general.printStatsDelay)

	for {
		select {
		case data := <-sysConfig.resultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if checkIfWeSkip(sysConfig.syslogOutputType, dnsQuery.Name) {
					syslogstats.Skipped++
					continue
				}
				syslogstats.SentToOutput++

				err := writer.Alert(data.String())
				// don't exit on connection failure, try to connect again if need be
				if err != nil {
					log.Info(err)
				}
				// we should skip to the next data since we've already saved all the questions. Multi-Question DNS queries are not common
				continue
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", syslogstats)
		}
	}
}
