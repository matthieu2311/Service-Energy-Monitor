package model

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

type User struct {
	ID            int          `json:"id"`
	Start_session time.Time    `json:"start_session"`
	End_session   sql.NullTime `json:"end_session"`
}

type TimeRange struct {
	ID       int          `json:"id"`
	Start    time.Time    `json:"start"`
	Stop     sql.NullTime `json:"stop"`
	NbrUsers int          `json:"nbrUsers"`
}

type Link struct {
	ID           int           `json:"id"`
	UserID       int           `json:"userid"`
	StartPlageID int           `json:"startPlageID"`
	EndPlageID   sql.NullInt32 `json:"endPlageID"`
}

func ConnectDB(username, password, host, port, dbname string) *sql.DB {
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbname)
	db, err := sql.Open("postgres", connString)

	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("Couldn't achieve connection with database... ")
	}

	return db
}

func GetUsers(db *sql.DB) []User {

	users := []User{}
	rows, err := db.Query("select * from users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Start_session, &u.End_session); err != nil {
			log.Fatal(err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return users
}

func GetUsersIDs(db *sql.DB) []int {
	usersIDs := []int{}
	var curID int

	rows, err := db.Query("select id from users")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		rows.Scan(&curID)
		usersIDs = append(usersIDs, curID)
	}
	return usersIDs
}

func GetUserById(db *sql.DB, id int) User {
	var u User
	row := db.QueryRow("select * from users where id = $1", id)
	if err := row.Scan(&u.ID, &u.Start_session, &u.End_session); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("No user with this ID !")
		} else {
			log.Fatal(err)
		}
	}
	return u
}

func GetUserTimesById(db *sql.DB, id int) []Link {
	var l []Link
	rows, err := db.Query("select * from link where userID = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("No user with this ID !")
		} else {
			log.Fatal(err)
		}
	}
	for rows.Next() {
		var lTemp Link
		if err := rows.Scan(&lTemp.ID, &lTemp.UserID, &lTemp.StartPlageID, &lTemp.EndPlageID); err != nil {
			log.Fatal(err)
		}
		l = append(l, lTemp)
	}
	return l
}

func GetUserTimes(db *sql.DB, id int) (timeRanges []TimeRange) {

	var startPlageID int
	var endPlageID sql.NullInt64

	timeRangeRows, err := db.Query("select startPlageID, endPlageID from link where userID = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("No user with this ID !")
		} else {
			log.Fatal(err)
		}
	}

	for timeRangeRows.Next() {

		if err := timeRangeRows.Scan(&startPlageID, &endPlageID); err != nil {
			log.Fatal(err)
		}
		if !endPlageID.Valid {
			db.QueryRow("select id from plages where stop is null").Scan(&endPlageID.Int64)
		}

		for i := startPlageID; i <= int(endPlageID.Int64); i++ {
			var t TimeRange
			temp := db.QueryRow("select * from plages where id = $1", i)
			if err := temp.Scan(&t.ID, &t.Start, &t.Stop, &t.NbrUsers); err != nil {
				log.Fatal(err)
			}

			timeRanges = append(timeRanges, t)
		}
	}

	return timeRanges
}

func GetTimeRanges(db *sql.DB) []TimeRange {
	timeRanges := []TimeRange{}
	rows, err := db.Query("select * from plages")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var t TimeRange
		if err := rows.Scan(&t.ID, &t.Start, &t.Stop, &t.NbrUsers); err != nil {
			log.Fatal(err)
		}
		timeRanges = append(timeRanges, t)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return timeRanges
}

func GetEarliestTimeRange(id int, db *sql.DB) TimeRange {

	var tID int
	var t TimeRange
	row := db.QueryRow("select startPlageID from link where userID = $1 order by id limit 1", id)
	if err := row.Scan(&tID); err != nil {
		log.Fatal(err)
	}

	row2 := db.QueryRow("select * from plages where id = $1", tID)
	if err := row2.Scan(&t.ID, &t.Start, &t.Stop, &t.NbrUsers); err != nil {
		log.Fatal(err)
	}

	return t

}

func GetTimerangeById(db *sql.DB, id int) TimeRange {
	var t TimeRange
	row := db.QueryRow("select * from plages where id = $1", id)
	if err := row.Scan(&t.ID, &t.Start, &t.Stop, &t.NbrUsers); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("No timerange with this ID !")
		} else {
			log.Fatal(err)
		}
	}
	return t
}
