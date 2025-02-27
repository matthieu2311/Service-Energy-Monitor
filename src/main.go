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

	//controller.Reset(db) Reset the postgres db (delete all the tables)
	controller.StartServer(db) //Create the tables if needed, and close any previous sessions that didn't end correctly

	day := time.Now().Day() //To get today's DEMETER csv

	pointsChan := make(chan model.Point, 10) //Used for relaying the points between the csv and the database

	//Local (DEMETER) :
	go controller.ReadCsvWhileRunning("C:\\Users\\mattt\\Downloads\\log-"+strconv.Itoa(day)+"_02_2025-Matthieu Theret.csv", pointsChan, &wg, false)
	//Grid5000 (RAPL) :
	//go controller.MonitorEnergy(pointsChan, &wg)
	wg.Add(1)

	go model.PopulateDBFromChan(bucket, org, token, url, pointsChan, &wg) //Inserts the points from the channel into the influxdb
	wg.Add(1)

	router := gin.Default() //Simulate a local server
	routes.CreateRoutes(router, db, bucket, org, token, url)
	go router.Run("0.0.0.0:8080") //To accept connections from other IP addresses.

	for i := range 10 {
		controller.UserConnection(db, i) //Simulate 10 new users that connect to the server
	}

	fmt.Println("Starting Fibo")
	fmt.Println(client.Fibo(50)) //Simulate some computations done on the server by the users
	time.Sleep(15 * time.Second)
	go client.Fibo(50) //Some more intense computations now with parallelization
	go client.Fibo(50)

	time.Sleep(15 * time.Second)

	for i := range 10 {
		controller.UserDeconnection(db, i) //Disconnect the users
	}

	wg.Add(1) //To keep the server running even if no data is added to the csv
	wg.Wait()

}
