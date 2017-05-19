package util

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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

func GenerateYMDDateStringWithSlash() string {
	// dateString := time.Now().Format("2006-01-02")
	dateString := time.Now().Format("2006/01/02")
	fmt.Println("date string: ", dateString)
	return dateString
}

func GenerateMD5HashCode(data string) string {
	if data == "" {
		panic("Generating hash code error.")
	}
	h := md5.New()
	io.WriteString(h, data)
	hashCode := h.Sum(nil)
	hashCodeString := fmt.Sprintf("%x", hashCode)
	return hashCodeString
}

type ByteSize float64

const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

func (b ByteSize) String() string {
	switch {
	case b >= YB:
		return fmt.Sprintf("%.2fYB", b/YB)
	case b >= ZB:
		return fmt.Sprintf("%.2fZB", b/ZB)
	case b >= EB:
		return fmt.Sprintf("%.2fEB", b/EB)
	case b >= PB:
		return fmt.Sprintf("%.2fPB", b/PB)
	case b >= TB:
		return fmt.Sprintf("%.2fTB", b/TB)
	case b >= GB:
		return fmt.Sprintf("%.2fGB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.2fMB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.2fKB", b/KB)
	}
	return fmt.Sprintf("%.2fB", b)
}

func GetFileSizeInString(size int64) string {
	return ByteSize(size).String()
}

// code => 1: postive, 0: normal, -1: negative
func GenerteReqResult(success bool, data interface{}, msg string, code int) map[string]interface{} {
	ret := map[string]interface{}{
		"success": success,
		"data":    data,
		"msg":     msg,
		"code":    code,
	}
	return ret
}
