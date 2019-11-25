package main

import (
	"flag"
	"github.com/appf-anu/chamber-tools"
	"log"
	"time"
	"os"

)

var (
	errLog *log.Logger
)

var (
	loopFirstDay            				  bool
	useLight1, useLight2, usehttp             bool
	address                                   string
	conditionsPath, hostTag, groupTag, didTag string
	interval                                  time.Duration
)



// runStuff, should send values and write metrics.
// returns true if program should continue, false if program should retry
func runStuff(point *chamber_tools.TimePoint) bool {
	errLog.Printf("%+v\n", point)
	return true
}

func init() {

	var err error
	errLog = log.New(os.Stderr, "[test] ", log.Ldate|log.Ltime|log.Lshortfile)


	flag.StringVar(&conditionsPath, "conditions", "", "conditions file to")
	if tempV := os.Getenv("CONDITIONS_FILE"); tempV != "" {
		conditionsPath = tempV
	}
	flag.DurationVar(&interval, "interval", time.Minute*10, "interval to run conditions/record metrics at")
	if tempV := os.Getenv("INTERVAL"); tempV != "" {
		interval, err = time.ParseDuration(tempV)
		if err != nil {
			errLog.Println("Couldnt parse interval from environment")
			errLog.Println(err)
		}
	}
	flag.Parse()

	if conditionsPath != "" {
		chamber_tools.InitIndexConfig(errLog, conditionsPath)
		if chamber_tools.IndexConfig.TemperatureIdx == -1 || chamber_tools.IndexConfig.HumidityIdx == -1 {
			errLog.Println("No temperature or humidity headers found in conditions file" )
		}
	}
	errLog.Printf("loopFirstDay: \t%s\n", loopFirstDay)
	errLog.Printf("light1: \t%s\n", useLight1)
	errLog.Printf("light2: \t%s\n", useLight2)
	errLog.Printf("timezone: \t%s\n", chamber_tools.ZoneName)
	errLog.Printf("hostTag: \t%s\n", hostTag)
	errLog.Printf("groupTag: \t%s\n", groupTag)
	errLog.Printf("address: \t%s\n", address)
	errLog.Printf("file: \t%s\n", conditionsPath)
	errLog.Printf("interval: \t%s\n", interval)

}

func main() {

	if conditionsPath != ""{
		chamber_tools.RunConditions(errLog, runStuff, conditionsPath, loopFirstDay)
	}

}
