package main

import (
	"encoding/json"
	"fileserver/dao"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// The max memory when upload a file
var maxUploadMemory int64 = 1024 * 5

// The working directory
var homedir string = os.Getenv("HOME")

// Folder to store files uploaded
var fileStorePath string = path.Join("file")

// View path
var viewpath string = path.Join("fileserver-static/view")

type Todo struct {
	Name      string
	Completed bool
	Due       time.Time
}

func welcome(w http.ResponseWriter, r *http.Request) {
	todos := []Todo{
		{"Raywang", true, time.Now()},
		{"Xiaoming", true, time.Now()},
		{"Hanmei", true, time.Now()},
		{"Xiaoying", true, time.Now()},
	}
	json.NewEncoder(w).Encode(todos)
}

func get(w http.ResponseWriter, r *http.Request) {
	// fileId := r.FormValue("fid")
	vals := mux.Vars(r)
	retList := dao.Query("select t.* from t_file t where id = ?", vals["id"])
	var filePath string
	for i := 0; i < retList.Len(); i++ {
		if i == 0 {
			// filePath = homedir + "/" + retList.Front().Value.(map[string]interface{})["path"].(string)
			// Concatenate to a string
			// var buffer bytes.Buffer
			// buffer.WriteString(filePath)
			// buffer.WriteString(homedir)
			// buffer.WriteString("/")
			// buffer.WriteString(retList.Front().Value.(map[string]interface{})["path"].(string))
			// filePath = buffer.String()
			filePath = path.Join(retList.Front().Value.(map[string]interface{})["path"].(string))
			log.Println("file path: ", filePath)
			break
		}
	}
	if filePath == "" {
		panic("The file path is blank.")
	}
	// read the whole file at once
	content, err := ioutil.ReadFile(filePath)
	// fmt.Println(content)
	if err != nil {
		log.Println("Read file ", filePath, " error.")
		panic(err)
	}
	w.Write(content)
}

func getUploadPage(w http.ResponseWriter, r *http.Request) {
	filePath := path.Join(viewpath, "upload.html")
	http.ServeFile(w, r, filePath)
}

func upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(maxUploadMemory)
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println("Upload file error.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	newFilePath := path.Join(fileStorePath, header.Filename)
	newFile, err := os.OpenFile(newFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Fprint(w, "Upload file error => "+err.Error())
		return
	}
	defer newFile.Close()
	io.Copy(newFile, file)
	fmt.Fprint(w, "File upload succeed!")
}

func download(w http.ResponseWriter, r *http.Request) {
	vals := mux.Vars(r)
	if vals == nil || vals["id"] == "" {
		fmt.Fprintln(w, "File download error. Parameter abscent.")
		return
	}
	retList := dao.Query("select t.* from t_file t where id = ?", vals["id"])
	var filePath string
	var fileObj map[string]interface{}
	if retList.Len() > 0 {
		fileObj = retList.Front().Value.(map[string]interface{})
		filePath = path.Join(fileObj["path"].(string))
		log.Println("file path: ", filePath)
	} else {
		fmt.Fprintln(w, "File download error.")
		return
	}
	if filePath == "" {
		panic("The file path is blank.")
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileObj["name"].(string))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", r.Header.Get("Content-Length"))

	reader, _ := os.Open(filePath)
	io.Copy(w, reader)
}

func main() {
	// http.Handle("/getQRCode", http.HandlerFunc(generateQRCode))
	// http.Handle("/file", http.HandlerFunc(getFile))

	router := mux.NewRouter().StrictSlash(true)
	// fs := http.FileServer(http.Dir("fileserver-static"))
	// router.Handle("/static/", http.StripPrefix("/static/", fs))
	router.HandleFunc("/get/{id}", get)
	router.HandleFunc("/view/upload", getUploadPage)
	router.HandleFunc("/upload", upload)
	router.HandleFunc("/download/{id}", download)
	error := http.ListenAndServe(":8090", router)
	if error != nil {
		panic(error.Error())
	}
}
