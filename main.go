package chamber_tools

import (
	"bufio"
	"fmt"
	"github.com/bcampbell/fuzzytime"
	"github.com/tealeg/xlsx"
	"log"
	"os"
	"reflect"
	"strings"
	"time"
	"path/filepath"
	"strconv"
	"regexp"
	"github.com/pkg/errors"
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

type TimePoint struct {
	Datetime    time.Time
	SimDatetime time.Time
	Temperature float64
	RelativeHumidity    float64
	Light1      int
	Light2      int
	CO2         float64
	TotalSolar  float64
	Channels    []float64
}

var (
	ctx fuzzytime.Context
	// ZoneName exported so that packages that use this package can refer to the current timezone
	ZoneName   string
	zoneOffset int
)

const (
	matchFloatExp = `[-+]?\d*\.\d+|\d+`
)

var /* const */ matchFloat = regexp.MustCompile(matchFloatExp)

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
	if filepath.Ext(conditionsPath) == ".xlsx" {

		xlFile, err := xlsx.OpenFile(conditionsPath)
		sheet, ok := xlFile.Sheet["timepoints"]
		if !ok {
			fmt.Println("no sheet named \"timepoints\" in xlsx file")
			os.Exit(3)
		}
		if err != nil {
			log.Fatal(err)
		}
		row := sheet.Row(0)

		headers := make([]string, 0)
		for _, cell := range row.Cells {
			headers = append(headers, cell.String())
		}

		getIndices(errLog, headers)
	}
	if filepath.Ext(conditionsPath) == ".csv" {
		file, err := os.Open(conditionsPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Scan()
		line := scanner.Text()
		lineSplit := strings.Split(line, ",")

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		getIndices(errLog, lineSplit)
	}
	errLog.Printf("%#v\n", IndexConfig)
	if IndexConfig.DatetimeIdx < 0 {
		errLog.Println("No datetime header in conditions file")
		os.Exit(1)
	}
}

func runFromCsv(errLog *log.Logger, runStuff func(point *TimePoint) bool, conditionsPath string, loopFirstDay bool){
	InitIndexConfig(errLog, conditionsPath)
	file, err := os.Open(conditionsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	idx := 0
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
					continue
				}
				// run the last timepoint if its the first run.
				if firstRun {
					firstRun = false
					errLog.Println("running firstrun line")
					for i := 0; i < 10; i++ {
						tp, err := NewTimePointFromStringArray(errLog, lastLineSplit)
						if err != nil{
							errLog.Println(err)
							break
						}
						if runStuff(tp) {
							break
						}
					}
				}

				// we have reached
				errLog.Printf("sleeping for %ds\n", int(time.Until(theTime).Seconds()))
				time.Sleep(time.Until(theTime))

				// RUN STUFF HERE
				for i := 0; i < 10; i++ {
					tp, err := NewTimePointFromStringArray(errLog, lineSplit)
					if err != nil{
						errLog.Println(err)
						break
					}
					if runStuff(tp) {
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
			continue
		}

		if firstRun {
			firstRun = false
			errLog.Println("running firstrun line")
			for i := 0; i < 10; i++ {
				tp, err := NewTimePointFromStringArray(errLog, lastLineSplit)
				if err != nil{
					errLog.Println(err)
					break
				}
				if runStuff(tp) {
					break
				}
			}
		}

		errLog.Printf("sleeping for %ds\n", int(time.Until(theTime).Seconds()))
		time.Sleep(time.Until(theTime))

		// RUN STUFF HERE
		for i := 0; i < 10; i++ {
			tp, err := NewTimePointFromStringArray(errLog, lineSplit)
			if err != nil{
				errLog.Println(err)
				break
			}
			if runStuff(tp) {
				break
			}
		}
		// end RUN STUFF
		idx++
	}
}

func NewTimePointFromStringArray(errLog *log.Logger, row []string) (*TimePoint, error) {
	tp := &TimePoint{}
	for i, cell := range row{

		if i == IndexConfig.DatetimeIdx {
			t, err := parseDateTime(cell, errLog)
			if err != nil{
				return nil, err
			}
			tp.Datetime = t
		}
		if i == IndexConfig.SimDatetimeIdx {
			t, err := parseDateTime(cell, errLog)
			if err != nil {
				errLog.Println("Couldn't get SimDatetime")
				continue
			}
			tp.SimDatetime = t
		}
		if i == IndexConfig.TemperatureIdx {
			found := matchFloat.FindString(cell)
			if len(found) < 0 {
				return nil, errors.New("no temp value found")
			}
			t, err := strconv.ParseFloat(found, 64)
			if err != nil {
				errLog.Println("failed parsing float")
				return nil, err
			}
			tp.Temperature = t
		}
		if i == IndexConfig.HumidityIdx {
			found := matchFloat.FindString(cell)
			if len(found) < 0 {
				return nil, errors.New("no hum value found")
			}
			t, err := strconv.ParseFloat(found, 64)
			if err != nil {
				errLog.Println("failed parsing float")
				return nil, err
			}
			tp.RelativeHumidity = t
		}
		if i == IndexConfig.CO2Idx {
			found := matchFloat.FindString(cell)
			if len(found) < 0 {
				return nil, errors.New("no Co2 value found")
			}
			t, err := strconv.ParseFloat(found, 64)
			if err != nil {
				errLog.Println("failed parsing float")
				return nil, err
			}
			tp.CO2 = t
		}
		if i == IndexConfig.TotalSolarIdx {
			found := matchFloat.FindString(cell)
			if len(found) < 0 {
				return nil, errors.New("no total solar value found")
			}
			t, err := strconv.ParseFloat(found, 64)
			if err != nil {
				errLog.Println("failed parsing float")
				return nil, err
			}
			tp.TotalSolar = t
		}
		if i == IndexConfig.Light1Idx {
			found := strings.TrimSpace(cell)
			t, err := strconv.ParseInt(found,10, 64)
			if err != nil {
				errLog.Println("failed parsing int")
				return nil, err
			}
			tp.Light1 = int(t)
		}
		if i == IndexConfig.Light2Idx {
			found := strings.TrimSpace(cell)
			t, err := strconv.ParseInt(found,10, 64)
			if err != nil {
				errLog.Println("failed parsing int")
				return nil, err
			}
			tp.Light2 = int(t)
		}
	}
	// do channels

	for _, chanIdx := range IndexConfig.ChannelsIdx {
		v := row[chanIdx]
		found := matchFloat.FindString(v)
		if len(found) < 0 {
			errLog.Printf("couldnt parse %s as float.\n", v)
			continue
		}
		chanValue, err := strconv.ParseFloat(found, 64)
		if err != nil{
			errLog.Println(err)
			tp.Channels = append(tp.Channels, -1.0)
			continue
		}
		tp.Channels = append(tp.Channels, chanValue)
	}
	return tp, nil

}

func NewTimePointFromRow(errLog *log.Logger, row *xlsx.Row) (*TimePoint, error) {
	tp := &TimePoint{}
	for i, cell := range row.Cells{

		if i == IndexConfig.DatetimeIdx {
			t, err := cell.GetTime(false)
			if err != nil{
				return nil, err
			}
			tp.Datetime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
				time.Local)
		}
		if i == IndexConfig.SimDatetimeIdx {
			t, err := cell.GetTime(false)
			if err != nil {
				errLog.Println("Couldn't get SimDatetime")
				continue
			}
			tp.SimDatetime = t
		}
		if i == IndexConfig.TemperatureIdx {
			t, err := cell.Float()
			if err != nil{
				return nil, err
			}
			tp.Temperature = t
		}
		if i == IndexConfig.HumidityIdx {
			t, err := cell.Float()
			if err != nil{
				return nil, err
			}
			tp.RelativeHumidity = t
		}
		if i == IndexConfig.CO2Idx {
			t, err := cell.Float()
			if err != nil{
				return nil, err
			}
			tp.CO2 = t
		}
		if i == IndexConfig.TotalSolarIdx {
			t, err := cell.Float()
			if err != nil{
				errLog.Println("Couldn't get TotalSolar")
				continue
			}
			tp.TotalSolar = t
		}
		if i == IndexConfig.Light1Idx {
			t, err := cell.Int()
			if err != nil{
				return nil, err
			}
			tp.Light1 = t
		}
		if i == IndexConfig.Light2Idx {
			t, err := cell.Int()
			if err != nil{
				return nil, err
			}
			tp.Light2 = t
		}
	}
	// do channels
	for _, chanIdx := range IndexConfig.ChannelsIdx {
		chanValue, err := row.Cells[chanIdx].Float()
		if err != nil{
			tp.Channels = append(tp.Channels, -1.0)
			continue
		}
		tp.Channels = append(tp.Channels, chanValue)
	}
	return tp, nil

}

func runFromXlsx(errLog *log.Logger, runStuff func(point *TimePoint) bool, conditionsPath string, loopFirstDay bool){
	InitIndexConfig(errLog, conditionsPath)
	xlFile, err := xlsx.OpenFile(conditionsPath)
	if err != nil {
		log.Fatal(err)
	}
	// excel files dont require closing?
	//defer file.Close()
	sheet, ok := xlFile.Sheet["timepoints"]
	if !ok {
		fmt.Println("no sheet named \"timepoints\" in xlsx file")
		os.Exit(3)
	}

	var lastRow *xlsx.Row
	firstRun := true

	if loopFirstDay {

		var firstTime time.Time
		data := make([]*xlsx.Row, 0)
		for i, row := range sheet.Rows {
			if i == 0 {
				continue
			}
			tempTime, err := row.Cells[IndexConfig.DatetimeIdx].GetTime(false)

			if err!= nil {
				errLog.Println(err)
				continue
			}
			theTime := time.Date(
				tempTime.Year(),
				tempTime.Month(),
				tempTime.Day(),
				tempTime.Hour(),
				tempTime.Minute(),
				tempTime.Second(),
				tempTime.Nanosecond(),
				time.Local)
			if firstTime.Unix() <= 0 {
				firstTime = theTime
			}
			if theTime.After(firstTime.Add(time.Hour * 24)) {
				break
			}
			data = append(data, row)
		}

		errLog.Printf("looping over %d timepoints", len(data))
		for {
			for _, row := range data {
				tempTime, err := row.Cells[IndexConfig.DatetimeIdx].GetTime(false)
				if err!= nil {
					errLog.Println(err)
					continue
				}
				theTime := time.Date(
					tempTime.Year(),
					tempTime.Month(),
					tempTime.Day(),
					tempTime.Hour(),
					tempTime.Minute(),
					tempTime.Second(),
					tempTime.Nanosecond(),
					time.Local)


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
					lastRow = row
					continue
				}
				// run the last timepoint if its the first run.
				if firstRun {
					firstRun = false
					errLog.Println("running firstrun line")
					for i := 0; i < 10; i++ {
						tp, err := NewTimePointFromRow(errLog, lastRow)
						if err != nil{
							errLog.Println(err)
							break
						}
						if runStuff(tp) {
							break
						}
					}
				}

				// we have reached sleeptime
				errLog.Printf("sleeping for %ds\n", int(time.Until(theTime).Seconds()))
				time.Sleep(time.Until(theTime))

				// RUN STUFF HERE
				for i := 0; i < 10; i++ {
					tp, err := NewTimePointFromRow(errLog, lastRow)
					if err != nil{
						errLog.Println(err)
						break
					}
					if runStuff(tp) {
						break
					}
				}
				// end RUN STUFF

			}
		}

	}

	for i, row := range sheet.Rows {
		if i == 0 {
			continue
		}
		tempTime, err := row.Cells[IndexConfig.DatetimeIdx].GetTime(false)

		if err!= nil {
			errLog.Println(err)
			continue
		}
		theTime := time.Date(
			tempTime.Year(),
			tempTime.Month(),
			tempTime.Day(),
			tempTime.Hour(),
			tempTime.Minute(),
			tempTime.Second(),
			tempTime.Nanosecond(),
			time.Local)

		// if we are before the time skip until we are after it
		// the -10s means that we shouldnt run again.
		if theTime.Before(time.Now()) {
			lastRow = row
			continue
		}

		// run the last timepoint if its the first run.
		if firstRun {
			firstRun = false
			errLog.Println("running firstrun line")
			for i := 0; i < 10; i++ {
				tp, err := NewTimePointFromRow(errLog, lastRow)
				if err != nil{
					errLog.Println(err)
					break
				}
				if runStuff(tp) {
					break
				}
			}
		}

		errLog.Printf("sleeping for %ds\n", int(time.Until(theTime).Seconds()))
		time.Sleep(time.Until(theTime))

		// we have reached sleeptime
		errLog.Printf("sleeping for %ds\n", int(time.Until(theTime).Seconds()))
		time.Sleep(time.Until(theTime))

		// RUN STUFF HERE
		for i := 0; i < 10; i++ {
			tp, err := NewTimePointFromRow(errLog, row)
			if err != nil{
				errLog.Println(err)
				break
			}
			if runStuff(tp) {
				break
			}
		}
		// end RUN STUFF
	}
}

// RunConditions runs conditions for a file
func RunConditions(errLog *log.Logger, runStuff func(point *TimePoint) bool, conditionsPath string, loopFirstDay bool) {

	errLog.Printf("running conditions file: %s\n", conditionsPath)

	if filepath.Ext(conditionsPath) == ".xlsx" {
		runFromXlsx(errLog, runStuff, conditionsPath, loopFirstDay)
	}
	if filepath.Ext(conditionsPath) == ".csv" {
		runFromCsv(errLog, runStuff, conditionsPath, loopFirstDay)
	}

}
