package main

// +build version_embedded

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	trace = debugT(false)

	TWBSversion               = "3.0.1"
	defaultUserID             = "sample.user"
	majorVersion              = "1.1dev"
	maxGoroutine              = 100
	permDir       os.FileMode = 0755
	permFile      os.FileMode = 0644
)

type debugT bool

func (d debugT) Println(args ...interface{}) {
	if d {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			if line > maxLineNumber {
				maxLineNumber = line
				maxLineDigits = len(fmt.Sprint(maxLineNumber))
			}
			args = append([]interface{}{
				file,
				fmt.Sprintf(fmt.Sprintf("%%%dd:", maxLineDigits), line),
			}, args...)
		}
		log.Println(args...)
	}
}

func (d debugT) Do(f func()) {
	if d {
		f()
	}
}

var (
	TWBSfilename       = fmt.Sprintf("bootstrap-%v.min.css", TWBSversion)
	TWBSurl            = fmt.Sprintf("https://github.com/twbs/bootstrap/raw/v%v/dist/css/bootstrap.min.css", TWBSversion)
	buildAt            string
	debug              bool
	develop            debugT
	distDir            string
	maxLineDigits      int
	maxLineNumber      = 0
	maxProcesses       = runtime.NumCPU()
	memStats           runtime.MemStats
	semaphoreFile      chan int
	semaphoreFileCount = maxProcesses * 2 * 2
	semaphoreHTTP      chan int
	semaphoreHTTPCount = maxProcesses * 2 * 2
	userID             string
	version            string
	waitWG             bool
	wg                 sync.WaitGroup
)

func init() {
	runtime.GOMAXPROCS(maxProcesses)

	if semaphoreFileCount > 8 {
		semaphoreFileCount = 8
	}
	if semaphoreHTTPCount > 8 {
		semaphoreHTTPCount = 8
	}
	semaphoreFile = make(chan int, semaphoreFileCount)
	semaphoreHTTP = make(chan int, semaphoreHTTPCount)

	flag.BoolVar(&debug, "v", false, "print debug messages")
	flag.StringVar(&distDir, "d", "", "destination directory")
	flag.StringVar(&userID, "u", defaultUserID, "user ID")
	flag.Parse()

	develop = debugT(debug)
}

func AddWaitGroup(f func()) {
	trace.Println()
	if waitWG == true {
		trace.Println()
		waitWG = false
		for {
			numGoroutine := runtime.NumGoroutine()
			if numGoroutine < maxGoroutine {
				break
			}
			develop.Println(
				"NumGoroutine:", numGoroutine,
				"semaphoreFile:", len(semaphoreFile),
				"semaphoreHTTP:", len(semaphoreHTTP),
			)
			trace.Println()
			runtime.Gosched()
			trace.Println()
		}
	} else {
		trace.Println()
		if rand.Intn(10) == 0 {
			numGoroutine := runtime.NumGoroutine()
			if numGoroutine > maxGoroutine {
				runtime.ReadMemStats(&memStats)
				develop.Println(
					"Alloc:", memStats.Alloc,
					"NumGC:", memStats.NumGC,
					"NumGoroutine:", numGoroutine,
					"semaphoreFile:", len(semaphoreFile),
					"semaphoreHTTP:", len(semaphoreHTTP),
				)
				waitWG = true
			}
		}
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}

const indexhtml = `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="refresh" content="0; URL=albums/index.html">
</head>`

/* haml -f html5 -t ugly
!!! 5
%html/
%head
 %link(href="../bootstrap-%v.min.css" rel="stylesheet")
%body/
.row %v
*/
const html = `<!DOCTYPE html>
<html>
<head>
<link href='../bootstrap-%v.min.css' rel='stylesheet'>
</head>
<body>
<div class='row'>%v</div>
`

/* haml -f html5 -t ugly
{{range .Entry}}
.col-sm-4.col-md-2
 %a(href="{{.GphotoId}}.html" style="color: #000; text-decoration: none;")
  .thumbnail(style="width: 197px; margin: 3px 0 0 3px;")
   %img(src="img/index/{{.GphotoId}}.jpg")
   .caption
    %h6(style="margin-top: 0px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;")
     {{.Title}}
    %p.muted(style="font-size: 11px; margin-top: -10px; margin-bottom: -5px;")
     写真{{.Numphotos}}枚
{{end}}
*/
const li_album = `
{{range .Entry}}
<div class='col-sm-4 col-md-2'>
<a href='{{.GphotoId}}.html' style='color: #000; text-decoration: none;'>
<div class='thumbnail' style='width: 197px; margin: 3px 0 0 3px;'>
<img src='img/index/{{.GphotoId}}.jpg'>
<div class='caption'>
<h6 style='margin-top: 0px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;'>
{{.Title}}
</h6>
<p class='muted' style='font-size: 11px; margin-top: -10px; margin-bottom: -5px;'>
写真{{.Numphotos}}枚
</p>
</div>
</div>
</a>
</div>
{{end}}
`

/* haml -f html5 -t ugly
{{$GphotoId := .GphotoId}}
{{range .Photo}}
.col-sm-4.col-md-2
 .thumbnail(style="width: 197px; margin: 3px 0 0 3px;")
  %img(src="img/{{$GphotoId}}/{{.Content.Name}}")
  %h6(style="margin-top: 0px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;")
   {{.Title}}
   %span.muted {{timeFormat .TimestampTime "2006-01-02T15:04:05"}}
  %p.muted(style="font-size: 11px; margin-top: -10px; margin-bottom: -5px;")
   %a(href="{{.Content.MediaUrlBase}}s2048/{{.Content.Name}}") Max
   %a(href="{{.Content.MediaUrlBase}}s640/{{.Content.Name}}") s640
   %a(href="{{.Content.MediaUrlBase}}w236/{{.Content.Name}}") w236
   %a(href="{{.Content.MediaUrlBase}}h196/{{.Content.Name}}") h196
{{end}}
*/
const li_photo = `
{{$GphotoId := .GphotoId}}
{{range .Photo}}
<div class='col-sm-4 col-md-2'>
<div class='thumbnail' style='width: 197px; margin: 3px 0 0 3px;'>
<img src='img/{{$GphotoId}}/{{.Content.Name}}'>
<h6 style='margin-top: 0px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;'>
{{.Title}}
<span class='muted'>{{timeFormat .TimestampTime "2006-01-02T15:04:05"}}</span>
</h6>
<p class='muted' style='font-size: 11px; margin-top: -10px; margin-bottom: -5px;'>
<a href='{{.Content.MediaUrlBase}}s2048/{{.Content.Name}}'>Max</a>
<a href='{{.Content.MediaUrlBase}}s640/{{.Content.Name}}'>s640</a>
<a href='{{.Content.MediaUrlBase}}w236/{{.Content.Name}}'>w236</a>
<a href='{{.Content.MediaUrlBase}}h196/{{.Content.Name}}'>h196</a>
</p>
</div>
</div>
{{end}}
`

type Albums struct {
	//Updated string  `xml:"updated"`
	Entry []Entry `xml:"entry"`
}

type Entry struct {
	Updated   string `xml:"updated"`
	Title     string `xml:"title"`
	GphotoId  string `xml:"id"`
	LinkList  []Link `xml:"link"`
	Link      string
	Numphotos int       `xml:"numphotos"`
	Timestamp int64     `xml:"timestamp"`
	Thumbnail Thumbnail `xml:"group>content"`
}

func (e *Entry) SetLink() {
	for _, link := range e.LinkList {
		if link.Rel == "http://schemas.google.com/g/2005#feed" {
			e.Link = link.Href
			e.LinkList = nil
			return
		}
	}
}

type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Thumbnail struct {
	Url          string `xml:"url,attr"`
	MediaUrlBase string
}

func (t *Thumbnail) SetMediaUrlBase() {
	if t.MediaUrlBase == "" {
		t.MediaUrlBase = t.Url[:strings.LastIndex(t.Url, "/")]
	}
}

type Album struct {
	//Updated  string  `xml:"updated"`
	GphotoId string  `xml:"id"`
	Photo    []Photo `xml:"entry"`
}

type Photo struct {
	Updated       string  `xml:"updated"`
	Title         string  `xml:"title"`
	Content       Content `xml:"content"`
	Timestamp     int64   `xml:"timestamp"`
	TimestampTime time.Time
}

type Content struct {
	Src          string `xml:"src,attr"`
	Name         string
	MediaUrlBase string
}

func (c *Content) SetName() {
	if c.Name == "" {
		bits := strings.Split(c.Src, "/")
		lenBits := len(bits)
		c.Name = bits[lenBits-2] + "." + strings.Split(bits[lenBits-1], ".")[1]
	}
	return
}

func (c *Content) SetMediaUrlBase() {
	if c.MediaUrlBase == "" {
		c.MediaUrlBase = c.Src[:strings.LastIndex(c.Src, "/")+1]
	}
	return
}

func writeIndex(albums *Albums) error {
	t := template.Must(template.New("html").Parse(fmt.Sprintf(html, TWBSversion, li_album)))
	filename := "albums/index.html"
	f, closer, err := OpenFile(filename)
	defer func() {
		closer <- 0
	}()
	if err != nil {
		return err
	}
	trace.Println()
	err = t.Execute(f, albums)
	trace.Println()
	if err1 := f.Close(); err == nil {
		develop.Println(err)
		err = err1
	}
	develop.Println("writeIndex")
	return err
}

func writeAlbum(album *Album) error {
	trace.Println()
	funcMap := template.FuncMap{
		"timeFormat": func(t time.Time, f string) string {
			return t.Format(f)
		},
	}

	for i := range album.Photo {
		trace.Println(i)
		album.Photo[i].Content.SetName()
		album.Photo[i].Content.SetMediaUrlBase()
		album.Photo[i].TimestampTime = time.Unix(album.Photo[i].Timestamp/1000, 0)
		dirname := "albums/img/" + album.GphotoId
		err := os.MkdirAll(dirname, permDir)
		if err != nil {
			develop.Println(err)
			log.Print(err)
			continue
		}
		url := album.Photo[i].Content.MediaUrlBase + "w197-h134-p/"
		filename := dirname + "/" + album.Photo[i].Content.Name
		updated := album.Photo[i].Updated
		trace.Println(i)
		AddWaitGroup(func() {
			writeImage(url, filename, updated)
		})
	}
	t := template.Must(template.New("html").Funcs(funcMap).Parse(fmt.Sprintf(html, TWBSversion, li_photo)))
	filename := "albums/" + album.GphotoId + ".html"
	f, closer, err := OpenFile(filename)
	defer func() {
		closer <- 0
	}()
	if err != nil {
		return err
	}
	trace.Println()
	err = t.Execute(f, album)
	trace.Println()
	if err1 := f.Close(); err == nil {
		err = err1
	}
	develop.Println("writeAlbum: ", album.GphotoId, runtime.NumGoroutine())
	return err
}

func writeImage(url string, filename string, updated string) (err error) {
	fi, err := os.Stat(filename)
	if err == nil {
		if fi.Size() > 0 {
			t, _ := time.Parse("2006-01-02T15:04:05.000Z", updated)
			if fi.ModTime().Sub(t) > 0 {
				return
			}
		}
	}

	fmt.Println("Fetching", url)
	defer fmt.Println("Fetched", url)

	trace.Println()
	semaphoreHTTP <- 0
	trace.Println()
	defer func() {
		<-semaphoreHTTP
	}()
	trace.Println()
	resp, err := http.Get(url)
	trace.Println()
	if err != nil {
		develop.Println(err)
		log.Print(err)
		return
	}
	defer resp.Body.Close()

	f, closer, err := OpenFile(filename)
	defer func() {
		closer <- 0
	}()
	if err != nil {
		develop.Println(err)
		log.Print(err)
		return
	}

	trace.Println()
	written, err := io.Copy(f, resp.Body)
	trace.Println()
	if err1 := f.Close(); err == nil {
		err = err1
		if err != nil {
			develop.Println(err)
			log.Print(err)
		}
	}
	develop.Println(filename, written)
	return
}

func OpenFile(filename string) (file *os.File, closer chan int, err error) {
	trace.Println()
	semaphoreFile <- 0
	trace.Println()
	closer = make(chan int)
	go func() {
		<-closer
		close(closer)
		<-semaphoreFile
	}()
	file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
	if err != nil {
		develop.Println(err)
	}
	return
}

func HTTPGET(url string) (body []byte, err error) {
	fmt.Println("Fetching", url)
	defer fmt.Println("Fetched", url)

	trace.Println()
	semaphoreHTTP <- 0
	trace.Println()
	defer func() {
		<-semaphoreHTTP
	}()

	trace.Println()
	resp, err := http.Get(url)
	trace.Println()
	if err != nil {
		develop.Println(err)
		return
	}
	trace.Println()
	body, err = ioutil.ReadAll(resp.Body)
	trace.Println()
	if err != nil {
		develop.Println(err)
		return
	}
	defer resp.Body.Close()
	trace.Println()
	return
}

func FeedGet(userID string) (body []byte, err error) {
	body, err = HTTPGET("https://picasaweb.google.com/data/feed/api/user/" + userID)
	return
}

func getAlbums(userID string) (albums Albums) {
	body, err := FeedGet(userID)
	if err != nil {
		log.Print(err)
		return
	}
	develop.Println("Got album feed")

	xml.Unmarshal(body, &albums)
	for i := range albums.Entry {
		albums.Entry[i].SetLink()
		albums.Entry[i].Thumbnail.SetMediaUrlBase()
		dirname := "albums/img/index"
		err := os.MkdirAll(dirname, permDir)
		if err != nil {
			develop.Println(err)
			log.Print(err)
			continue
		}
		url := albums.Entry[i].Thumbnail.MediaUrlBase + "/w197-h134-p/"
		filename := dirname + "/" + albums.Entry[i].GphotoId + ".jpg"
		updated := albums.Entry[i].Updated
		AddWaitGroup(func() {
			writeImage(url, filename, updated)
		})
	}
	return albums
}

func writeRootIndex() (err error) {
	filename := "index.html"
	data := []byte(indexhtml)

	fi, err := os.Stat(filename)
	if err == nil {
		if fi.Size() > 0 {
			return
		}
	}

	fmt.Println("Writing", filename)
	defer fmt.Println("Wrote", filename)

	err = ioutil.WriteFile(filename, data, permFile)
	return
}

func writeTWBS() (err error) {
	fi, err := os.Stat(TWBSfilename)
	if err == nil {
		if fi.Size() > 0 {
			return
		}
	}

	fmt.Println("Downloading", TWBSfilename)
	defer fmt.Println("Downloaded", TWBSfilename)

	trace.Println()
	semaphoreHTTP <- 0
	trace.Println()
	defer func() {
		<-semaphoreHTTP
	}()

	trace.Println()
	resp, err := http.Get(TWBSurl)
	trace.Println()
	if err != nil {
		develop.Println(err)
		log.Print(err)
		return
	}
	defer resp.Body.Close()

	f, closer, err := OpenFile(TWBSfilename)
	defer func() {
		closer <- 0
	}()
	if err != nil {
		develop.Println(err)
		log.Print(err)
		return
	}

	trace.Println()
	written, err := io.Copy(f, resp.Body)
	trace.Println()
	if err1 := f.Close(); err == nil {
		err = err1
		if err != nil {
			develop.Println(err)
			log.Print(err)
		}
	}
	develop.Println(TWBSfilename, written)
	return
}

func isExistsDir(path string) (isExists bool) {
	fi, err := os.Stat(path)
	if err != nil {
		develop.Println(err)
		isExists = false
		return
	}
	isExists = fi.IsDir()
	return
}

func chDir(path string) (err error) {
	if isExistsDir(path) == false {
		err = os.MkdirAll(path, permDir)
		if err != nil {
			develop.Println(err)
			return
		}
	}

	err = os.Chdir(path)
	if err != nil {
		develop.Println(err)
		return
	}
	return
}

func main() {
	start := time.Now()
	fmt.Printf("picasa-dl %v (%v, %v, %v/%v)\n", majorVersion, version, buildAt, runtime.GOOS, runtime.GOARCH)
	defer func() {
		wg.Wait()
		develop.Do(func() {
			runtime.ReadMemStats(&memStats)
			develop.Println(memStats.Alloc, memStats.NumGC)
		})
		log.Println("Finished:", time.Now().Sub(start))
	}()

	var err error

	if distDir != "" {
		err = chDir(distDir)
		if err != nil {
			log.Print(err)
			return
		}
	}

	writeRootIndex()
	writeTWBS()
	albums := getAlbums(userID)
	err = writeIndex(&albums)
	if err != nil {
		develop.Println(err)
		log.Print(err)
		return
	}

	for _, entry := range albums.Entry {
		body, err := HTTPGET(entry.Link)
		if err != nil {
			develop.Println(err)
			log.Print(err)
			continue
		}

		var album Album
		xml.Unmarshal(body, &album)
		AddWaitGroup(func() { writeAlbum(&album) })
	}
}
