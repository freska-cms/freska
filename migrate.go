package main

import (
	"fmt"
	"github.com/freska/query"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrate runs all pending migrations
func RunMigrate(args []string) {

	// Remove freska backup from args list
	args = args[2:]

	switch freskaConfig(args) {
	case ModeProduction:
		migrateDB(ConfigProduction)
	case ModeTest:
		migrateDB(ConfigTest)
	default:
		migrateDB(ConfigDevelopment)
	}

}

// migrateDB finds the last run migration, and run all those after it in order
// We use the freska_metadata table to do this
func migrateDB(config map[string]string) {
	var migrations []string
	var completed []string

	// Get a list of migration files
	files, err := filepath.Glob("./db/migrate/*.sql")
	if err != nil {
		log.Printf("Error running restore %s", err)
		return
	}

	// Sort the list alphabetically
	sort.Strings(files)

	// Try opening the db (db may not exist at this stage)
	err = openDatabase(config)
	if err != nil {
		// if no db, proceed with empty migrations list
		log.Printf("No database found")
	} else {
		migrations = readMetadata()
	}

	for _, file := range files {
		filename := filepath.Base(file)

		if !contains(filename, migrations) {
			log.Printf("Running migration %s", filename)

			args := []string{"-d", config["db"], "-f", file}
			if strings.Contains(filename, createDatabaseMigrationName) {
				args = []string{"-f", file}
				log.Printf("Running database creation migration: %s", file)
			}

			// Execute this sql file against the database
			result, err := runCommand("psql", args...)
			if err != nil || strings.Contains(string(result), "ERROR") {
				if err == nil {
					err = fmt.Errorf("\n%s", string(result))
				}

				// If at any point we fail, log it and break
				log.Printf("ERROR loading sql migration:%s\n", err)
				log.Printf("All further migrations cancelled\n\n")
				break
			}

			completed = append(completed, filename)
			log.Printf("Completed migration %s\n%s\n%s", filename, string(result), freskaDivider)
		}
	}

	if len(completed) > 0 {
		writeMetadata(config, completed)
		log.Printf("Migrations complete up to migration %s on db %s\n\n", completed[len(completed)-1], config["db"])
	} else {
		log.Printf("No migrations to perform at path %s\n\n", "./db/migrate")
	}

}

// openDatabase opens the database specified in the config map
func openDatabase(config map[string]string) error {
	// Open the database
	options := map[string]string{
		"adapter":  config["db_adapter"],
		"user":     config["db_user"],
		"password": config["db_pass"],
		"db":       config["db"],
		// "debug"     : "true",
	}

	err := query.OpenDatabase(options)
	if err != nil {
		return err
	}

	log.Printf("%s\n", freskaDivider)
	log.Printf("Opened database at %s for user %s", config["db"], config["db_user"])
	return nil
}

// readMetadata reads the metadata from the freska_metadata table
func readMetadata() []string {
	var migrations []string

	sql := "select migration_version from freska_metadata order by id desc;"

	rows, err := query.QuerySQL(sql)
	if err != nil {
		log.Printf("Database ERROR %s", err)
		return migrations
	}

	// We expect just one row, with one column (count)
	defer rows.Close()
	for rows.Next() {
		var migration string
		err := rows.Scan(&migration)
		if err != nil {
			log.Printf("Database ERROR %s", err)
			return migrations
		}
		migrations = append(migrations, migration)

	}

	return migrations
}

// writeMetadata writes a new row in the freska_metadata table to record our action
func writeMetadata(config map[string]string, migrations []string) {

	for _, m := range migrations {
		sql := "Insert into freska_metadata(updated_at,freska_version,migration_version,status) VALUES(NOW(),$1,$2,100);"
		result, err := query.ExecSQL(sql, freskaVersion, m)
		if err != nil {
			log.Printf("Database ERROR %s %s", err, result)
		}
	}

}

// contains checks whether an array of strings contains a string
func contains(s string, a []string) bool {
	for _, k := range a {
		if s == k {
			return true
		}
	}
	return false
}
