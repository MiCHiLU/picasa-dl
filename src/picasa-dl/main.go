package main

import (
	//"fmt"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"text/template"
	"time"
)

const userId = "djmchl@gmail.com"
const permDir os.FileMode = 0755
const permFile os.FileMode = 0644

var debug = debugT(false)

type debugT bool

func (d debugT) Println(args ...interface{}) {
	if d {
		log.Println(args...)
	}
}

var (
	maxProcesses   = runtime.NumCPU()
	semaphoreHTTP  = make(chan int, maxProcesses*2)
	semaphoreFile  = make(chan int, maxProcesses*2)
	workers        [](chan int)
	monitorWorkers = make(chan (chan int))
)

func GoroutineChannel(f func()) (receiver chan int) {
	receiver = make(chan int)
	go func() {
		defer close(receiver)
		f()
	}()
	return
}

func addWorkers(f func()) {
	monitorWorkers <- GoroutineChannel(f)
}

func _monitorWorkers() {
	for {
		worker := <-monitorWorkers
		workers = append(workers, worker)
	}
}

func waitWorkers() {
	for {
		_, ok := <-workers[0]
		if !ok {
			if len(workers) <= 1 {
				return
			}
			workers = workers[1:]
		}
	}
}

/* haml -f html5 -t ugly
!!! 5
%html/
%head
 %link(href="../bootstrap-3.0.0.min.css" rel="stylesheet")
%body/
.row %v
*/
const html = `<!DOCTYPE html>
<html>
<head>
<link href='../bootstrap-3.0.0.min.css' rel='stylesheet'>
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
	t := template.Must(template.New("html").Parse(strings.Replace(html, "%v", li_album, 1)))
	filename := "albums/index.html"
	f, closer, err := OpenFile(filename)
	defer func() {
		closer <- 0
	}()
	if err != nil {
		return err
	}
	err = t.Execute(f, albums)
	if err1 := f.Close(); err == nil {
		err = err1
	}
	log.Print("writeIndex")
	return err
}

func writeAlbum(album *Album) error {
	funcMap := template.FuncMap{
		"timeFormat": func(t time.Time, f string) string {
			return t.Format(f)
		},
	}

	for i := range album.Photo {
		album.Photo[i].Content.SetName()
		album.Photo[i].Content.SetMediaUrlBase()
		album.Photo[i].TimestampTime = time.Unix(album.Photo[i].Timestamp/1000, 0)
		dirname := "albums/img/" + album.GphotoId
		err := os.MkdirAll(dirname, permDir)
		if err != nil {
			log.Print(err)
			continue
		}
		url := album.Photo[i].Content.MediaUrlBase + "w197-h134-p/"
		filename := dirname + "/" + album.Photo[i].Content.Name
		updated := album.Photo[i].Updated
		addWorkers(func() {
			writeImage(url, filename, updated)
		})
	}
	t := template.Must(template.New("html").Funcs(funcMap).Parse(strings.Replace(html, "%v", li_photo, 1)))
	filename := "albums/" + album.GphotoId + ".html"
	f, closer, err := OpenFile(filename)
	defer func() {
		closer <- 0
	}()
	if err != nil {
		return err
	}
	err = t.Execute(f, album)
	if err1 := f.Close(); err == nil {
		err = err1
	}
	log.Println("writeAlbum: ", album.GphotoId)
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
	f, closer, err := OpenFile(filename)
	defer func() {
		closer <- 0
	}()
	if err != nil {
		log.Print(err)
		return
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Print(err)
		return
	}
	defer resp.Body.Close()
	written, err := io.Copy(f, resp.Body)
	if err1 := f.Close(); err == nil {
		err = err1
		if err != nil {
			log.Print(err)
		}
	}
	log.Println("writeImage: ", url, filename, written)
	return
}

func OpenFile(filename string) (file *os.File, closer chan int, err error) {
	semaphoreFile <- 0
	closer = make(chan int)
	go func() {
		<-closer
		close(closer)
		<-semaphoreFile
	}()
	file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
	return
}

func HTTPGET(url string) (body []byte, err error) {
	semaphoreHTTP <- 0
	defer func() {
		<-semaphoreHTTP
	}()
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return
}

func FeedGet(userId string) (body []byte, err error) {
	body, err = HTTPGET("https://picasaweb.google.com/data/feed/api/user/" + userId)
	return
}

func getAlbums(userId string) Albums {
	body, err := FeedGet(userId)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	log.Print("Got album feed")

	var albums Albums
	xml.Unmarshal(body, &albums)
	for i := range albums.Entry {
		albums.Entry[i].SetLink()
		albums.Entry[i].Thumbnail.SetMediaUrlBase()
		dirname := "albums/img/index"
		err := os.MkdirAll(dirname, permDir)
		if err != nil {
			log.Print(err)
			continue
		}
		url := albums.Entry[i].Thumbnail.MediaUrlBase + "/w197-h134-p/"
		filename := dirname + "/" + albums.Entry[i].GphotoId + ".jpg"
		updated := albums.Entry[i].Updated
		addWorkers(func() {
			writeImage(url, filename, updated)
		})
	}
	return albums
}

func main() {
	runtime.GOMAXPROCS(maxProcesses)
	go _monitorWorkers()
	albums := getAlbums(userId)
	err := writeIndex(&albums)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	for _, entry := range albums.Entry {
		body, err := HTTPGET(entry.Link)
		if err != nil {
			log.Print(err)
			continue
		}

		var album Album
		xml.Unmarshal(body, &album)
		addWorkers(func() { writeAlbum(&album) })
	}
	waitWorkers()
}
