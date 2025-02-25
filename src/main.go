package main

import (
	"data_api/client"
	"data_api/server/config"
	"data_api/server/controller"
	"data_api/server/model"
	routes "data_api/server/view"
	"fmt"
	_ "net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

const bucket string = config.BUCKET

//First line is local organisation/token for influxdb, second line is remote org/token in grid5000

const org string = config.LOCAL_ORG

//const org string = config.REMOTE_ORG

const token string = config.LOCAL_TOKEN

//const token string = config.REMOTE_TOKEN

const url string = config.INFLUX_URL

func main() {

	var wg sync.WaitGroup
	//Local :
	db := controller.ConnectDB(config.POSTGRES_USERNAME, config.LOCAL_POSTGRES_PASSWORD,
		config.POSTGRES_HOST, config.POSTGRES_PORT, config.POSTGRES_DB_NAME)

	//Grid5000 :
	//db := controller.ConnectDB(config.POSTGRES_USERNAME, config.REMOTE_POSTGRES_PASSWORD,	config.POSTGRES_HOST, config.POSTGRES_PORT, config.POSTGRES_DB_NAME)
	defer db.Close()

	controller.StartServer(db)

	day := time.Now().Day()

	pointsChan := make(chan model.Point, 10)
	//Local :
	go controller.ReadCsvWhileRunning("C:\\Users\\mattt\\Downloads\\log-"+strconv.Itoa(day)+"_02_2025-Matthieu Theret.csv", "test.json", pointsChan, &wg)
	//Grid5000 :
	//go controller.MonitorEnergy(pointsChan, &wg)
	wg.Add(1)
	go model.PopulateDBFromChan(bucket, org, token, url, pointsChan, &wg)
	wg.Add(1)

	router := gin.Default()
	routes.CreateRoutes(router, db, bucket, org, token, url)
	go router.Run("0.0.0.0:8080") //To accept connections from other IP addresses.

	controller.UserConnection(db, 6)
	fmt.Println("user 1 connected")
	time.Sleep(120 * time.Second)
	controller.UserConnection(db, 8)
	fmt.Println("user 2 connected")
	time.Sleep(10 * time.Second)
	fmt.Println("Starting Fibo")
	go fmt.Print(client.Fibo(50))
	time.Sleep(15 * time.Second)
	go fmt.Println(client.Fibo(51))
	go fmt.Println(client.Fibo(52))

	time.Sleep(15 * time.Second)

	wg.Add(1) //To keep the server running even if no data is added to the csv
	wg.Wait()
	controller.UserDeconnection(db, 1)
	controller.UserDeconnection(db, 2)
	fmt.Println("user 1 & 2 disconnected")
}
