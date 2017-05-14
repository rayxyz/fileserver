package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"
)

func GenerateQRCode(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Handling a action...")
	var png []byte
	png, err := qrcode.Encode("http://localhost:8080/wordgen", qrcode.Medium, 256)
	if err != nil {
		log.Fatal("Generating QR code error.", err)
	}
	w.Write(png)
}

func GenerateQRCodeFromDB(w http.ResponseWriter, req *http.Request) {
	db, err := sql.Open("mysql", "root:root@/wordgen")
	if err != nil {
		log.Fatal("Generating QR code error.", err)
	}
	defer db.Close()
	rows, error := db.Exec("select t.* from t_file t")
	if error != nil {
		fmt.Println("Error.")
	}
	fmt.Println(rows)
}
