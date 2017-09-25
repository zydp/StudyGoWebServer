package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"strings"
)

var templatesA = make(map[string]*template.Template)

func init() {
	fmt.Printf("OS:%s SystemArch:%s\n", strings.Title(runtime.GOOS), strings.ToUpper(runtime.GOARCH))
	fileInfoArr, err := ioutil.ReadDir(TEMPLATE_DIR)
	if err != nil {
		panic(err)
		return
	}
	var templateName, templatePath string
	for _, fileInfo := range fileInfoArr {
		templateName = fileInfo.Name()
		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}
		templatePath = TEMPLATE_DIR + "/" + templateName
		log.Println("Loading template:", templatePath)
		t := template.Must(template.ParseFiles(templatePath))
		templatesA[templateName] = t
	}
}

const (
	UPLOAD_DIR   = "source/uploads"
	TEMPLATE_DIR = "source/views"
)

func main() {
	HttpServer()
}

func HttpServer() {
	http.HandleFunc("/", safeHandler(listHandler))
	http.HandleFunc("/view", safeHandler(viewHandler))
	http.HandleFunc("/upload", safeHandler(uploadHandler))
	err := http.ListenAndServe(":3244", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		err := renderHtml(w, "upload.html", nil)
		check(err)
		return
	}
	if r.Method == "POST" {
		f, h, err := r.FormFile("image")
		check(err)
		filename := h.Filename
		defer f.Close()
		t, err := os.Create(UPLOAD_DIR + "/" + filename)
		check(err)
		defer t.Close()
		_, err = io.Copy(t, f)
		check(err)
		http.Redirect(w, r, "/view?id="+filename, http.StatusFound)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	if exists := isExists(imagePath); !exists {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image")
	http.ServeFile(w, r, imagePath)
}

func isExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	fileInfoArr, err := ioutil.ReadDir(UPLOAD_DIR)
	check(err)
	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images
	err = renderHtml(w, "list.html", locals)
	check(err)
}

func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) (err error) {
	return templatesA[tmpl].Execute(w, locals)
}

func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err, ok := recover().(error)
			if ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				// 或者输出自定义的 50x 错误页面
				// w.WriteHeader(http.StatusInternalServerError)
				// renderHtml(w, "error", e)

				// logging
				log.Println("WARN: panic in %v - %v", fn, err)
				log.Println(string(debug.Stack()))
			}
		}()
		fn(w, r)
	}
}
