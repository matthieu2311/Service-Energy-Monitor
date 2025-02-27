package controller

import (
	"data_api/server/model"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Connects to the postgres db, return a pointer to it
func ConnectDB(username, password, host, port, dbname string) *sql.DB {
	return model.ConnectDB(username, password, host, port, dbname)
}

// Delete all tables inside the postgres db, you should use StartServer after this one to create them again
func Reset(db *sql.DB) {
	model.Reset(db)
}

// Create all the tables if it is the first time the server is launched,
// else end all the previous user sessions that did not end correctly and start a new time-range with 0 users connected.
func StartServer(db *sql.DB) {
	model.DemarrageServeur(db)
}

// Connect a new user to the server, and return the id that was attributed
func NewUserConnection(db *sql.DB) int {
	return model.NewUserConnection(db)
}

// Connect an already known user to the server. If a user with this id does not exists yet, behave like NewUserConnection
func UserConnection(db *sql.DB, id int) {
	model.UserConnection(db, id)
}

// Disconnect a user from the server. If the user wasn't connected, do nothing
func UserDeconnection(db *sql.DB, id int) {
	model.UserDeconnection(db, id)
}

// Gin handler function for the api endpoint. Show the list of all the users known by the server in a json.
// Access it with .../users
func GetUsers(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		users := model.GetUsers(db)
		c.IndentedJSON(http.StatusOK, users)
	}
}

// Gin handler function for the api endpoint. Retrieve a specific user by the id specified in the url.
// Access it with .../users/:id
func GetUserById(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		user := model.GetUserById(db, id)
		c.IndentedJSON(http.StatusOK, user)
	}
}

// Gin handler function for the api endpoint. Retrieve all the links associated with the user specified by the id in the url.
// Access it with .../users/:id/links
func GetUserTimesById(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		userTimes := model.GetUserTimesById(db, id)
		c.IndentedJSON(http.StatusOK, userTimes)
	}
}

// Gin handler function for the api endpoint. Retrieve all the time-ranges since the start of the server.
// Access it with .../plages
func GetTimeRanges(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		timeRanges := model.GetTimeRanges(db)
		c.IndentedJSON(http.StatusOK, timeRanges)
	}
}

// Gin handler function for the api endpoint. Retrieve a specific time-range by the id specified in the url.
// Access it with .../plages/:id
func GetTimerangeById(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		timeRange := model.GetTimerangeById(db, id)
		c.IndentedJSON(http.StatusOK, timeRange)
	}
}

// Gin handler function for the api endpoint. Retrieve some key data about today's consumption.
// Access it with .../users/:id/today
func GetTodayHighlights(db *sql.DB, bucket, org, token, url string) gin.HandlerFunc {
	year := time.Now().Year()
	month := time.Now().Month()
	day := time.Now().Day()
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		today := getTodayHighlights(id, year, day, month, db, bucket, org, token, url)
		c.IndentedJSON(http.StatusOK, today)
	}
}

// Gin handler func : Return a list of all the daily average consumptions since the first connection of the user to the server.
// Access it with .../users/:id/consumption
func GetAllDailyMean(db *sql.DB, bucket, org, token, url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		dailyMeans := getAllDailyMean(id, db, bucket, org, token, url)
		c.IndentedJSON(http.StatusOK, dailyMeans)
	}
}

// Return a gin function that gives the average consumption of each of the 52 last weeks
// Access it with .../users/:id/weeklyMean
func GetWeeklyMean(db *sql.DB, bucket, org, token, url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		weeklyMean := getAllWeeklyMeans(id, db, bucket, org, token, url)
		c.IndentedJSON(http.StatusOK, weeklyMean)
	}
}

// Gin handler function for the api endpoint. Retrieve the rank of the user specified by the id in the url
// among all the users of the server. There are four ranks, corresponding respectively to the :
// rank over this year, this month, this week and today.
// Access it with .../users/:id/rank
func GetRank(db *sql.DB, bucket, org, token, url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		ranks := RankUser(id, db, bucket, org, token, url)
		c.IndentedJSON(http.StatusOK, ranks)
	}
}

func getUserTimes(id int, db *sql.DB) (timeRanges []model.TimeRange) {
	timeRanges = model.GetUserTimes(db, id)
	return timeRanges
}

func worker(tasks <-chan model.TimeRange, results chan<- []model.Point, wg *sync.WaitGroup, bucket, org, token, url string) {
	defer wg.Done()
	for t := range tasks {
		start := t.Start
		stop := time.Now()
		if t.Stop.Valid {
			stop = t.Stop.Time
		}

		influxData := model.GetData(bucket, org, token, url, start, stop)
		for i, elt := range influxData {
			influxData[i] = model.Point{Timestamp: elt.Timestamp, Value: elt.Value / float64(t.NbrUsers)}
		}

		results <- influxData
	}
}

// Get all the points stored in the influx db during the time when the user was connected.
// It uses the subfunction worker to parallelize and accelerate the process.
func getUserEnergyConsumption(id int, db *sql.DB, bucket, org, token, url string) []model.Point {
	defer model.CloseClient()

	var userEnergyC []model.Point
	timeRanges := getUserTimes(id, db) //get all the time-ranges during which the user was connected

	nbrWorkers := 5
	tasks := make(chan model.TimeRange, len(timeRanges))
	results := make(chan []model.Point, len(timeRanges))
	var wg sync.WaitGroup

	for i := 0; i < nbrWorkers; i++ {
		wg.Add(1)
		go worker(tasks, results, &wg, bucket, org, token, url)
	}

	go func() {
		for _, t := range timeRanges {
			tasks <- t // Add these time-ranges to the tasks channel, letting workers treat them
		}
		close(tasks)
	}()

	go func() {
		wg.Wait()
		close(results)
		model.CloseClient()
	}()

	for influxData := range results {
		userEnergyC = append(userEnergyC, influxData...)
	}

	// Allow to create a json file with all data
	//name := fmt.Sprintf("outputUser%d.json", id)
	//createJSONFile(name, userEnergyC)

	return userEnergyC
}

// Return a list of all the daily average consumptions since the first connection of the user to the server.
func getAllDailyMean(id int, db *sql.DB, bucket, org, token, url string) []model.Point {
	defer model.CloseClient()

	var result []model.Point

	firstTimeRange := model.GetEarliestTimeRange(id, db)

	year := firstTimeRange.Start.Year()
	month := firstTimeRange.Start.Month()
	day := firstTimeRange.Start.Day()

	curDay := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	for curDay.Before(time.Now()) {
		result = append(result, getTodayHighlights(id, year, day, month, db, bucket, org, token, url)[3])
		curDay = curDay.Add(24 * time.Hour)
		year = curDay.Year()
		month = curDay.Month()
		day = curDay.Day()
	}

	return result
}

// UNUSED Gin handler function for the api endpoint. Return a list of all the points corresponding to the energy consumption of the user.
func GetUserEnergyConsumption(db *sql.DB, bucket, org, token, url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			fmt.Println(err)
		}
		points := getUserEnergyConsumption(id, db, bucket, org, token, url)
		c.IndentedJSON(http.StatusOK, points)
	}
}

// Delete the user specified by id
func DeleteUser(db *sql.DB, id int) {
	model.DeleteUser(db, id)
}

// Allow to create a json file from a list of points. Can be useful for debugging but is unused otherwise.
func createJSONFile(name string, data []model.Point) {
	f, _ := os.Create(name)
	defer f.Close()
	var jsonData []byte
	var err error
	if jsonData, err = json.MarshalIndent(data, "", "\t"); err != nil {
		log.Fatal(err)
	}
	f.Write(jsonData)
}

// Return an array of points (timestamp, value) corresponding to :
//
// today's maximum consumption, minimum consumption, total consumption, and average consumption.
func getTodayHighlights(id, year, day int, month time.Month, db *sql.DB, bucket, org, token, url string) []model.Point {

	defer model.CloseClient()

	today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	maxMinSumMean := []model.Point{{Timestamp: today, Value: 0}, {Timestamp: today, Value: 1000},
		{Timestamp: today, Value: 0}, {Timestamp: today, Value: 0}}
	meanDivider := 0
	timeRanges := getUserTimes(id, db)

	for _, t := range timeRanges {
		start := t.Start
		stop := time.Now()
		if t.Stop.Valid {
			stop = t.Stop.Time
		}

		if t.Stop.Time.Before(today) || start.After(today.Add(24*time.Hour)) {
			continue
		}
		influxData := model.GetData(bucket, org, token, url, start, stop)

		for _, elt := range influxData {
			if elt.Timestamp.Before(today.Add(24*time.Hour)) && elt.Timestamp.After(today) {

				elt.Value = elt.Value / float64(t.NbrUsers)

				if elt.Value > maxMinSumMean[0].Value {
					maxMinSumMean[0] = elt
				} else if elt.Value < maxMinSumMean[1].Value {
					maxMinSumMean[1] = elt
				}

				maxMinSumMean[2].Value += elt.Value
				maxMinSumMean[3].Value += elt.Value
				meanDivider++
			}
		}
	}
	if maxMinSumMean[1].Value == 1000 {
		maxMinSumMean[1].Value = 0
	}
	if meanDivider > 0 {
		maxMinSumMean[3].Value /= float64(meanDivider)
	}

	return maxMinSumMean
}

// Return an array with the average consumption (per 10s passed on the server) of each of the last 52 weeks.
// The first element of the array is the mean consumption of the actual, ongoing week.
func getAllWeeklyMeans(id int, db *sql.DB, bucket, org, token, url string) [52]float64 {

	defer model.CloseClient()

	weeklyMeansTemp := [52]struct {
		float64 //The sum value of cpu consumption during that week
		int     //The number of points to divide with to obtain the mean
	}{}
	dates := [][]time.Time{}
	t := time.Now().UTC()

	//Create the week intervals
	today := t.Weekday()
	mondayGap := int(today) - 1
	mondayTime := t.Add(-time.Duration(mondayGap*24) * time.Hour).Add(-time.Duration(t.Hour())*time.Hour - time.Duration(t.Minute())*time.Minute - time.Duration(t.Second())*time.Second)
	dates = append(dates, []time.Time{mondayTime, t})
	for range 51 {
		newMondayTime := mondayTime.Add(-time.Duration(7*24) * time.Hour)
		dates = append(dates, []time.Time{newMondayTime, mondayTime})
		mondayTime = newMondayTime
	}

	//Get all data points of the user
	globalUserConsumption := getUserEnergyConsumption(id, db, bucket, org, token, url)

	//Get the data corresponding to the intervals
	for _, point := range globalUserConsumption {
		for i, week := range dates {
			if point.Timestamp.Before(week[1]) && point.Timestamp.After(week[0]) {
				weeklyMeansTemp[i].float64 += point.Value
				weeklyMeansTemp[i].int++ //The coefficient to divide with depends on the number of points,
				// so the mean depends on the monitoring frequency
				break //Only breaks inner loop (hopefully)
			}
		}
	}

	//Do the mean of each intervals and store it into array
	weeklyMeans := [52]float64{}
	for i, elt := range weeklyMeansTemp {
		if elt.int == 0 {
			weeklyMeans[i] = 0
		} else {
			weeklyMeans[i] = elt.float64 / float64(elt.int)
		}
	}

	return weeklyMeans

}

// Return the average consumption (per 10s passed on the server) of this week (from Monday to today)
func getWeeklyMean(id int, db *sql.DB, bucket, org, token, url string) float64 {

	defer model.CloseClient()
	var result float64
	var allPoints []model.Point

	now := time.Now().UTC()
	//Create the week interval
	today := now.Weekday()
	mondayGap := int(today) - 1
	mondayTime := now.Add(-time.Duration(mondayGap*24) * time.Hour).Add(-time.Duration(now.Hour())*time.Hour - time.Duration(now.Minute())*time.Minute - time.Duration(now.Second())*time.Second)

	timeRanges := getUserTimes(id, db)

	for _, t := range timeRanges {

		start := t.Start
		stop := now
		if t.Stop.Valid {
			stop = t.Stop.Time
		}

		if t.Stop.Time.Before(mondayTime) || start.After(now) {
			continue
		}

		for _, elt := range model.GetData(bucket, org, token, url, start, stop) {
			allPoints = append(allPoints, model.Point{Timestamp: elt.Timestamp, Value: elt.Value / float64(t.NbrUsers)})
		}
	}
	for _, elt := range allPoints {
		result += elt.Value
	}
	if result != 0 {
		result /= float64(len(allPoints))
	}

	return result
}

// Return the average month consumption (per 10s passed on the server) for this month (from the 1st of the month to today)
func getMonthlyMean(id int, db *sql.DB, bucket, org, token, url string) float64 {

	defer model.CloseClient()
	var result float64
	var allPoints []model.Point

	now := time.Now().UTC()
	//Create the month interval
	today := now.Day()
	monthGap := int(today) - 1
	monthTime := now.Add(-time.Duration(monthGap*24) * time.Hour).Add(-time.Duration(now.Hour())*time.Hour - time.Duration(now.Minute())*time.Minute - time.Duration(now.Second())*time.Second)

	timeRanges := getUserTimes(id, db)
	for _, t := range timeRanges {

		start := t.Start
		stop := now
		if t.Stop.Valid {
			stop = t.Stop.Time
		}

		if t.Stop.Time.Before(monthTime) || start.After(now) {
			continue
		}

		for _, elt := range model.GetData(bucket, org, token, url, start, stop) {
			allPoints = append(allPoints, model.Point{Timestamp: elt.Timestamp, Value: elt.Value / float64(t.NbrUsers)})
		}
	}
	for _, elt := range allPoints {
		result += elt.Value
	}
	if result != 0 {
		result /= float64(len(allPoints))
	}

	return result
}

// Return the average consumption (per 10s passed on the server) during this civil year (from January, 1st to today)
func getYearlyMean(id int, db *sql.DB, bucket, org, token, url string) float64 {

	defer model.CloseClient()
	var result float64
	var allPoints []model.Point

	now := time.Now().UTC()

	//Create the year interval
	yearTime := time.Date(time.Now().Year(), time.January, 1, 0, 0, 0, 0, time.UTC)

	timeRanges := getUserTimes(id, db)

	for _, t := range timeRanges {

		start := t.Start
		stop := now
		if t.Stop.Valid {
			stop = t.Stop.Time
		}

		if t.Stop.Time.Before(yearTime) || start.After(now) {
			continue
		}

		for _, elt := range model.GetData(bucket, org, token, url, start, stop) {
			allPoints = append(allPoints, model.Point{Timestamp: elt.Timestamp, Value: elt.Value / float64(t.NbrUsers)})
		}
	}
	for _, elt := range allPoints {
		result += elt.Value
	}
	if result != 0 {
		result /= float64(len(allPoints))
	}

	return result
}

// Return means in the following order : mean over the year, mean over the last month, over the last
// week and over the last day (!not the last 24h!).
// All means are expressed in mWh/10s or J/10s depending of the version (so the average consumption for 10s passed on the server).
func getAllMeans(id int, db *sql.DB, bucket, org, token, url string) []float64 {

	yMWDMeans := []float64{}

	yMWDMeans = append(yMWDMeans, getYearlyMean(id, db, bucket, org, token, url))
	yMWDMeans = append(yMWDMeans, getMonthlyMean(id, db, bucket, org, token, url))
	yMWDMeans = append(yMWDMeans, getWeeklyMean(id, db, bucket, org, token, url))
	year := time.Now().Year()
	month := time.Now().Month()
	day := time.Now().Day()
	yMWDMeans = append(yMWDMeans, getTodayHighlights(id, year, day, month, db, bucket, org, token, url)[3].Value)

	return yMWDMeans
}

// Return an array with the different rankings of the user with id "id".
// For example, if a user is the least consumer of energy (on average) on the server, his rank will be 1.
// Since the consumption can vary depending of the time of the year, this program computes the rank for different periods of time.
// It also add the total number of users, to allow comparisons and percentages.
// The elements of the array corresponds respectively to : the year rank, the month rank, the week rank,
// the daily rank and the total number of users in the database.
func RankUser(id int, db *sql.DB, bucket, org, token, url string) []int {

	type Mean struct {
		value float64
		id    int
	}

	ranks := []int{}
	yearMeans := []Mean{}
	monthMeans := []Mean{}
	weekMeans := []Mean{}
	dayMeans := []Mean{}
	ids := model.GetUsersIDs(db)
	if !slices.Contains(ids, id) {
		fmt.Println("This user is not registered in the database ! ")
		log.Fatal()
	}

	for _, id := range ids {
		temp := getAllMeans(id, db, bucket, org, token, url)
		yearMeans = append(yearMeans, Mean{value: temp[0], id: id})
		monthMeans = append(monthMeans, Mean{value: temp[1], id: id})
		weekMeans = append(weekMeans, Mean{value: temp[2], id: id})
		dayMeans = append(dayMeans, Mean{value: temp[3], id: id})
	}

	cmp := func(i, j Mean) int {
		if i.value == j.value {
			return 0
		} else if i.value < j.value {
			return -1
		}
		return 1
	}

	slices.SortFunc(yearMeans, cmp)
	slices.SortFunc(monthMeans, cmp)
	slices.SortFunc(weekMeans, cmp)
	slices.SortFunc(dayMeans, cmp)

	ind := func(i Mean) bool {
		return i.id == id
	}

	nbrUsers := len(yearMeans)
	yearRank := slices.IndexFunc(yearMeans, ind) + 1
	monthRank := slices.IndexFunc(monthMeans, ind) + 1
	weekRank := slices.IndexFunc(weekMeans, ind) + 1
	dayRank := slices.IndexFunc(dayMeans, ind) + 1

	ranks = append(ranks, yearRank, monthRank, weekRank, dayRank, nbrUsers)

	return ranks
}
