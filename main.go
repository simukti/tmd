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
	// BASEURL main tumblr user api base domain
	BASEURL = "http://%s.tumblr.com"

	// APIURL main tumblr user api full path
	APIURL = "%s/api/read?type=%s&num=%d&start=%d"

	// DEFAULTSTART default start post number
	DEFAULTSTART = 0

	// DEFAULTDTO default download timeout
	DEFAULTDTO = 3600

	// DEFAULTCTO default connect timeout, this is used on first request to get total posts
	DEFAULTCTO = 15

	// DEFAULTBATCH default files download at once
	DEFAULTBATCH = 2

	// DEFAULTPERPAGE default posts per page request
	DEFAULTPERPAGE = 20

	// MAXPERBATCH maximum files download at once
	MAXPERBATCH = 10

	// MAXPERPAGE maximum posts perpage request
	MAXPERPAGE = 40

	// DEFAULTMEDIA default value for cli -m param
	DEFAULTMEDIA = "all"

	// PHOTO photo post type
	PHOTO = "photo"

	// VIDEO video post type
	VIDEO = "video"

	// BYTE byte unit float
	BYTE = 1.0

	// KiB Kibibyte
	KiB = 1024 * BYTE

	// MiB Mebibyte
	MiB = 1024 * KiB

	// GiB Gibibyte
	GiB = 1024 * MiB
)

var (
	input         string
	uname         string
	dest          string
	media         string
	list          []string
	batch         int
	cto           int
	dto           int
	perPage       int
	limitPage     int
	counterFile   int
	counterPost   int
	counterSize   int64
	counterStored int64
	allowedMedia  = map[string]bool{"all": true, PHOTO: true, VIDEO: true}
	allMedia      = []string{PHOTO, VIDEO}

	// every http request will randomly pick one user agent from this string list
	defaultUserAgents = [...]string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.11; rv:48.0) Gecko/20100101 Firefox/48.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.91 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/601.7.7 (KHTML, like Gecko) Version/9.1.2 Safari/601.7.7",
		"Mozilla/5.0 (Linux; Android 5.0.2; LG-V410/V41020c Build/LRX22G) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/34.0.1847.118 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 6.0.1; Nexus 6P Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.83 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 5.1.1; SM-G928X Build/LMY47X) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.83 Mobile Safari/537.36",
		"Mozilla/5.0 (Linux; Android 5.0.2; SAMSUNG SM-T550 Build/LRX22G) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/3.3 Chrome/38.0.2125.102 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 5.0.2; LG-V410/V41020c Build/LRX22G) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/34.0.1847.118 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36",
		"Mozilla/5.0 (Windows Phone 10.0; Android 4.2.1; Microsoft; Lumia 950) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2486.0 Mobile Safari/537.36 Edge/13.10586",
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
	username        string
	mainFolder      string
	media           string
	batch           int
	connectTimeout  int
	downloadTimeout int
	start           int
	perPage         int
	limitPage       int
}

func init() {
	// default destination folder is current exexutable dir
	defaultDest, _ := os.Getwd()

	flag.StringVar(&input, "s", ".", "JSON input file")
	flag.StringVar(&uname, "u", ".", "Tumblr username to download, WITHOUT ending .tumblr.com ! -- comma separated for multiple username")
	flag.StringVar(&dest, "d", defaultDest, "Destination directory")
	flag.StringVar(&media, "m", DEFAULTMEDIA, "Media type to download")
	flag.IntVar(&batch, "b", DEFAULTBATCH, "File per download")
	flag.IntVar(&cto, "cto", DEFAULTCTO, "Connect timeout on XML parsing")
	flag.IntVar(&dto, "dto", DEFAULTDTO, "Download timeout per n file batch in param -b")
	flag.IntVar(&perPage, "pp", DEFAULTPERPAGE, "Default post per page")
	flag.IntVar(&limitPage, "lp", 0, "Max page to fetch, 0 is unlimited (all page)")
	flag.Parse()
	flag.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "" {
			msg := color.New(color.FgHiRed, color.Bold).
				SprintfFunc()("[ERROR] Flag param -%s is required", f.Name)
			fmt.Println(msg)
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
		msg := color.New(color.FgHiRed, color.Bold).
			SprintfFunc()("[ERROR] Allowed media is: %s", strings.Join(am, ","))
		fmt.Println(msg)
		os.Exit(0)
	}

	if uname == "." && input == "." {
		msg := color.New(color.FgHiRed, color.Bold).
			SprintfFunc()("[ERROR] Flag param -u (username comma separated) OR -s (json file) IS required !")
		fmt.Println(msg)
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
			msg := color.New(color.FgHiRed, color.Bold).
				SprintfFunc()("[ERROR] %s", err.Error())
			fmt.Println(msg)
			os.Exit(0)
		}
	}

	if err := checkDest(dest); err != nil {
		msg := color.New(color.FgHiRed, color.Bold).
			SprintfFunc()("[ERROR] %s", err.Error())
		fmt.Println(msg)
		os.Exit(0)

	}

	if batch < 1 {
		batch = 1
	}

	if batch > MAXPERBATCH {
		batch = MAXPERBATCH
	}

	if perPage < 1 {
		perPage = DEFAULTPERPAGE
	}

	if perPage > MAXPERPAGE {
		perPage = MAXPERPAGE
	}

	if cto < 1 {
		cto = DEFAULTCTO
	}

	if dto < 1 {
		dto = DEFAULTDTO
	}
}

func main() {
	absDest, _ := filepath.Abs(dest)
	fmt.Println(color.GreenString("[SAVE TO] %s/*", absDest))
	// mulai dari nol ya mbak...
	counterFile = 0
	counterPost = 0
	startTime := time.Now()

	for _, username := range list {
		job := &tumblrJob{
			username:        username,
			mainFolder:      dest,
			media:           media,
			batch:           batch,
			start:           DEFAULTSTART,
			perPage:         perPage,
			limitPage:       limitPage,
			connectTimeout:  cto,
			downloadTimeout: dto,
		}

		if err := job.processJob(); err != nil {
			msg := color.New(color.FgHiRed, color.Bold).
				SprintfFunc()("[ERROR] %s", err.Error())
			fmt.Println(msg)
		}
	}

	processedUsers := strings.Join(list, ",")
	totalDownloaded := float32(counterSize) / GiB
	totalStored := float32(counterStored) / GiB
	totalTime := time.Since(startTime).Seconds()
	summary := color.New(color.FgHiYellow, color.Bold).
		SprintfFunc()(""+
		"\n--------"+
		"\n[USER] [%s]"+
		"\n[POST] %d posts"+
		"\n[FILE] %d files (%.3f GiB)"+
		"\n[SIZE] %.3f GiB downloaded"+
		"\n[TIME] %.2f seconds"+
		"\n--------",
		processedUsers,
		counterPost,
		counterFile,
		totalStored,
		totalDownloaded,
		totalTime,
	)

	fmt.Println(summary)
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
	useragent := defaultUserAgents[rand.Intn(len(defaultUserAgents))]
	client := http.DefaultClient
	userURL := fmt.Sprintf(BASEURL, job.username)
	headReq, _ := http.NewRequest("HEAD", userURL, nil) // check username existence
	headReq.Header.Set("User-Agent", useragent)
	headResp, headErr := client.Do(headReq)
	headError := fmt.Errorf("Unable to parse %s", userURL)
	if headErr != nil {
		return headError
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		return headError
	}

	mediaType := []string{}

	if job.media == "all" {
		mediaType = allMedia
	} else {
		mediaType = append(mediaType, job.media)
	}

	userDir := filepath.Join(job.mainFolder, job.username)
	if _, cErr := os.Stat(userDir); os.IsNotExist(cErr) {
		if err := os.Mkdir(filepath.Join(job.mainFolder, job.username), 0700); err != nil {
			return err
		}
	}

	for _, m := range mediaType {
		job.media = m
		mJob := &mediaJob{
			userURL: userURL,
			mainJob: job,
		}

		mediaDir := filepath.Join(job.mainFolder, job.username, m)
		if _, mErr := os.Stat(mediaDir); os.IsNotExist(mErr) {
			if err := os.Mkdir(mediaDir, 0700); err != nil {
				return err
			}
		}

		if err := mJob.processMedia(); err != nil {
			msg := color.New(color.FgHiRed, color.Bold).
				SprintfFunc()("[ERROR] %s", err.Error())
			// don't cancel job
			fmt.Println(msg)
		}
	}

	return nil
}

type mediaJob struct {
	userURL string
	mainJob *tumblrJob
}

func (m *mediaJob) processMedia() error {
	ping := fmt.Sprintf(APIURL, m.userURL, m.mainJob.media, 0, 0)
	blog, err := getXMLSource(ping, m.mainJob.connectTimeout)
	if err != nil {
		return err
	}

	currentPage := 0
	startAt := m.mainJob.start
	countTotal := float64(blog.Posts.Total) / float64(m.mainJob.perPage)
	totalPage := int(math.Ceil(countTotal))

	if m.mainJob.limitPage != 0 && m.mainJob.limitPage < totalPage {
		if m.mainJob.limitPage < 0 {
			totalPage = 1
			fmt.Println(color.GreenString("[INFO] REVERT LIMIT PAGE TO: %d", totalPage))
		} else {
			totalPage = m.mainJob.limitPage
			fmt.Println(color.GreenString("[INFO] SET LIMIT PAGE TO: %d", totalPage))
		}
	} else {
		fmt.Println(color.GreenString("[INFO] TOTAL PAGE: %d", totalPage))
	}

	for currentPage < totalPage {
		currentPage++
		startAt = (m.mainJob.perPage * (currentPage - 1)) + 1
		if startAt == 1 {
			startAt = 0
		}
		api := fmt.Sprintf(APIURL, m.userURL, m.mainJob.media, m.mainJob.perPage, startAt)
		blogPage, pageErr := getXMLSource(api, m.mainJob.connectTimeout)

		if pageErr != nil {
			msg := color.New(color.FgHiRed, color.Bold).
				SprintfFunc()("[ERROR PAGE %d] [%s]", currentPage, pageErr.Error())
			fmt.Println(msg)
		} else {
			fmt.Println(
				color.CyanString(
					"\n====================================[%s] [%s] [PAGE %d/%d]====================================",
					strings.ToUpper(fmt.Sprintf(BASEURL, m.mainJob.username)),
					strings.ToUpper(m.mainJob.media),
					currentPage,
					totalPage,
				))

			blogPage.processPage(m.mainJob.mainFolder, m.mainJob.batch, m.mainJob.downloadTimeout)
		}
	}

	return nil
}

func getXMLSource(api string, cto int) (*Tumblr, error) {
	t := &Tumblr{}
	useragent := defaultUserAgents[rand.Intn(len(defaultUserAgents))]
	client := &http.Client{Timeout: (time.Second * time.Duration(cto))}
	apiReq, _ := http.NewRequest("GET", api, nil)
	apiReq.Header.Set("User-Agent", useragent)
	apiResp, apiErr := client.Do(apiReq)
	if apiErr != nil {
		return t, apiErr
	}
	defer apiResp.Body.Close()
	err := xml.NewDecoder(apiResp.Body).Decode(t)

	return t, err
}

func (t *Tumblr) processPage(mainTargetFolder string, perBatch, dto int) bool {
	fl := []*fileToDownload{}
	for _, p := range t.Posts.Posts {
		counterPost++

		if p.Type == PHOTO {
			pfj := t.getPhotoFileJob(&p, mainTargetFolder)
			fl = append(fl, pfj...)
		}
		if p.Type == VIDEO {
			vfj := t.getVideoFileJob(&p, mainTargetFolder)
			fl = append(fl, vfj...)
		}
	}

	dl := downloadList{list: fl, perBatch: perBatch, dto: dto, uname: t.TumbleBlog.Name, media: t.Posts.Type}
	return dl.process()
}

func (t *Tumblr) getPhotoFileJob(p *Post, mainTargetFolder string) []*fileToDownload {
	fl := []*fileToDownload{}

	if len(p.PhotoSet.Photo) > 0 {
		for _, psp := range p.PhotoSet.Photo {
			for _, pu := range psp.PhotoURL {
				if fd, ok := pu.normalizePhotoURL(mainTargetFolder, t.TumbleBlog.Name, p); ok {
					fl = append(fl, fd)
				}
			}
		}
	} else {
		for _, pu := range p.PhotoURLs {
			if fd, ok := pu.normalizePhotoURL(mainTargetFolder, t.TumbleBlog.Name, p); ok {
				fl = append(fl, fd)
			}
		}
	}

	return fl
}

func (pu *PhotoURL) normalizePhotoURL(mainfolder, username string, p *Post) (*fileToDownload, bool) {
	if pu.MaxWidth == 1280 {
		fname := normalizeDestination(pu.FileURL, p.ID, p.Timestamp)
		fd := fileToDownload{
			url:      pu.FileURL,
			destFile: filepath.Join(mainfolder, username, p.Type, fname),
		}

		return &fd, true
	}

	return &fileToDownload{}, false
}

func (t *Tumblr) getVideoFileJob(p *Post, mainTargetFolder string) []*fileToDownload {
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
					destFile: filepath.Join(mainTargetFolder, t.TumbleBlog.Name, p.Type, fname),
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
			files:   batchJob,
			timeout: dl.dto,
		}

		fmt.Println(fmt.Sprintf("    [PROCESSING %d %s AT ONCE]", dl.perBatch, strings.ToUpper(dl.media)))
		a.download()
	}

	return true
}

type actualBatchDownload struct {
	timeout int
	files   []*fileToDownload
}

type downloadResult struct {
	timeStart         time.Time
	processError      error
	elapsedDuration   float64
	alreadyDownloaded bool
	sizeStored        int64
	sizeDownloaded    int64
	job               *fileToDownload
}

func (d *actualBatchDownload) download() {
	wg := &sync.WaitGroup{}

	for _, f := range d.files {
		wg.Add(1)

		go func(ftd *fileToDownload) {
			resultChan := make(chan downloadResult, 1)

			go func() {
				startTime := time.Now()
				result := downloadResult{
					job:       ftd,
					timeStart: startTime,
				}

				if s, mErr := os.Stat(ftd.destFile); os.IsNotExist(mErr) {
					fmt.Println(color.WhiteString("\t[DOWNLOADING] [%d] [%s]", startTime.Unix(), ftd.url))
					client := &http.Client{Timeout: (time.Second * time.Duration(d.timeout))}
					useragent := defaultUserAgents[rand.Intn(len(defaultUserAgents))]
					request, _ := http.NewRequest("GET", ftd.url, nil)
					request.Header.Set("User-Agent", useragent)
					response, requestError := client.Do(request)
					output, _ := os.Create(ftd.destFile)
					defer output.Close()

					if requestError != nil {
						_ = os.Remove(ftd.destFile)
						result.processError = requestError
					}

					if response.StatusCode != http.StatusOK {
						_ = os.Remove(ftd.destFile)
						result.processError = fmt.Errorf("[%d] [%s]", response.StatusCode, ftd.url)
					}
					defer response.Body.Close()

					if _, writeError := io.Copy(output, response.Body); writeError != nil {
						_ = os.Remove(ftd.destFile)
						result.processError = writeError
					}

					result.alreadyDownloaded = false
					result.sizeDownloaded = response.ContentLength
					result.sizeStored = response.ContentLength
				} else {
					result.alreadyDownloaded = true
					result.sizeStored = s.Size()
					fmt.Println(color.WhiteString("\t[DOWNLOADED] [%s]", ftd.destFile))
				}

				result.elapsedDuration = time.Since(startTime).Seconds()
				resultChan <- result

				close(resultChan)
			}()

			select {
			case r := <-resultChan:
				if r.processError != nil {
					// file already deleted if any http/io error occured
					msg := color.New(color.FgHiRed, color.Bold).SprintfFunc()("\t[ERROR] %s", r.processError.Error())
					fmt.Println(msg)
				} else {
					counterFile++
					counterStored = (counterStored + r.sizeStored)
					if !r.alreadyDownloaded {
						counterSize = (counterSize + r.sizeDownloaded)
						fmt.Println(color.GreenString("\t[SUCCESS] [%f] [%s]", r.elapsedDuration, r.job.destFile))
					}
				}
				wg.Done()
			case <-time.After(time.Second * time.Duration(d.timeout)):
				r := <-resultChan
				_ = os.Remove(r.job.destFile)
				msg := color.New(color.FgHiMagenta, color.Bold).
					SprintfFunc()("\t[ERROR TIMEOUT] [%f] [%s]", r.elapsedDuration, r.job.url)
				fmt.Println(msg)
				wg.Done()
			}
		}(f)
	}

	wg.Wait()
}
