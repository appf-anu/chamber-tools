package chamber_tools

import (
	"bufio"
	"fmt"
	"github.com/bcampbell/fuzzytime"
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
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

// it is extremely unlikely (see. impossible) that we will be measuring or sending a humidity of 214,748,365 %RH or
// a temperature of -340,282,346,638,528,859,811,704,183,484,516,925,440°C until we invent some new physics, so
// until then, I will use these values as the unset or null values for HumidityTarget and TemperatureTarget
const	NullTargetInt     int     = math.MinInt32
const	NullTargetFloat64 float64 = -math.MaxFloat32


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
	Datetime         time.Time
	SimDatetime      time.Time
	Temperature      float64
	RelativeHumidity float64
	Light1           int
	Light2           int
	CO2              float64
	TotalSolar       float64
	Channels         []float64
}


func (tp TimePoint) NulledString() string{
	repr := fmt.Sprintf("%+v", tp)
	nullTargetFloatStr := fmt.Sprintf("%v", NullTargetFloat64)
	repr = strings.Replace(repr, nullTargetFloatStr,"NULL", -1)
	nullTargetIntStr := fmt.Sprintf("%v", NullTargetInt)
	repr = strings.Replace(repr, nullTargetIntStr,"NULL", -1)
	return repr
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
		errLog.Printf("xslx, tried to extract valid time object from this string \"%s\", error: %v",
			tString, err)
		errLog.Printf("couldn't extract datetime from xlsx file: %s\n", err)
		return time.Time{}, err
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
				field.Set(reflect.Zero(field.Type()))
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
		if err != nil {
			log.Fatal(err)
		}
		sheet, ok := xlFile.Sheet["timepoints"]
		if !ok {
			fmt.Println("no sheet named \"timepoints\" in xlsx file")
			os.Exit(3)
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

func NewTimePointFromStringArray(errLog *log.Logger, row []string) (*TimePoint, error) {
	tp := &TimePoint{}
	for i, cell := range row {

		if i == IndexConfig.DatetimeIdx {
			t, err := parseDateTime(cell, errLog)
			if err != nil {
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
				errLog.Println("failed parsing humidity float")
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
				errLog.Println("failed parsing CO2 float")
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
				errLog.Println("failed parsing TotalSolar float")
				return nil, err
			}
			tp.TotalSolar = t
		}
		if i == IndexConfig.Light1Idx {
			found := strings.TrimSpace(cell)
			t, err := strconv.ParseInt(found, 10, 64)
			if err != nil {
				errLog.Println("failed parsing Light1 int")
				return nil, err
			}
			tp.Light1 = int(t)
		}
		if i == IndexConfig.Light2Idx {
			found := strings.TrimSpace(cell)
			t, err := strconv.ParseInt(found, 10, 64)
			if err != nil {
				errLog.Println("failed parsing Light2 int")
				return nil, err
			}
			tp.Light2 = int(t)
		}
	}
	// do channels

	for chaNumber, chanIdx := range IndexConfig.ChannelsIdx {
		v := row[chanIdx]
		found := matchFloat.FindString(v)
		if len(found) < 0 {
			errLog.Printf("couldnt parse channel-%d \"%s\" as a float\n", chaNumber+1, v)
			continue
		}
		chanValue, err := strconv.ParseFloat(found, 64)
		if err != nil {
			errLog.Println(err)
			tp.Channels = append(tp.Channels, -1.0)
			continue
		}
		tp.Channels = append(tp.Channels, chanValue)
	}
	return tp, nil

}

func NewTimePointFromRow(errLog *log.Logger, row *xlsx.Row) (*TimePoint, error) {
	tp := &TimePoint{
		Temperature     :  NullTargetFloat64,
		RelativeHumidity : NullTargetFloat64,
		Light1           : NullTargetInt,
		Light2           : NullTargetInt,
		CO2              : NullTargetFloat64,
		TotalSolar       : NullTargetFloat64,
	}
	for i, cell := range row.Cells {

		if i == IndexConfig.DatetimeIdx {
			t, err := cell.GetTime(false)
			if err != nil {
				return nil, err
			}
			tp.Datetime = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
				time.Local)
		}
		if i == IndexConfig.SimDatetimeIdx {
			t, err := cell.GetTime(false)
			if err != nil {
				errLog.Println("Couldn't get SimDatetime from row")
				continue
			}
			tp.SimDatetime = t
		}
		if i == IndexConfig.TemperatureIdx {
			if cell.String() == "" || cell.String() == "NULL" {
				tp.Temperature = NullTargetFloat64
				continue
			}
			t, err := cell.Float()
			if err != nil {
				return nil, err
			}
			tp.Temperature = t

		}
		if i == IndexConfig.HumidityIdx {
			if cell.String() == "" || cell.String() == "NULL" {
				tp.RelativeHumidity = NullTargetFloat64
				continue
			}
			t, err := cell.Float()
			if err != nil {
				return nil, err
			}
			tp.RelativeHumidity = t

		}
		if i == IndexConfig.CO2Idx {
			if cell.String() == "" || cell.String() == "NULL" {
				tp.CO2 = NullTargetFloat64
				continue
			}
			t, err := cell.Float()
			if err != nil {
				return nil, err
			}
			tp.CO2 = t

		}
		if i == IndexConfig.TotalSolarIdx {
			if cell.String() == "" || cell.String() == "NULL" {
				tp.TotalSolar = NullTargetFloat64
				continue
			}
			t, err := cell.Float()
			if err != nil {
				errLog.Println("Couldn't get TotalSolar from row")
				continue
			}
			tp.TotalSolar = t

		}
		if i == IndexConfig.Light1Idx {
			if cell.String() == "" || cell.String() == "NULL" {
				tp.Light1 = NullTargetInt
				continue
			}
			t, err := cell.Int()
			if err != nil {
				return nil, err
			}
			tp.Light1 = t

		}
		if i == IndexConfig.Light2Idx {
			if cell.String() == "" || cell.String() == "NULL" {
				tp.Light2 = NullTargetInt
				continue
			}

			t, err := cell.Int()
			if err != nil {
				return nil, err
			}
			tp.Light2 = t

		}
	}
	// do channels
	for _, chanIdx := range IndexConfig.ChannelsIdx {
		cell := row.Cells[chanIdx]
		// handle NULL targets
		if cell.String() == "" || cell.String() == "NULL" {
			tp.Channels = append(tp.Channels, NullTargetFloat64)
			continue
		}

		chanValue, err := cell.Float()
		// we still need to append a null target if we cant get the value
		if err != nil {
			tp.Channels = append(tp.Channels, NullTargetFloat64)
			continue
		}
		tp.Channels = append(tp.Channels, chanValue)
	}
	return tp, nil

}

func loopFromCsv(errLog *log.Logger, runStuff func(point *TimePoint) bool, conditionsPath string) {
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
		for i, line := range data {
			lineSplit := strings.Split(line, ",")
			timeStr := lineSplit[0]
			tempTime, err := parseDateTime(timeStr, errLog)
			if err != nil {
				errLog.Println(err)
				continue
			}

			now := time.Now()

			theTime := time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				tempTime.Hour(),
				tempTime.Minute(),
				tempTime.Second(),
				tempTime.Nanosecond(),
				time.Local)

			if i == len(data)-1 { // end of data, set the next time to tomorrow
				lastLineSplit = lineSplit
				timeStr = strings.Split(data[0], ",")[0]
				tempTime, err = parseDateTime(timeStr, errLog)
				if err != nil {
					errLog.Println(err)
					continue
				}
				theTime = time.Date(
					now.Year(),
					now.Month(),
					now.Day(),
					tempTime.Hour(),
					tempTime.Minute(),
					tempTime.Second(),
					tempTime.Nanosecond(),
					time.Local).Add(time.Hour * 24)

				errLog.Println("Reached end of data, set time to ", theTime)
			} else {
				// check if theTime is Before
				if theTime.Before(time.Now()) {
					lastLineSplit = lineSplit
					continue
				}
			}

			// run the last timepoint if its the first run.
			if firstRun {
				firstRun = false
				errLog.Println("running firstrun line")
				for i := 0; i < 10; i++ {
					tp, err := NewTimePointFromStringArray(errLog, lastLineSplit)
					if err != nil {
						errLog.Println(err)
						break
					}
					// print the point:
					errLog.Printf("TimePoint: %s", tp.NulledString())
					if runStuff(tp) {
						break
					}
				}
			}

			// we have reached
			errLog.Printf("sleeping for %s\n", time.Until(theTime).String())
			time.Sleep(time.Until(theTime))

			// RUN STUFF HERE
			for i := 0; i < 10; i++ {
				tp, err := NewTimePointFromStringArray(errLog, lineSplit)
				if err != nil {
					errLog.Println(err)
					break
				}
				errLog.Printf("TimePoint: %s", tp.NulledString())
				if runStuff(tp) {
					break
				}
			}

		}
	}
}

func runFromCsv(errLog *log.Logger, runStuff func(point *TimePoint) bool, conditionsPath string) {
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
				if err != nil {
					errLog.Println(err)
					break
				}
				errLog.Printf("TimePoint: %s", tp.NulledString())
				if runStuff(tp) {
					break
				}
			}
		}

		errLog.Printf("sleeping for %s\n", time.Until(theTime).String())
		time.Sleep(time.Until(theTime))

		// RUN STUFF HERE
		for i := 0; i < 10; i++ {
			tp, err := NewTimePointFromStringArray(errLog, lineSplit)
			if err != nil {
				errLog.Println(err)
				break
			}
			errLog.Printf("TimePoint: %s", tp.NulledString())
			if runStuff(tp) {
				break
			}
		}
		// end RUN STUFF
		idx++
	}
}

func loopFromXlsx(errLog *log.Logger, runStuff func(point *TimePoint) bool, conditionsPath string) {
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

	var lastTp *TimePoint
	var lastTpIdx int
	firstRun := true

	var firstTime time.Time
	data := make([]*TimePoint, 0)
	for i, row := range sheet.Rows {
		if i == 0 {
			continue
		}
		// skip rows with less than 2 cells
		if len(row.Cells) < 2 {
			errLog.Printf("row %05d has less than 2 cells", i)
			continue
		}
		if row.Cells[IndexConfig.DatetimeIdx].String() == "" {
			errLog.Printf("row %05d has empty datetime cell", i)
			continue
		}

		tempTime, err := row.Cells[IndexConfig.DatetimeIdx].GetTime(false)
		if err != nil {
			errLog.Printf("error while extracting time value for row %d, %v", i, err)
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
		if firstTime.Unix() <= 0 { // check to see if firsttime has been set
			firstTime = theTime
		}
		if theTime.After(firstTime.Add(time.Hour * 24)) {
			break
		}
		tp, err := NewTimePointFromRow(errLog, row)
		if err != nil {
			errLog.Printf(" Error adding timepoint to loop data: %v", err)
			continue
		}
		data = append(data, tp)
	}

	totalTimepoints := len(data)-1
	errLog.Printf("looping over %d timepoints", totalTimepoints)
	for {
		for i, tp := range data {
			thisTpIdx := i
			now := time.Now()

			theTime := time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				tp.Datetime.Hour(),
				tp.Datetime.Minute(),
				tp.Datetime.Second(),
				tp.Datetime.Nanosecond(),
				time.Local)


			if i == len(data)-1 { // end of data, set the next time to tomorrow
				lastTp = tp
				lastTpIdx = i
				thisTpIdx = 0
				theTime = time.Date(
					now.Year(),
					now.Month(),
					now.Day(),
					data[0].Datetime.Hour(),
					data[0].Datetime.Minute(),
					data[0].Datetime.Second(),
					data[0].Datetime.Nanosecond(),
					time.Local).Add(time.Hour * 24)
				errLog.Printf("Reached end of data, looping from beginning. next TimePoint at %v ", theTime)
			} else {
				// check if theTime is Before
				if theTime.Before(time.Now()) {
					lastTp = tp
					lastTpIdx = i
					continue
				}
			}

			// run the last timepoint if its the first run.
			if firstRun {
				firstRun = false
				errLog.Printf("running initial TimePoint %05d/%05d", lastTpIdx, totalTimepoints)
				for i := 0; i < 10; i++ {
					errLog.Printf("TimePoint: %s", lastTp.NulledString())
					if runStuff(lastTp) {
						break
					}
				}
			}

			// we have reached sleeptime
			errLog.Printf("sleeping for %s until TimePoint %05d/%05d at %v",
				time.Until(theTime).String(), thisTpIdx, totalTimepoints, tp.Datetime)
			time.Sleep(time.Until(theTime))

			// RUN STUFF HERE
			for try := 0; try < 10; try++ {
				errLog.Printf("running TimePoint %05d/%05d", lastTpIdx, totalTimepoints)
				errLog.Printf("TimePoint: %s", tp.NulledString())
				if runStuff(tp) {
					break
				}
			}
			// end RUN STUFF
		}
	}
}

func runFromXlsx(errLog *log.Logger, runStuff func(point *TimePoint) bool, conditionsPath string) {
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

	for i, row := range sheet.Rows {
		if i == 0 {
			continue
		}
		// skip rows with less than 2 cells
		if len(row.Cells) < 2 {
			errLog.Printf("row %05d has less than 2 cells", i)
			continue
		}
		if row.Cells[IndexConfig.DatetimeIdx].String() == "" {
			errLog.Printf("row %05d has empty datetime cell", i)
			continue
		}
		tempTime, err := row.Cells[IndexConfig.DatetimeIdx].GetTime(false)

		if err != nil {
			errLog.Printf("error while extracting time value for row %d, %v", i, err)
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
				if err != nil {
					errLog.Println(err)
					break
				}
				errLog.Printf("TimePoint: %s", tp.NulledString())
				if runStuff(tp) {
					break
				}
			}
		}

		// we have reached sleeptime
		errLog.Printf("sleeping for %s\n", time.Until(theTime).String())
		time.Sleep(time.Until(theTime))

		// RUN STUFF HERE
		for i := 0; i < 10; i++ {
			tp, err := NewTimePointFromRow(errLog, row)
			if err != nil {
				errLog.Println(err)
				break
			}
			errLog.Printf("TimePoint: %s", tp.NulledString())
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
		if loopFirstDay {
			loopFromXlsx(errLog, runStuff, conditionsPath)
		}
		runFromXlsx(errLog, runStuff, conditionsPath)
	}
	if filepath.Ext(conditionsPath) == ".csv" {
		if loopFirstDay {
			loopFromCsv(errLog, runStuff, conditionsPath)
		}
		runFromCsv(errLog, runStuff, conditionsPath)
	}

}
