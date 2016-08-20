// Copyright (c) 2016 - Sarjono Mukti Aji <me@simukti.net>
// Unless otherwise noted, this source code license is MIT-License

package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	baseURL        = "http://%s.tumblr.com"
	apiURL         = "%s/api/read?type=%s&num=%d&start=%d"
	defaultPerPage = 20
	start          = 0
	defaultDto     = 3600
	defaultCto     = 10
	defaultBatch   = 2
	defaultMedia   = "all"
	photo          = "photo"
	video          = "video"
)

var (
	input        string
	uname        string
	dest         string
	media        string
	list         []string
	batch        int
	cto          int
	dto          int
	perPage      int
	limitPage    int
	allowedMedia = map[string]bool{"all": true, photo: true, video: true}
	allMedia     = []string{photo, video}
	userAgents   = []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.11; rv:48.0) Gecko/20100101 Firefox/48.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.91 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/601.7.7 (KHTML, like Gecko) Version/9.1.2 Safari/601.7.7",
	}
)

// Tumblr parent xml result
type Tumblr struct {
	TumbleBlog TumbleBlog `xml:"tumblelog"`
	Posts      Posts      `xml:"posts"`
}

// TumbleBlog detail of current tumblr blog
type TumbleBlog struct {
	Name      string `xml:"name,attr"`
	Timezone  string `xml:"timezone,attr"`
	Canonical string `xml:"cname,attr"`
}

// Posts entities of current page
type Posts struct {
	Type  string `xml:"type,attr"`
	Start int    `xml:"start,attr"`
	Total int    `xml:"total,attr"`
	Posts []Post `xml:"post"`
}

// Post single entity which contain photo/video list
type Post struct {
	ID            string        `xml:"id,attr"`
	Timestamp     int           `xml:"unix-timestamp,attr"`
	Type          string        `xml:"type,attr"`
	Slug          string        `xml:"slug,attr"`
	PhotoURLs     []PhotoURL    `xml:"photo-url"`         // only available in photo media type
	PhotoSet      Photoset      `xml:"photoset"`          // only available in photo media type (optional)
	IsDirectVideo bool          `xml:"direct-video,attr"` // only available in video media type
	VideoPlayer   []VideoPlayer `xml:"video-player"`      // only available in video media type
	VideoSource   VideoSource   `xml:"video-source"`      // only available in video media type
}

// VideoSource single detail on VideoPlayer
type VideoSource struct {
	ContentType string `xml:"content-type"`
	Extension   string `xml:"extension"`
	Width       int    `xml:"width"`
	Height      int    `xml:"height"`
	Duration    int    `xml:"duration"`
}

// VideoPlayer entity which contain video file url
type VideoPlayer struct {
	MaxWidth int    `xml:"max-width,attr"`
	Content  string `xml:",chardata"`
}

// PhotoURL entity which contain photo file url
type PhotoURL struct {
	MaxWidth int    `xml:"max-width,attr"`
	FileURL  string `xml:",chardata"`
}

// Photoset optional entity which can be exists on photo Post
type Photoset struct {
	Photo []Photo `xml:"photo"`
}

// Photo entities which exists on Photoset, this contains photo file urls
type Photo struct {
	PhotoURL []PhotoURL `xml:"photo-url"`
}

// tumblrJob main tumblr download jobs per username and media
type tumblrJob struct {
	uname           string
	dest            string
	media           string
	batch           int
	connectTimeout  int
	downloadTimeout int
	start           int
	perPage         int
	limitPage       int
}

func main() {
	// default destination folder is current exexutable dir
	defaultDest, _ := os.Getwd()

	flag.StringVar(&input, "s", ".", "JSON input file")
	flag.StringVar(&uname, "u", ".", "Tumblr username to download, WITHOUT ending .tumblr.com ! -- comma separated for multiple username")
	flag.StringVar(&dest, "d", defaultDest, "Destination directory")
	flag.StringVar(&media, "m", defaultMedia, "Media type to download")
	flag.IntVar(&batch, "b", defaultBatch, "File per download")
	flag.IntVar(&cto, "cto", defaultCto, "Connect timeout on XML parsing")
	flag.IntVar(&dto, "dto", defaultDto, "Download timeout per n file batch in param -b")
	flag.IntVar(&perPage, "pp", defaultPerPage, "Default post per page")
	flag.IntVar(&limitPage, "lp", 0, "Max page to fetch, 0 is unlimited (all page)")
	flag.Parse()

	flag.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "" {
			fmt.Println(color.RedString("Flag param -%s is required", f.Name))
			fmt.Println("Usage:")
			flag.PrintDefaults()
			os.Exit(0)
		}
	})

	if !allowedMedia[media] {
		am := []string{}
		for m := range allowedMedia {
			am = append(am, m)
		}
		fmt.Println(color.RedString("Allowed media is: %s", strings.Join(am, ",")))
		os.Exit(0)
	}

	if uname == "." && input == "." {
		fmt.Println(color.RedString("Flag param -u (username comma separated) OR -s (json file) IS required !"))
		fmt.Println("Usage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if uname != "." {
		sp := strings.Split(uname, ",")
		for _, su := range sp {
			if su != "" {
				list = append(list, strings.TrimSpace(strings.ToLower(su)))
			}
		}
	} else {
		if err := loadList(input); err != nil {
			fmt.Println(color.RedString(err.Error()))
			os.Exit(0)
		}
	}

	if err := checkDest(dest); err != nil {
		fmt.Println(color.RedString(err.Error()))
		os.Exit(0)

	}
	fmt.Println(color.GreenString("DESTINATION: %s", dest))

	for _, u := range list {
		job := &tumblrJob{
			uname:           u,
			dest:            dest,
			media:           media,
			batch:           batch,
			start:           start,
			perPage:         perPage,
			limitPage:       limitPage,
			connectTimeout:  cto,
			downloadTimeout: dto,
		}

		if err := job.processJob(); err != nil {
			fmt.Println(color.RedString("[ERROR] => %s", err.Error()))
		}
	}
}

func checkDest(dir string) error {
	abs, absErr := filepath.Abs(dir)
	if absErr != nil {
		return fmt.Errorf("Unable to parse %s", abs)
	}

	s, sErr := os.Stat(abs)
	if sErr != nil {
		if os.IsNotExist(sErr) {
			return fmt.Errorf("Destination folder '%s' not found, create first, I'll do the rest", abs)
		}
	}

	if !s.IsDir() {
		return fmt.Errorf("Destination '%s' is not a folder", abs)
	}

	return nil
}

func loadList(file string) error {
	abs, absErr := filepath.Abs(file)
	if absErr != nil {
		return fmt.Errorf("Unable to parse %s", abs)
	}

	s, sErr := os.Stat(abs)
	if sErr != nil {
		if os.IsNotExist(sErr) {
			return fmt.Errorf("Input file %s not found", abs)
		}
	}

	if cur, _ := os.Getwd(); cur == abs || s.IsDir() {
		return errors.New("Input file not found")
	}

	r, rErr := os.Open(abs)
	if rErr != nil {
		return fmt.Errorf("Input file %s cannot be opened", abs)
	}
	defer r.Close()

	if err := json.NewDecoder(r).Decode(&list); err != nil {
		return errors.New("JSON decoding error, make sure input file contains json string list")
	}

	for k, u := range list {
		list[k] = strings.TrimSpace(strings.ToLower(u))
	}

	return nil
}

func (job *tumblrJob) processJob() error {
	ua := userAgents[rand.Intn(len(userAgents))]
	client := http.DefaultClient
	userURL := fmt.Sprintf(baseURL, job.uname)
	headReq, _ := http.NewRequest("HEAD", userURL, nil) // check username existence
	headReq.Header.Set("User-Agent", ua)
	headResp, headErr := client.Do(headReq)
	headError := fmt.Errorf("Unable to parse %s", userURL)
	if headErr != nil {
		return headError
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		return headError
	}

	var mediaType []string

	if job.media == "all" {
		mediaType = allMedia
	} else {
		mediaType = append(mediaType, job.media)
	}

	userDir := filepath.Join(job.dest, job.uname)
	if _, cErr := os.Stat(userDir); os.IsNotExist(cErr) {
		if err := os.Mkdir(filepath.Join(job.dest, job.uname), 0777); err != nil {
			return err
		}
	}

	for _, m := range mediaType {
		job.media = m
		mJob := &mediaJob{
			userURL: userURL,
			mainJob: job,
		}

		mediaDir := filepath.Join(job.dest, job.uname, m)
		if _, mErr := os.Stat(mediaDir); os.IsNotExist(mErr) {
			if err := os.Mkdir(mediaDir, 0777); err != nil {
				return err
			}
		}

		if err := mJob.processMedia(); err != nil {
			// don't cancel job
			fmt.Println(color.RedString("[ERROR] %s", err.Error()))
		}
	}

	return nil
}

type mediaJob struct {
	userURL string
	mainJob *tumblrJob
}

func (m *mediaJob) processMedia() error {
	ping := fmt.Sprintf(apiURL, m.userURL, m.mainJob.media, 0, 0)
	blog, err := getXMLSource(ping, m.mainJob.connectTimeout)
	if err != nil {
		return err
	}

	currentPage := 0
	startAt := m.mainJob.start
	countTotal := float64(blog.Posts.Total) / float64(m.mainJob.perPage)
	totalPage := int(math.Ceil(countTotal))

	if m.mainJob.limitPage != 0 && m.mainJob.limitPage < totalPage {
		totalPage = m.mainJob.limitPage
	}

	for currentPage < totalPage {
		currentPage++
		startAt = int((m.mainJob.perPage * (currentPage - 1)) + 1)
		if startAt == 1 {
			startAt = 0
		}
		api := fmt.Sprintf(apiURL, m.userURL, m.mainJob.media, m.mainJob.perPage, startAt)

		blogPage, pageErr := getXMLSource(api, m.mainJob.connectTimeout)
		if pageErr != nil {
			fmt.Println(pageErr.Error())
		} else {
			fmt.Println(
				color.CyanString(
					"\n[%s] [%s] [PAGE %d/%d]",
					strings.ToLower(fmt.Sprintf(baseURL, m.mainJob.uname)),
					strings.ToUpper(m.mainJob.media),
					currentPage,
					totalPage,
				))

			blogPage.processPage(m.mainJob.dest, m.mainJob.batch, m.mainJob.downloadTimeout)
		}
	}

	return nil
}

func getXMLSource(api string, cto int) (*Tumblr, error) {
	t := &Tumblr{}
	ua := userAgents[rand.Intn(len(userAgents))]
	client := &http.Client{Timeout: (time.Second * time.Duration(cto))}
	apiReq, _ := http.NewRequest("GET", api, nil)
	apiReq.Header.Set("User-Agent", ua)
	apiResp, apiErr := client.Do(apiReq)
	if apiErr != nil {
		return t, apiErr
	}
	defer apiResp.Body.Close()

	if err := xml.NewDecoder(apiResp.Body).Decode(t); err != nil {
		return t, err
	}

	return t, nil
}

func (t *Tumblr) processPage(mainDestFolder string, perBatch, dto int) bool {
	fl := []*fileToDownload{}
	for _, p := range t.Posts.Posts {
		if p.Type == photo {
			pfj := t.getPhotoFileJob(&p, mainDestFolder)
			fl = append(fl, pfj...)
		}
		if p.Type == video {
			vfj := t.getVideoFileJob(&p, mainDestFolder)
			fl = append(fl, vfj...)
		}
	}

	dl := downloadList{list: fl, perBatch: perBatch, dto: dto, uname: t.TumbleBlog.Name, media: t.Posts.Type}
	return dl.process()
}

func (t *Tumblr) getPhotoFileJob(p *Post, mainDestFolder string) []*fileToDownload {
	fl := []*fileToDownload{}

	if len(p.PhotoSet.Photo) > 0 {
		for _, psp := range p.PhotoSet.Photo {
			for _, pu := range psp.PhotoURL {
				if pu.MaxWidth == 1280 {
					fname := normalizeDestination(pu.FileURL, p.ID, p.Timestamp)
					fd := fileToDownload{
						url:      pu.FileURL,
						destFile: filepath.Join(mainDestFolder, t.TumbleBlog.Name, p.Type, fname),
					}
					fl = append(fl, &fd)
				}
			}
		}
	} else {
		for _, pu := range p.PhotoURLs {
			if pu.MaxWidth == 1280 {
				fname := normalizeDestination(pu.FileURL, p.ID, p.Timestamp)
				fd := fileToDownload{
					url:      pu.FileURL,
					destFile: filepath.Join(mainDestFolder, t.TumbleBlog.Name, p.Type, fname),
				}
				fl = append(fl, &fd)
			}
		}
	}

	return fl
}

func (t *Tumblr) getVideoFileJob(p *Post, mainDestFolder string) []*fileToDownload {
	fl := []*fileToDownload{}

	for _, vp := range p.VideoPlayer {
		// only direct video will be downloaded
		if vp.MaxWidth == 0 && p.IsDirectVideo {
			rgx := regexp.MustCompile(`<source[^>]+\bsrc=["']([^"']+)["']`)
			match := rgx.FindStringSubmatch(vp.Content)
			if len(match) == 2 {
				videoURL := match[1]
				fname := fmt.Sprintf(
					"%s.%s",
					normalizeDestination(videoURL, p.ID, p.Timestamp),
					p.VideoSource.Extension,
				)

				fd := fileToDownload{
					url:      videoURL,
					destFile: filepath.Join(mainDestFolder, t.TumbleBlog.Name, p.Type, fname),
				}
				fl = append(fl, &fd)
			}
		}
	}

	return fl
}

func normalizeDestination(sourceURL string, id string, timestamp int) string {
	fu, _ := url.Parse(sourceURL)
	psplit := strings.Split(fu.Path, "/")
	fname := fmt.Sprintf("%s_%d_%s", id, timestamp, psplit[len(psplit)-1])

	return fname
}

type fileToDownload struct {
	url      string
	destFile string
}

type downloadList struct {
	media    string
	uname    string
	perBatch int
	dto      int
	list     []*fileToDownload
}

// process download per batch concurrently
func (dl *downloadList) process() bool {
	lenList := len(dl.list)
	countBatch := float64(lenList) / float64(dl.perBatch)
	totalBatch := int(math.Ceil(countBatch))
	if totalBatch < 1 {
		return false
	}

	currentBatch := 0
	start := 0
	end := 0

	for currentBatch < totalBatch {
		currentBatch++
		if currentBatch == 1 {
			if totalBatch == 1 {
				end = lenList - 1
			} else {
				end = start + (start + (dl.perBatch - 1))
			}
		} else {
			start = (end + 1)
			end = (start + (dl.perBatch - 1))
			if totalBatch > 1 && currentBatch == totalBatch {
				end = (lenList - 1)
			}
		}

		size := (end - start) + 1
		batchJob := make([]*fileToDownload, size)

		n := start
		c := 0
		for n <= end {
			batchJob[c] = dl.list[n]
			n++
			c++
		}

		a := actualBatchDownload{
			file: batchJob,
			dto:  dl.dto,
		}

		fmt.Println(fmt.Sprintf("    [%s DOWNLOAD]", strings.ToUpper(dl.media)))
		a.download()
	}

	return true
}

type actualBatchDownload struct {
	dto  int
	file []*fileToDownload
}

func (d *actualBatchDownload) download() bool {
	wg := sync.WaitGroup{}

	for _, f := range d.file {
		wg.Add(1)

		go func(fd *fileToDownload) {
			st := make(chan time.Time, 1)
			done := make(chan error, 1)
			elapsed := make(chan float64, 1)
			cur := make(chan string, 1)
			res := make(chan string, 1)

			go func() {
				startTime := time.Now()
				st <- startTime
				cur <- fd.url
				res <- fd.destFile

				if _, mErr := os.Stat(fd.destFile); os.IsNotExist(mErr) {
					fmt.Println(color.WhiteString("\t[DOWNLOADING] [%d] [%s]", startTime.Unix(), fd.url))
					client := &http.Client{Timeout: (time.Second * time.Duration(d.dto))}
					ua := userAgents[rand.Intn(len(userAgents))]
					req, _ := http.NewRequest("GET", fd.url, nil)
					req.Header.Set("User-Agent", ua)
					r, rErr := client.Do(req)
					o, _ := os.Create(fd.destFile)
					defer o.Close()

					if rErr != nil {
						os.Remove(fd.destFile)
						done <- rErr
					}

					if r.StatusCode != http.StatusOK {
						os.Remove(f.destFile)
						done <- fmt.Errorf("%d <- %s", r.StatusCode, f.url)
					}
					defer r.Body.Close()

					if _, dErr := io.Copy(o, r.Body); dErr != nil {
						os.Remove(fd.destFile)
						done <- dErr
					}
				} else {
					fmt.Println(color.WhiteString("\t[ALREADY_DOWNLOADED] [%s]", fd.url))
				}

				endTime := time.Now().Sub(startTime).Seconds()
				elapsed <- endTime

				close(st)
				close(done)
				close(elapsed)
				close(cur)
				close(res)
			}()

			select {
			case err := <-done:
				if err != nil {
					// file already deleted if any http/io error occured
					fmt.Println(color.RedString("\t[ERROR] [%s]", err.Error()))
				} else {
					fmt.Println(color.GreenString("\t[SUCCESS] [%f] [%s]", <-elapsed, <-res))
				}
			case <-time.After(time.Second * time.Duration(d.dto)):
				os.Remove(f.destFile)
				fmt.Println(color.MagentaString("\t[ERROR TIMEOUT] [%f] [%s]", <-elapsed, <-cur))
			}
			wg.Done()
		}(f)
	}

	wg.Wait()

	return true
}
