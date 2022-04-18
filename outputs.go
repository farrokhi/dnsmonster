package main

import (
	"time"

	_ "github.com/mosajjal/dnsmonster/output" //this will automatically set up all the outputs
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

// a helper function to remove one of the outputs from the globaldispatch list
func removeIndex(s []util.GenericOutput, index int) []util.GenericOutput {
	return append(s[:index], s[index+1:]...)
}

// main output dispatch function. first, it goes through all the registered outputs,
// sees if any of them are not meant to be set up as outputs, and removes them
// then, sets up skipdomains and allowdomains tickers to periodically get them updated
// main loop of the function is a blocking loop wrapped in a goroutine. Grabs each output
// generated by our processing channel, and dispatches it to globaldispatch list
func setupOutputs(resultChannel *chan util.DNSResult) {
	log.Info("Creating the dispatch Channel")
	// go through all the registered outputs, and see if they are configured to push data, otherwise, remove them from the dispatch list
	for i := 0; i < len(util.GlobalDispatchList); i++ {
		err := util.GlobalDispatchList[i].Initialize()
		if err != nil {
			// the output does not exist, time to remove the item from our globaldispatcher
			util.GlobalDispatchList = removeIndex(util.GlobalDispatchList, i)
			// since we just removed the last item, we should go back one index to keep it consistent
			i--
		}
	}

	//check to see if at least one output is specified, otherwise we should panic exit
	if len(util.GlobalDispatchList) == 0 {
		log.Fatal("No output specified. Please specify at least one output")
	}
	//todo: currently, there's no check to see if allowdomains and skipdomains are provided if the output type demands it.

	skipDomainsFileTicker := time.NewTicker(util.GeneralFlags.SkipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if util.GeneralFlags.SkipDomainsFile == "" {
		log.Infof("skipping skipDomains refresh since it's not provided")
		skipDomainsFileTicker.Stop()
	} else {
		log.Infof("skipDomains refresh interval is %s", util.GeneralFlags.SkipDomainsRefreshInterval)
	}

	allowDomainsFileTicker := time.NewTicker(util.GeneralFlags.AllowDomainsRefreshInterval)
	allowDomainsFileTickerChan := allowDomainsFileTicker.C
	if util.GeneralFlags.AllowDomainsFile == "" {
		log.Infof("skipping allowDomains refresh since it's not provided")
		allowDomainsFileTicker.Stop()
	} else {
		log.Infof("allowDomains refresh interval is %s", util.GeneralFlags.AllowDomainsRefreshInterval)
	}
	go func() {
		// blocking loop
		for {
			select {
			case data := <-*resultChannel:
				for _, o := range util.GlobalDispatchList {
					// todo: this blocks on type0 outputs. This is still blocking for some reason
					o.OutputChannel() <- data
				}

			case <-skipDomainsFileTickerChan:
				util.GeneralFlags.LoadSkipDomain()
			case <-allowDomainsFileTickerChan:
				util.GeneralFlags.LoadAllowDomain()
			}
		}
	}()
}
