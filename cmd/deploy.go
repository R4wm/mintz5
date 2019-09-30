package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/r4wm/mintz5/db"
	"github.com/r4wm/mintz5/kjv"
	"github.com/r4wm/sqlite3_kjv"
	log "github.com/sirupsen/logrus"
)

func main() {

	dbPathp := flag.String("dbPath", "/tmp/kjv.db", "Path to kjv database.")
	createDB := flag.Bool("createDB", false, "Create the kjv database.")
	flag.Parse()

	// Create the DB if asked
	if *createDB == true {
		path, err := os.Stat(*dbPathp)
		if os.IsNotExist(err) {
			_, err := sqlite3_kjv.CreateKJVDB(*dbPathp)

			if err != nil {
				panic(err)
			}

			log.Infof("Created database %v", path)
		}
	}

	// Check the db path exists
	_, err := os.Stat(*dbPathp)
	if os.IsNotExist(err) {
		log.Errorf("database path does not exist: %s", *dbPathp)
		os.Exit(1)
	}

	// Create database connection
	db, err := db.CreateDatabase(*dbPathp)
	if err != nil {
		panic(err)
	}

	log.Infof("Database connection OK.")

	// Router
	router := mux.NewRouter().StrictSlash(false)

	app := kjv.App{
		Router:   router,
		Database: db,
	}

	app.SetupRouter()
	port := ":8000"
	log.Infof("Listening on %s\n", port)

	// Serve
	log.Fatal(http.ListenAndServe(port, router))
}
