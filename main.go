package chamber_tools

import (
	"bufio"
	"fmt"
	"github.com/bcampbell/fuzzytime"
	"log"
	"os"
	"reflect"
	"strings"
	"time"
)

// Indices type to store the column indexes of columns with specific headers denoted by "header" tags
type Indices struct {
	DatetimeIdx    int   `header:"datetime"`
	SimDatetimeIdx int   `header:"datetime-sim"`
	TemperatureIdx int   `header:"temperature"`
	HumidityIdx    int   `header:"humidity"`
	Light1Idx      int   `header:"light1"`
	Light2Idx      int   `header:"light2"`
	CO2Idx         int   `header:"co2"`
	TotalSolarIdx  int   `header:"totalsolar"`
	ChannelsIdx    []int `header:"channel-%d"`
}

// IndexConfig package level struct to store indices. -1 means it doesnt exist.
var IndexConfig = &Indices{
	-1,
	-1,
	-1,
	-1,
	-1,
	-1,
	-1,
	-1,
	[]int{},
}

var (
	ctx fuzzytime.Context
	// ZoneName exported so that packages that use this package can refer to the current timezone
	ZoneName   string
	zoneOffset int
)

func init() {
	ZoneName, zoneOffset = time.Now().Zone()
	ctx = fuzzytime.Context{
		DateResolver: fuzzytime.DMYResolver,
		TZResolver:   fuzzytime.DefaultTZResolver(ZoneName),
	}
}

func parseDateTime(tString string, errLog *log.Logger) (time.Time, error) {

	datetimeValue, _, err := ctx.Extract(tString)
	if err != nil {
		errLog.Printf("couldn't extract datetime: %s", err)
	}

	datetimeValue.Time.SetHour(datetimeValue.Time.Hour())
	datetimeValue.Time.SetMinute(datetimeValue.Time.Minute())
	datetimeValue.Time.SetSecond(datetimeValue.Time.Second())
	datetimeValue.Time.SetTZOffset(zoneOffset)

	return time.Parse("2006-01-02T15:04:05Z07:00", datetimeValue.ISOFormat())
}

// Min returns a value clamped to a lower limit limit
func Min(value, limit int) int {
	if value < limit {
		return value
	}
	return limit
}

// Max returns a value clamped to an upper limit
func Max(value, limit int) int {
	if value > limit {
		return value
	}
	return limit
}

// Clamp clamps a value to between a minimum and maximum value
func Clamp(value, minimum, maximum int) int {
	return Min(Max(value, minimum), maximum)
}

func indexInSlice(a string, list []string) int {
	for i, b := range list {
		if strings.Trim(b, "\t ,\n") == a {
			return i
		}
	}
	return -1
}

func getIndices(errLog *log.Logger, headerLine []string) {
	// initialize as invalid/empty

	v := reflect.ValueOf(IndexConfig)
	t := reflect.TypeOf(IndexConfig)
	for i := 0; i < t.Elem().NumField(); i++ {
		field := v.Elem().Field(i)
		header, ok := t.Elem().Field(i).Tag.Lookup("header")
		header = strings.Trim(header, ", \n\t")
		if !ok {
			continue
		}

		if field.CanSet() {
			if field.Kind() == reflect.Int {
				if idx := indexInSlice(header, headerLine); idx >= 0 {
					field.SetInt(int64(idx))
				}
			}
			if field.Kind() == reflect.Slice {
				cIdx := 1 // start at channel 1
				for {
					cHeader := fmt.Sprintf(header, cIdx)
					if idx := indexInSlice(cHeader, headerLine); idx >= 0 {
						iVal := reflect.ValueOf(int(idx))
						field.Set(reflect.Append(field, iVal))
						cIdx++
						continue
					}
					break
				}
			}
		}
	}
}

// InitIndexConfig populates the chamber_tools.IndexConfig struct from a header line
func InitIndexConfig(errLog *log.Logger, conditionsPath string) {

	file, err := os.Open(conditionsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	line := scanner.Text()
	lineSplit := strings.Split(line, ",")
	errLog.Println(lineSplit)
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	getIndices(errLog, lineSplit)

	if IndexConfig.DatetimeIdx < 0 {
		errLog.Println("No datetime header in conditions file")
		os.Exit(1)
	}
}

// RunConditions runs conditions for a file
func RunConditions(errLog *log.Logger, runStuff func(time.Time, []string) bool, conditionsPath string, loopFirstDay bool) {

	errLog.Printf("running conditions file: %s\n", conditionsPath)
	file, err := os.Open(conditionsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	idx := 0
	var lastTime time.Time
	var lastLineSplit []string
	firstRun := true

	if loopFirstDay {

		var firstTime time.Time
		data := make([]string, 0)
		for scanner.Scan() {
			line := scanner.Text()
			lineSplit := strings.Split(line, ",")
			if idx == 0 {
				idx++
				continue
			}

			timeStr := lineSplit[IndexConfig.DatetimeIdx]
			theTime, err := parseDateTime(timeStr, errLog)
			if err != nil {
				errLog.Println(err)
				continue
			}
			if firstTime.Unix() <= 0 {
				firstTime = theTime
			}
			if theTime.After(firstTime.Add(time.Hour * 24)) {
				break
			}
			data = append(data, line)
		}

		errLog.Printf("looping over %d timepoints", len(data))
		for {
			for _, line := range data {
				lineSplit := strings.Split(line, ",")
				timeStr := lineSplit[0]
				theTime, err := parseDateTime(timeStr, errLog)
				if err != nil {
					errLog.Println(err)
					continue
				}

				now := time.Now()
				// get the 00 value for the current time
				nowDate := now.Truncate(time.Hour * 24)
				// get the 00:00 value for the first time in the dataset
				firstDate := firstTime.Truncate(time.Hour * 24)
				// get the days difference.
				daysDifference := nowDate.Sub(firstDate)

				// adjust theTime so that we can sleep until it later.
				theTime = theTime.Add(daysDifference)

				// check if theTime is Before
				if theTime.Before(time.Now()) {
					lastLineSplit = lineSplit
					lastTime = theTime
					continue
				}
				// run the last timepoint if its the first run.
				if firstRun {
					firstRun = false
					errLog.Println("running firstrun line")
					for i := 0; i < 10; i++ {
						if runStuff(lastTime, lastLineSplit) {
							break
						}
					}
				}

				// we have reached
				errLog.Printf("sleeping for %ds\n", int(time.Until(theTime).Seconds()))
				time.Sleep(time.Until(theTime))

				// RUN STUFF HERE
				for i := 0; i < 10; i++ {
					if runStuff(theTime, lineSplit) {
						break
					}
				}
				// end RUN STUFF

			}
		}

	}

	for scanner.Scan() {
		line := scanner.Text()

		lineSplit := strings.Split(line, ",")
		if idx == 0 {
			idx++
			continue
		}

		timeStr := lineSplit[IndexConfig.DatetimeIdx]
		theTime, err := parseDateTime(timeStr, errLog)
		if err != nil {
			errLog.Println(err)
			continue
		}

		// if we are before the time skip until we are after it
		// the -10s means that we shouldnt run again.
		if theTime.Before(time.Now()) {
			lastLineSplit = lineSplit
			lastTime = theTime
			continue
		}

		if firstRun {
			firstRun = false
			errLog.Println("running firstrun line")
			for i := 0; i < 10; i++ {
				if runStuff(lastTime, lastLineSplit) {
					break
				}
			}
		}

		errLog.Printf("sleeping for %ds\n", int(time.Until(theTime).Seconds()))
		time.Sleep(time.Until(theTime))

		// RUN STUFF HERE
		for i := 0; i < 10; i++ {
			if runStuff(theTime, lineSplit) {
				break
			}
		}
		// end RUN STUFF
		idx++
	}
}
