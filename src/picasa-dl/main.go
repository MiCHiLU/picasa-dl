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

var (
	maxProcesses = runtime.NumCPU()
	semaphore    = make(chan int, maxProcesses*2)
)

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
	Updated string  `xml:"updated"`
	Entry   []Entry `xml:"entry"`
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
		c.Name = bits[len(bits)-1]
	}
	return
}

func (c *Content) SetMediaUrlBase() {
	if c.MediaUrlBase == "" {
		c.MediaUrlBase = strings.Split(c.Src, c.Name)[0]
	}
	return
}

func writeIndex(albums *Albums) error {
	t := template.Must(template.New("html").Parse(strings.Replace(html, "%v", li_album, 1)))
	filename := "albums/index.html"
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
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
	semaphore <- 1
	defer func() {
		<-semaphore
	}()

	funcMap := template.FuncMap{
		"timeFormat": func(t time.Time, f string) string {
			return t.Format(f)
		},
	}

	c := make(chan int)
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
		go writeImage(album.Photo[i].Content.MediaUrlBase+"w197-h134-p/", dirname+"/"+album.Photo[i].Content.Name, album.Photo[i].Updated)
	}
	t := template.Must(template.New("html").Funcs(funcMap).Parse(strings.Replace(html, "%v", li_photo, 1)))
	filename := "albums/" + album.GphotoId + ".html"
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
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
	semaphore <- 1
	defer func() {
		<-semaphore
	}()
	fi, err := os.Stat(filename)
	if err == nil {
		if fi.Size() > 0 {
			t, _ := time.Parse("2006-01-02T15:04:05.000Z", updated)
			if fi.ModTime().Sub(t) > 0 {
				return
			}
		}
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
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

func getAlbums() Albums {
	resp, err := http.Get("https://picasaweb.google.com/data/feed/api/user/" + userId)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	log.Print("Got album feed")

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

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
		go writeImage(albums.Entry[i].Thumbnail.MediaUrlBase+"/w197-h134-p/", dirname+"/"+albums.Entry[i].GphotoId+".jpg", albums.Entry[i].Updated)
	}
	return albums
}

func main() {
	runtime.GOMAXPROCS(maxProcesses)
	albums := getAlbums()
	err := writeIndex(&albums)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	for _, entry := range albums.Entry {
		resp, err := http.Get(entry.Link)
		if err != nil {
			log.Print(err)
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Print(err)
			continue
		}

		var album Album
		xml.Unmarshal(body, &album)
		go writeAlbum(&album)
	}
}
