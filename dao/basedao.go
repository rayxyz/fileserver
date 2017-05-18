package dao

import (
	"container/list"
	"database/sql"
	"fmt"
	"log"
)

func getDB() *sql.DB {
	db, err := sql.Open("mysql", "root:rayxyz123@/fileserverdb")
	if err != nil {
		log.Fatal("Generating QR code error.", err)
	}
	return db
}

func Query(queryString string, params ...interface{}) *list.List {
	db := getDB()
	rows, err := db.Query(queryString, params...)
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer db.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Result container
	retlist := list.New()

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		rowmap := make(map[string]interface{})

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			fmt.Println(columns[i], ": ", value)
			rowmap[columns[i]] = value
		}
		// If rowmap is not nil, then put it into the list.
		if rowmap != nil {
			retlist.PushFront(rowmap)
		}
	}
	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	defer rows.Close()

	return retlist
}
