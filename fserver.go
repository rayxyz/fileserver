package main

import (
	"encoding/json"
	"fileserver/dao"
	"fileserver/util"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// The max memory when upload a file
var maxUploadMemory int64 = 1024 * 1024 * 1

// The max file size to be permitted.
var maxFileSize int64 = 1024 * 1024 * 5

// The working directory
var homedir string = os.Getenv("HOME")

// The folder to store upload files in homedir.
var fileSotreDir string = "file"

// Folder to store files uploaded
var fileStorePath string = path.Join(homedir, "file")

// Path of static resources of file server.
var staticResourcePath = path.Join(homedir, "fileserver-static")

// View path
var viewpath string = path.Join(homedir, "fileserver-static/view")

type FileModel struct {
	Id         string    `json:"id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	SizeStr    string    `json:"sizeStr"`
	UploadTime time.Time `json:"uploadTime"`
}

func get(w http.ResponseWriter, r *http.Request) {
	// fileId := r.FormValue("fid")
	vals := mux.Vars(r)
	retList := dao.Query("select t.* from t_file t where id = ?", vals["id"])
	var filePath string
	for i := 0; i < retList.Len(); i++ {
		if i == 0 {
			filePath = path.Join(homedir, retList.Front().Value.(map[string]interface{})["path"].(string))
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
	fmt.Println("file path: ", filePath)
	http.ServeFile(w, r, filePath)
}

func upload(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Access-Control-Allow-Origin", "*")
	// w.Header().Set("Access-Control-Allow-Methods", "*")
	// w.Header().Set("Access-Control-Allow-Headers", "*")
	// w.Header().Set("Access-Control-Expose-Headers", "*")
	// w.Header().Set("Access-Control-Allow-Credentials", "true")
	r.ParseMultipartForm(maxUploadMemory)
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println("Upload file error => ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	fileName := header.Filename
	fileLen := r.ContentLength
	if maxFileSize < fileLen {
		http.Error(w, "File size is too large.", http.StatusInternalServerError)
		return
	}
	// rw-rw-rw- for file created
	filePathWithoutRoot, filePath := generateFilePath(fileName)
	newFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Upload file error => ", err.Error())
		panic("Upload file error")
	}
	io.Copy(newFile, file)
	defer newFile.Close()
	// fmt.Fprint(w, "File upload succeed!")
	preparedSQL := "insert into t_file(id, name, path, size, upload_time) values(uuid_short(), ?, ?, ?, now())"
	fid := save(preparedSQL, fileName, filePathWithoutRoot, fileLen)
	// BUild file basic info object.
	fd := FileModel{
		strconv.FormatInt(fid, 10),
		fileName,
		fileLen,
		util.GetFileSizeInString(fileLen),
		time.Now(),
	}
	ret := util.GenerteReqResult(true, fd, "Upload file complete.", 0)
	json.NewEncoder(w).Encode(ret)
}

func uploadWithProgress(w http.ResponseWriter, r *http.Request) {
	fileLen := r.ContentLength
	if maxFileSize < fileLen {
		http.Error(w, "File size is too large.", http.StatusInternalServerError)
		return
	}
	mr, err := r.MultipartReader()
	if err != nil {
		fmt.Fprint(w, "Upload file error of parse multi part.")
		panic(err.Error())
	}
	part, err := mr.NextPart()
	if err != nil && err != io.EOF {
		panic("Upload file error.")
	}
	fileName := part.FileName()
	filePathWithoutRoot, filePath := generateFilePath(fileName)
	fmt.Println("file path in upload with progress: ", filePath, ", without root: ", filePathWithoutRoot)
	for {
		if err == io.EOF {
			break
		}
		var lengthOfRead int64
		var percentage float32
		dst, erro := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
		if erro != nil {
			fmt.Fprint(w, "Upload file error.")
			panic(err.Error())
		}
		defer dst.Close()
		for {
			buffer := make([]byte, 1024)
			countbytes, erro := part.Read(buffer)
			if erro == io.EOF {
				break
			}
			lengthOfRead += int64(countbytes)
			percentage = float32(lengthOfRead) / float32(fileLen) * 100
			fmt.Printf("File upload progress: %v%s\n", percentage, "%")
			// fmt.Fprintf(w, "progress: %v\n", percentage)
			dst.Write(buffer[0:countbytes])
		}
		part, err = mr.NextPart()
	}
	preparedSQL := "insert into t_file(id, name, path, size, upload_time) values(uuid_short(), ?, ?, ?, now())"
	fid := save(preparedSQL, fileName, filePathWithoutRoot, fileLen)
	// BUild file basic info object.
	fd := FileModel{
		fmt.Sprintf("%d", fid),
		fileName,
		fileLen,
		util.GetFileSizeInString(fileLen),
		time.Now(),
	}
	ret := util.GenerteReqResult(true, fd, "Upload file complete.", 0)
	json.NewEncoder(w).Encode(ret)
}

func multiUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	err := r.ParseMultipartForm(maxUploadMemory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mpf := r.MultipartForm
	files := mpf.File["files"]
	var fileInfos []FileModel
	for key, _ := range files {
		file, err := files[key].Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		filePathWithoutRoot, filePath := generateFilePath(files[key].Filename)
		newFile, err := os.Create(filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fileLen, _ := io.Copy(newFile, file)
		if maxFileSize < fileLen {
			http.Error(w, "File size is too large.", http.StatusInternalServerError)
			return
		}
		preparedSQL := "insert into t_file(id, name, path, size, upload_time) values(uuid_short(), ?, ?, ?, now())"
		fid := save(preparedSQL, files[key].Filename, filePathWithoutRoot, fileLen)
		fmt.Println("fid: ", fid)
		// Build file basic info object.
		fd := FileModel{
			strconv.FormatInt(fid, 10),
			files[key].Filename,
			fileLen,
			util.GetFileSizeInString(fileLen),
			time.Now(),
		}
		fileInfos = append(fileInfos, fd)
		if err != nil {
			panic(err.Error())
		}
		defer newFile.Close()
	}
	fmt.Println()
	ret := util.GenerteReqResult(true, fileInfos, "Upload files complete.", 0)
	json.NewEncoder(w).Encode(ret)
}

// Save file info to database.
func save(preparedSQL string, params ...interface{}) (fid int64) {
	return dao.Insert(preparedSQL, params...)
}

func generateFilePath(fileName string) (filePathWithoutRoot string, filePath string) {
	fileRelativePath := util.GenerateYMDDateStringWithSlash()
	timeNow := time.Now().Format("2006-01-02 15:04:05")
	fmt.Println("time now: ", timeNow)
	fileNameHash := util.GenerateMD5HashCode(fileName + timeNow)
	fmt.Println("file name hash: ", fileNameHash)
	fileRelativePath += "/" + fileNameHash[0:2]
	filePathWithoutRoot = path.Join(fileSotreDir, fileRelativePath)
	fileRelativePath = path.Join(fileStorePath, fileRelativePath)
	fmt.Println("file relative path before concat: ", fileRelativePath)
	if os.MkdirAll(fileRelativePath, 0777) != nil {
		panic("Making direcitories error.")
	}
	fmt.Println("After making directories...")
	fileRelativePath += "/" + string(fileNameHash[2:len(fileNameHash)])
	filePathWithoutRoot += "/" + string(fileNameHash[2:len(fileNameHash)])
	fileNameParts := strings.Split(fileName, ".")
	filePath = fileRelativePath + "." + fileNameParts[len(fileNameParts)-1]
	filePathWithoutRoot += "." + fileNameParts[len(fileNameParts)-1]
	fmt.Println("new file path: ", filePath)
	return filePathWithoutRoot, filePath
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
		filePath = path.Join(homedir, fileObj["path"].(string))
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
	// http.Handle("/getQRCstring(fid)ode", http.HandlerFunc(generateQRCode))
	// http.Handle("/file", http.HandlerFunc(getFile))

	router := mux.NewRouter().StrictSlash(true)
	// Match static resources begin
	fs := http.FileServer(http.Dir(staticResourcePath))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	// Match static resources end.
	router.HandleFunc("/get/{id}", get)
	router.HandleFunc("/view/upload", getUploadPage)
	router.HandleFunc("/upload", upload)
	router.HandleFunc("/uploadWithProgress", uploadWithProgress)
	router.HandleFunc("/multiUpload", multiUpload)
	router.HandleFunc("/download/{id}", download)
	// Handlers.CORS makes cross domain resources sharing possible.
	error := http.ListenAndServe(":8090", handlers.CORS()(router))
	if error != nil {
		panic(error.Error())
	}
}
