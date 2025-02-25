package model

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type Point struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

var client influxdb2.Client

func CloseClient() {
	client.Close()
}

func worker(id int, wg *sync.WaitGroup, pointsChan <-chan *write.Point, client influxdb2.Client, org, bucket string) {

	defer wg.Done()
	writeAPI := client.WriteAPIBlocking(org, bucket)

	for points := range pointsChan {
		err := writeAPI.WritePoint(context.Background(), points)
		if err != nil {
			log.Printf("Worker %d: Error writing to InfluxDB: %v\n", id, err)
		}
	}
}

func PopulateDBFromPoints(bucket, org, token, url string, data []Point) {
	if client == nil {
		client = influxdb2.NewClient(url, token)
	}

	numWorkers := 5
	pointsChan := make(chan *write.Point, numWorkers)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, &wg, pointsChan, client, org, bucket)
	}

	for _, p := range data {
		t := p.Timestamp
		value := p.Value
		point := influxdb2.NewPointWithMeasurement("energy").
			AddField("energyConsumption", value).
			SetTime(t)
		pointsChan <- point
	}
	close(pointsChan) // Close channel to signal workers to exit
	wg.Wait()
}

func PopulateDBFromChan(bucket, org, token, url string, pointsChan chan Point, wg *sync.WaitGroup) {
	defer wg.Done()
	if client == nil {
		client = influxdb2.NewClient(url, token)
	}

	writeAPI := client.WriteAPIBlocking(org, bucket)

	for p := range pointsChan {
		t := p.Timestamp
		value := p.Value
		point := influxdb2.NewPointWithMeasurement("energy").AddField("energyConsumption", value).SetTime(t)
		err := writeAPI.WritePoint(context.Background(), point)
		if err != nil {
			fmt.Printf("Problem when writing point : %s", err.Error())
		}
	}

}

func PopulateFakeDB(bucket, org, token, url string) {
	if client == nil {
		client = influxdb2.NewClient(url, token)
	}

	numWorkers := 5
	pointsChan := make(chan *write.Point, numWorkers)
	var wg sync.WaitGroup
	numPoints := 1200

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, &wg, pointsChan, client, org, bucket)
	}

	value := rand.Float64()
	for i := 0; i < numPoints; i++ {
		t := time.Now().Add(time.Duration(1*i) * time.Second).UTC()
		value = rand.Float64()*value*2 + 0.2
		point := influxdb2.NewPointWithMeasurement("energy").
			AddField("energyConsumption", value).
			SetTime(t)
		pointsChan <- point
	}
	close(pointsChan) // Close channel to signal workers to exit
	wg.Wait()
}

func GetData(bucket, org, token, url string, start, stop time.Time) []Point {
	if client == nil {
		client = influxdb2.NewClient(url, token)
	}

	var energy []Point
	queryAPI := client.QueryAPI(org)
	startS, stopS := start.UTC().Format(time.RFC3339Nano), stop.UTC().Format(time.RFC3339Nano)
	query := `from(bucket: "` + bucket + `")
					|> range(start: ` + startS + `, stop: ` + stopS + `)
					|> filter(fn: (r) => r._measurement == "energy")
					|> filter(fn: (r) => r["_field"] == "energyConsumption")`

	dataChan := make(chan Point, 1000)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	go func() {
		defer close(dataChan)
		result, err := queryAPI.Query(context.Background(), query)
		if err != nil {
			errChan <- err
			return
		}
		for result.Next() {
			if v, ok := result.Record().Value().(float64); ok {
				dataChan <- Point{Value: v, Timestamp: result.Record().Time()}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for point := range dataChan {
			energy = append(energy, point)
		}
	}()

	wg.Wait()
	select {
	case err := <-errChan:
		log.Println("Query error:", err)
	default:
	}

	return energy
}

func GetTodayHighlights(bucket, org, token, url string) []Point {
	if client == nil {
		client = influxdb2.NewClient(url, token)
	}

	var maxMinSum []Point
	var names = []string{"max", "min", "sum"}
	queryAPI := client.QueryAPI(org)

	for _, name := range names {
		query := `from(bucket: "` + bucket + `")
		|> range(start: -24h)
		|> filter(fn: (r) => r._measurement == "energy")
		|> filter(fn: (r) => r["_field"] == "energyConsumption")
		|> ` + name + `()`

		result, err := queryAPI.Query(context.Background(), query)
		if err != nil {
			log.Fatal(err)
		}
		for result.Next() {
			maxMinSum = append(maxMinSum, Point{Timestamp: result.Record().Time(), Value: result.Record().Value().(float64)})
		}
	}
	return maxMinSum
}

func GetWeeklyMean(bucket, org, token, url string) []Point {
	if client == nil {
		client = influxdb2.NewClient(url, token)
	}

	var weeklyMean []Point
	queryAPI := client.QueryAPI(org)

	query := `import "date"

				from(bucket: "` + bucket + `")
  					|> range(start: -1y)  
  					|> filter(fn: (r) => r["_measurement"] == "energy")
  					|> filter(fn: (r) => r["_field"] == "energyConsumption")
  					|> aggregateWindow(every: 1w, fn: mean, offset:-3d)
  					|> yield(name: "mean")`

	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		log.Fatal(err)
	}
	for result.Next() {
		var value float64
		resValue := result.Record().Value()
		if resValue == nil {
			value = 0
		} else {
			value = resValue.(float64)
		}
		weeklyMean = append(weeklyMean, Point{Timestamp: result.Record().Time(), Value: value})
	}

	return weeklyMean
}
