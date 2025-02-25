package routes

import (
	"data_api/server/controller"
	"database/sql"

	"github.com/gin-gonic/gin"
)

// Create all the endpoints for the gin router, and associate them with the correct functions from the controller package.
func CreateRoutes(router *gin.Engine, db *sql.DB, bucket, org, token, url string) {
	router.GET("/users", controller.GetUsers(db))
	router.GET("/users/:id", controller.GetUserById(db))
	router.GET("/users/:id/links", controller.GetUserTimesById(db))
	router.GET("/plages", controller.GetTimeRanges(db))
	router.GET("/plages/:id", controller.GetTimerangeById(db))
	router.GET("/users/:id/consumption", controller.GetAllDailyMean(db, bucket, org, token, url))
	router.GET("/users/:id/today", controller.GetTodayHighlights(db, bucket, org, token, url))
	router.GET("/users/:id/weeklyMean", controller.GetWeeklyMean(db, bucket, org, token, url))
}
