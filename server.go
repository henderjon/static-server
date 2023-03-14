package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// isDotF reports whether name contains a path element starting with a period.
// The name is assumed to be a delimited by forward slashes, as guaranteed
// by the http.FileSystem interface.
func isDotF(name string) bool {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// noDotF is the http.File use in noDotFS.
// It is used to wrap the Readdir method of http.File so that we can
// remove files and directories that start with a period from its output.
type noDotF struct {
	http.File
}

// Readdir is a wrapper around the Readdir method of the embedded File
// that filters out all files that start with a period in their name.
func (f noDotF) Readdir(n int) (fis []os.FileInfo, err error) {
	files, err := f.File.Readdir(n)
	for _, file := range files { // Filters out the dot files
		if !strings.HasPrefix(file.Name(), ".") {
			fis = append(fis, file)
		}
	}
	return
}

// noDotFS is an http.FileSystem that hides
// hidden "dot files" from being served.
type noDotFS struct {
	http.FileSystem
}

// Open is a wrapper around the Open method of the embedded FileSystem
// that serves a 403 permission error when name has a file or directory
// with whose name starts with a period in its path.
func (fs noDotFS) Open(name string) (http.File, error) {
	if isDotF(name) { // If dot file, return 403 response
		return nil, os.ErrPermission
	}

	file, err := fs.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return noDotF{file}, err
}

func main() {
	dir := "."
	flag.Func("dir", "the dir to serve", func(s string) error {
		dir = s
		return nil
	})
	flag.Parse()

	fs := noDotFS{http.Dir(dir)}
	staticMux := http.NewServeMux()
	staticMux.Handle("/", http.FileServer(fs))
	staticMux.Handle("/post", http.HandlerFunc(redir))

	// create the server
	srv := &http.Server{
		Addr: `:8080`,
	}

	srv.Handler = staticMux
	fmt.Printf("serving \"%s\" on %s\n", dir, srv.Addr)
	log.Fatal(srv.ListenAndServe())

	// Simple static webserver:
	// dir, _ := os.Getwd()
	// log.Fatal(http.ListenAndServe(":8080", http.FileServer(http.Dir(dir))))

	// To serve a directory on disk (/tmp) under an alternate URL
	// path (/tmpfiles/), use StripPrefix to modify the request
	// URL's path before the FileServer sees it:
	// http.Handle("/tmpfiles/", http.StripPrefix("/tmpfiles/", http.FileServer(http.Dir("/tmp"))))
}

func redir(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Println(r.Form)
	http.Redirect(w, r, "https://httpbin.org/get", http.StatusSeeOther)
}
