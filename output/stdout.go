package output

import (
	"errors"
	"fmt"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type StdoutConfig struct {
	StdoutOutputType        uint   `long:"stdoutOutputType"            env:"DNSMONSTER_STDOUTOUTPUTTYPE"            default:"0"                                                       description:"What should be written to stdout. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"        choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	StdoutOutputFormat      string `long:"stdoutOutputFormat"          env:"DNSMONSTER_STDOUTOUTPUTFORMAT"          default:"json"                                                    description:"Output format for stdout. options:json,csv. note that the csv splits the datetime format into multiple fields"                                                                                                                                                                        choice:"json" choice:"csv"`
	StdoutOutputWorkerCount uint   `long:"stdoutOutputWorkerCount"     env:"DNSMONSTER_STDOUTOUTPUTWORKERCOUNT"     default:"8"                                                       description:"Number of workers"`
	outputChannel           chan types.DNSResult
	closeChannel            chan bool
}

func (config StdoutConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("stdout_output", "Stdout Output", &config)

	config.outputChannel = make(chan types.DNSResult, util.GeneralFlags.ResultChannelSize)

	types.GlobalDispatchList = append(types.GlobalDispatchList, &config)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config StdoutConfig) Initialize() error {
	if config.StdoutOutputType > 0 && config.StdoutOutputType < 5 {
		log.Info("Creating Stdout Output Channel")
		go config.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (config StdoutConfig) Close() {
	//todo: implement this
	<-config.closeChannel
}

func (config StdoutConfig) OutputChannel() chan types.DNSResult {
	return config.outputChannel
}

func (stdConfig StdoutConfig) stdoutOutputWorker() {
	stdoutSentToOutput := metrics.GetOrRegisterCounter("stdoutSentToOutput", metrics.DefaultRegistry)
	stdoutSkipped := metrics.GetOrRegisterCounter("stdoutSkipped", metrics.DefaultRegistry)
	isOutputJson := stdConfig.StdoutOutputFormat == "json"
	for data := range stdConfig.outputChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(stdConfig.StdoutOutputType, dnsQuery.Name) {
				stdoutSkipped.Inc(1)
				continue
			}
			stdoutSentToOutput.Inc(1)
			if isOutputJson {
				fmt.Printf("%s\n", data.String())
			} else {
				fmt.Printf("%s\n", data.CsvRow())
			}
		}
	}

}

func (stdConfig StdoutConfig) Output() {
	if stdConfig.StdoutOutputFormat == "csv" {
		types.PrintCsvHeader()
	}
	for i := 0; i < int(stdConfig.StdoutOutputWorkerCount); i++ { //todo: make this configurable
		go stdConfig.stdoutOutputWorker()
	}
}

// actually run this as a goroutine
var _ = StdoutConfig{}.initializeFlags()