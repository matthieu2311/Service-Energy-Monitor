package controller

import (
	"data_api/server/model"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// Not verified so may not work, you should use ReadCsvWhileRunning which does the same thing but better.
func ReadCsv(fileName string) []model.Point {

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'

	var points []model.Point
	var sum float64
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		l := len(rec)
		val, _ := strconv.ParseFloat(rec[l-1], 64)
		sum += val
		if rec[1] == "CPU Energy" {

			timestamp, _ := strconv.Atoi(rec[0])
			t := time.Unix(int64(timestamp), 0).UTC()
			points = append(points, model.Point{Timestamp: t, Value: sum})
			sum = 0
		}

	}

	return points
}

// Reads the CSV produced by DEMETER and inserts them into the given channel. It will wait if the line isn't complete
// (because DEMETER is still writing it) or if there isn't any new point written yet.
// After 40s without new data added to the csv, it ends without error.
//
// This function is made specially to work with DEMETER, and will probably need some changes in order to work
// with all .csv files.
// Here is the expected format of csv with and example :
// ------------------------------------------------------------------------------------------------------
// |Timestamp (UNIX) | Monitored process name | Some more columns... | Total energy consumption (in mWh)|
// |                 |                        |                      |                                  |
// |1740389314       |Explorer.exe			  |          ...         | 0.209884                         |
// ...
// ... more lines with monitored processes at that timestamp
// ...
// |1740389314       |CPU Energy              |          ...         | 12.543352                        |
// |Next timestamp   |Explorer.exe            |          ...         | ...                              |
//
// The last row of each batch of processes monitored needs to be called CPU Energy before going to the next batch.
// It is designed to ignore the RESTART LINE of DEMETER csv files.
func ReadCsvWhileRunning(csvFileName string, pointsChan chan model.Point, wg *sync.WaitGroup, debug bool) {

	defer wg.Done()
	var running bool = true
	var counter int = 0
	var exportFile *os.File

	csvFile, err := os.Open(csvFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	reader.Comma = ';'

	if debug {
		exportFile, err := os.OpenFile("debug.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer exportFile.Close()
	}

	var sum float64

	for running {
		rec, err := reader.Read()
		if len(rec) == 0 {
			fmt.Println("Waiting for more data...")
			counter++
			if counter > 3 {
				close(pointsChan)
				break
			}
			time.Sleep(10 * time.Second)

		} else if err == io.EOF {
			close(pointsChan)
			break

		} else if err != nil {
			if errors.Is(err, csv.ErrFieldCount) {
				time.Sleep(10 * time.Second)
			} else {
				fmt.Print(err)
				log.Fatal(err)
			}

		} else {
			counter = 0
			l := len(rec)
			val, _ := strconv.ParseFloat(rec[l-1], 64)
			sum += val
			if rec[1] == "CPU Energy" {

				timestamp, _ := strconv.Atoi(rec[0])
				t := time.Unix(int64(timestamp), 0).UTC()
				p, err := json.MarshalIndent(model.Point{Timestamp: t, Value: sum}, "", "\t")
				if err != nil {
					log.Fatal(err)
				}
				if debug {
					exportFile.Write(p)
					exportFile.Write([]byte{',', '\n'})
				}
				pointsChan <- model.Point{Timestamp: t, Value: sum}
				sum = 0
			}
		}

	}

}
