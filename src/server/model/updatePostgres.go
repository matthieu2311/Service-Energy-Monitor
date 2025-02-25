package model

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func Reset(db *sql.DB) {
	if _, err := db.Exec("DROP TABLE IF EXISTS users CASCADE"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS plages CASCADE"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS link CASCADE"); err != nil {
		log.Fatal(err)
	}
}

func DemarrageServeur(db *sql.DB) {

	fmt.Println("-------------- Restarting server... ------------------")

	// Creating the tables if it is the first time the server starts
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		start_session TIMESTAMP NOT NULL,
		end_session TIMESTAMP)
						`); err != nil {
		log.Fatal(err)
	}

	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS plages (
		id serial PRIMARY KEY,
		start TIMESTAMP,
		stop TIMESTAMP,
		nbr_users integer)
`); err != nil {
		log.Fatal(err)
	}

	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS link (
		id serial PRIMARY KEY,
		userID integer NOT NULL references users (id),
		startPlageID integer NOT NULL references plages (id),
		endPlageID integer references plages (id)) 
		`); err != nil {
		log.Fatal(err)
	}

	var lastPlageID int

	if err := db.QueryRow("select id from plages where stop is null").Scan(&lastPlageID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return
		} else {
			log.Fatal("Database error:", err)
		}
	}

	if _, err := db.Exec("update link set endPlageID = $1 where endPlageID is null", lastPlageID); err != nil {
		fmt.Print(err)
	}
	var t time.Time
	if err := db.QueryRow("update plages set stop = $1 where id = $2 returning stop", time.Now().UTC(), lastPlageID).Scan(&t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("This shouldn't happen")
		} else {
			log.Fatal("Database error:", err)
		}
	}

	if _, err := db.Exec("insert into plages (start, stop, nbr_users) values ($1, null, 0)", t); err != nil {
		fmt.Print(err)
	}

}

// Doesn't delete the user, but forgets to which links he was associated. Some links are now tied to a null user.
// However if done more than once, the null-user links cannot be differentiated.
func DissociateUser(db *sql.DB, id int) {
	if _, err := db.Exec("update link set userID = null where userID = $1", id); err != nil {
		fmt.Printf("Problem when updating links in dissociateUser: %s", err)
	}
}

// Delete the user, but keep its ID on the links associated.
// The id doesn't refer to anyone, but it allows to make more precise statistics.
func ForgetUser(db *sql.DB, id int) {
	if _, err := db.Exec("delete from users where id = $1", id); err != nil {
		fmt.Printf("Problem when deleting user in forgetUser: %s", err)
	}
}

func DeleteUser(db *sql.DB, id int) {
	if _, err := db.Exec("delete from users where id = $1 ", id); err != nil {
		fmt.Println("Problem with deletion of the user ! " + err.Error())
	}
	if _, err := db.Exec("delete from link where userID = $1", id); err != nil {
		fmt.Printf("Problem with deletion of the links ! %s", err)
	}

}

func NewUserConnection(db *sql.DB) int {

	var id int
	if err := db.QueryRow(`insert into users (start_session, end_session) values ($1, NULL) returning id`, time.Now()).Scan(&id); err != nil {
		log.Fatal(err)
	}
	var nbrUsers int

	//In every case, we add a new time range
	if err := db.QueryRow("SELECT COUNT(*) FROM link WHERE link.endPlageID is null").Scan(&nbrUsers); err != nil {
		log.Fatal(err)
	}
	nbrUsers++
	var plageID int
	var t time.Time
	err := db.QueryRow("INSERT INTO plages (start, stop, nbr_users) VALUES ($1, null, $2) RETURNING id, start",
		time.Now().UTC(), nbrUsers).Scan(&plageID, &t)
	if err != nil {
		log.Fatal(err)
	}

	//We end the previous time range
	if _, err := db.Exec("UPDATE plages SET stop = $1 WHERE id = $2 - 1", t, plageID); err != nil {
		log.Fatal(err)
	}

	//We also add a new link
	if _, err := db.Exec("INSERT INTO link (userID, startPlageID, endPlageID) VALUES ($1, $2, null)", id, plageID); err != nil {
		log.Fatal(err)
	}

	return id

}

// Add a new connection from the user with ID id. If this user doesn't exists in the database, it is created.
// It takes care of updating the tables to ensure the database is coherent.
// If the user exists but is already logged into the server, this function does not do anything.
func UserConnection(db *sql.DB, id int) {

	var temp int
	var sessionOpen bool
	if err := db.QueryRow("SELECT count(*) from link where userID = $1 and endPlageID is null", id).Scan(&temp); err != nil {
		log.Fatal(err)
	}
	sessionOpen = temp > 0
	if sessionOpen {
		return
	}
	fmt.Printf("User %d connected at %s", id, time.Now().UTC().String())
	if _, err := db.Exec(`insert into users (id, start_session, end_session) values ($1, $2, NULL) ON CONFLICT (id) DO NOTHING`,
		id, time.Now().UTC()); err != nil {
		log.Fatal(err)
	}
	var nbrUsers int

	//In every case we add a new time range
	if err := db.QueryRow("SELECT COUNT(*) FROM link WHERE link.endPlageID is null").Scan(&nbrUsers); err != nil {
		log.Fatal(err)
	}
	nbrUsers++
	var plageID int
	var t time.Time
	err := db.QueryRow("INSERT INTO plages (start, stop, nbr_users) VALUES ($1, null, $2) RETURNING id, start",
		time.Now().UTC(), nbrUsers).Scan(&plageID, &t)
	if err != nil {
		log.Fatal(err)
	}

	//We end the previous time range
	if _, err := db.Exec("UPDATE plages SET stop = $1 WHERE id = $2 - 1", t, plageID); err != nil {
		log.Fatal(err)
	}

	//We also add a new link
	if _, err := db.Exec("INSERT INTO link (userID, startPlageID, endPlageID) VALUES ($1, $2, null)", id, plageID); err != nil {
		log.Fatal(err)
	}

}

// Disconnect a user specified by id. If the user wasn't connected in the first place, it does not do anything.
// It also update the database accordingly.
func UserDeconnection(db *sql.DB, id int) {

	var temp int
	var exists bool
	err := db.QueryRow("select count(*) from link where userID = $1 and endPlageID is null", id).Scan(&temp)
	if err != nil {
		fmt.Printf("The user doesn't exist, or there was an error in UserDeconnection : %s", err)
	}
	exists = temp > 0
	if exists {
		//We get the new current number of users logged in
		var nbrUsers int
		if err = db.QueryRow("SELECT COUNT(*) FROM link WHERE link.endPlageID is null").Scan(&nbrUsers); err != nil {
			log.Fatal(err)
		}
		nbrUsers--

		//We start a new time range
		var plageID int
		var t time.Time
		fmt.Printf("User %d disconnected at %s", id, time.Now().UTC().String())
		if err = db.QueryRow("insert into plages (start, stop, nbr_users) values ($1, null, $2) returning id, start",
			time.Now().UTC(), nbrUsers).Scan(&plageID, &t); err != nil {
			log.Fatal(err)
		}
		//We end the previous time range
		if _, err = db.Exec("update plages set stop = $1 where id = $2 - 1", t, plageID); err != nil {
			log.Fatal(err)
		}
		//We end the link of the user
		if _, err = db.Exec("update link set endPlageID = $1 where userID = $2 and endPlageID is null", plageID-1, id); err != nil {
			log.Fatal(err)
		}
		// This is to add some data to the users, not really useful
		if _, err = db.Exec("update users set end_session = $1 where id = $2", t, id); err != nil {
			log.Fatal(err)
		}
	}
}

// Insert fake data into the postgres database. It connects a random number of users, then disconnect some of them, and reconnect some.
// This means it also happen to connect/disconnect someone who is already connected/is not connecte, checking for errors and edge cases.
func PopulatePostgres(db *sql.DB) {
	fmt.Println("------------------ Starting to populate the database... please wait (can be quite long) ------------- ")

	r := 30000 - rand.Intn(15000)
	//We connect successively r users
	fmt.Printf("Connecting %d users to the server... ", r)

	for i := range r {
		if i == r/2 {
			fmt.Println("We are halfway ! Be strong !")
		}
		UserConnection(db, i)
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("Successfully connected the users. Now proceeding to try to deconnect %d users...", 4*r/5)
	//We disconnect some of them randomly (but not all)
	for i := range 4 * r / 5 {
		if i == 2*r/5 {
			fmt.Println("Half of the work is done : [##########          ]")
		}
		userDecoID := rand.Intn(r - 1)
		UserDeconnection(db, userDecoID)
		time.Sleep(10 * time.Millisecond)

	}

	fmt.Printf("Successfully disconnected the users. Now proceeding to try to reconnect %d users...", 4*r/5)
	//And then we reconnect some of them randomly

	for i := range 4 * r / 5 {
		if i == 2*r/5 {
			print("Almost finished !")
		}
		userRecoID := rand.Intn(r - 1)
		UserConnection(db, userRecoID)
		time.Sleep(10 * time.Millisecond)

	}

}
