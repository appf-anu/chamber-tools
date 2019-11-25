package main

import (
    "os"
    "fmt"
    "github.com/tealeg/xlsx"
    "time"
)


func main() {
    excelFileName := os.Args[1]

    xlFile, err := xlsx.OpenFile(excelFileName)
    if err != nil {
        panic(err)
    }
    sheet, ok := xlFile.Sheet["timepoints"]
    if !ok {
        fmt.Println("no sheet named \"timepoints\" in xlsx file")
        os.Exit(3)
    }
    for i, row := range sheet.Rows {
        if i == 0 {
            for _, cell := range row.Cells {
                text := cell.String()
                fmt.Printf("%s,", text)
            }
            fmt.Printf("\n")
            continue
        }
        for ci, cell := range row.Cells {
            if ci == 0 || ci == 1 {
                t, err := cell.GetTime(false)
                
                if err != nil{
                    fmt.Println(err)
                }
                nt := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
                fmt.Println(nt)
                fmt.Println(t)
                continue
            }
            text := cell.GetNumberFormat()
            fmt.Printf("%s\n", text)
        }
    }
    
}