package cache

import (
	"database/sql"
	"fmt"
	"os"

	c "github.com/gookit/color"
)

type Cache struct {
	Path string
	DB   *sql.DB
}

func Open(path string) (*Cache, error) {

	// exists?
	if _, err := os.Stat(path); err == nil {
		c.Printf("Opening <magenta>%s</>...\n", path)
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			return nil, fmt.Errorf("failed to open db %s: %w", path, err)
		}

		return &Cache{path, db}, nil
	}

	// create file
	c.Printf("Creating <magenta>%s</>...\n", path)
	os.Create(path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open db %s: %w", path, err)
	}

	// id, title, user, email, state, milestone, merged, merger, merger_email, created, closed
	c.Printf("  table <white>prs</>...\n")
	_, err = db.Exec(`
	CREATE TABLE "prs" (
	    "number" INTEGER PRIMARY KEY, 
	    "title" VARCHAR(256) NOT NULL, 
	    "user" CHAR(64) NOT NULL, 
	    "state" CHAR(32) NOT NULL,
	    "milestone" CHAR(32) NULL,
	    "merged" INTEGER,
	    "merger" CHAR(32),
	    "created" DATE NOT NULL,
	    "closed" DATE,
	    "daysopen" REAL,
	    "dayswaiting" REAL,
	    "daystofirst" REAL
	)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR table %s: %w", path, err)
	}

	// id, title, user, email, state, milestone, merged, merger, merger_email, created, closed
	c.Printf("  table <white>events</>...\n")
	_, err = db.Exec(`
	CREATE TABLE "events" (
	    "pr" INTEGER, 
	    "date" DATE NOT NULL,
	    "event" CHAR(32) NOT NULL,
	    "user" CHAR(64) NOT NULL, 
	    
	    "state" CHAR(32),
	    "label" CHAR(64),
	    "milestone" CHAR(64),
	    "body" VARCHAR,
	    
	    "url" CHAR(128) NOT NULL,
	    PRIMARY KEY (pr, date)
	)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create events table %s: %w", path, err)
	}

	return &Cache{path, db}, nil
}
