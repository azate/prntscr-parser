package main

import (
	"flag"
	"fmt"
	"github.com/martinlindhe/base36"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type JobStatus uint8

type Job struct {
	index  uint64
	status JobStatus
}

type Result struct {
	job Job
	err error
}

type Jobs chan Job

type Results chan Result

const (
	JobStatusNotProcessed        JobStatus = 0
	JobStatusErrorPageRequest    JobStatus = 1
	JobStatusErrorPageResponse   JobStatus = 2
	JobStatusErrorPageContent    JobStatus = 3
	JobStatusAccessDenied        JobStatus = 4
	JobStatusCaptcha             JobStatus = 5
	JobStatusNoImage             JobStatus = 6
	JobStatusErrorImageRequest   JobStatus = 7
	JobStatusErrorImageResponse  JobStatus = 8
	JobStatusErrorImageFile      JobStatus = 9
	JobStatusErrorImageFileWrite JobStatus = 10
	JobStatusSuccess             JobStatus = 11
)

var (
	indexStarting      uint64
	indexFinal         uint64
	workerCount        uint64
	imagesPath         string
	proxiesFilePath    string
	userAgentsFilePath string
)

func doWork(job Job) (Job, error) {
	proxyUrl := proxies.Get()
	userAgent := userAgents.Get()

	cookieJar, _ := cookiejar.New(nil)
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		},
		Jar:     cookieJar,
		Timeout: time.Minute * 2,
	}

	indexEncoded := base36.Encode(job.index)
	pageUrlString := "https://prnt.sc/" + strings.ToLower(indexEncoded)

	pageRequest, err := http.NewRequest("GET", pageUrlString, nil)

	if err != nil {
		job.status = JobStatusErrorPageRequest
		return job, err
	}

	pageRequest.Header.Set("Connection", "close")
	pageRequest.Header.Set("User-Agent", *userAgent)
	pageResponse, err := httpClient.Do(pageRequest)

	if err != nil {
		job.status = JobStatusErrorPageResponse
		return job, err
	}

	defer pageResponse.Body.Close()
	pageContentBytes, err := ioutil.ReadAll(pageResponse.Body)

	if err != nil {
		job.status = JobStatusErrorPageContent
		return job, err
	}

	pageContentString := string(pageContentBytes)

	accessDeniedMessage := `<h2 class="cf-subheadline">Access denied</h2>`
	accessDenied := strings.Index(pageContentString, accessDeniedMessage)

	if accessDenied != -1 {
		job.status = JobStatusAccessDenied
		return job, nil
	}

	accessDeniedMessage2 := `<span data-translate="complete_sec_check">Please complete the security check to access</span>`
	accessDenied2 := strings.Index(pageContentString, accessDeniedMessage2)

	if accessDenied2 != -1 {
		job.status = JobStatusCaptcha
		return job, nil
	}

	re := regexp.MustCompile(`<meta property="og:image" content="(.*?)"/>`)
	matches := re.FindStringSubmatch(pageContentString)

	if matches == nil || len(matches) != 2 {
		job.status = JobStatusNoImage
		return job, nil
	}

	imageUrl := matches[1]
	imageUrlSeparated := strings.Split(imageUrl, "/")
	imageUrlFileName := imageUrlSeparated[len(imageUrlSeparated)-1]

	imageFileNamePrefix := strconv.FormatUint(job.index, 10)
	imageFileName := imageFileNamePrefix + "." + imageUrlFileName
	imageRequest, err := http.NewRequest("GET", imageUrl, nil)

	if err != nil {
		job.status = JobStatusErrorImageRequest
		return job, err
	}

	imageRequest.Header.Set("Connection", "close")
	imageRequest.Header.Set("User-Agent", *userAgent)
	imageResponse, err := httpClient.Do(imageRequest)

	if err != nil {
		job.status = JobStatusErrorImageResponse
		return job, err
	}

	defer imageResponse.Body.Close()
	imageFilePath := imagesPath + "/" + imageFileName
	imageFile, err := os.Create(imageFilePath)

	if err != nil {
		job.status = JobStatusErrorImageFile
		return job, err
	}

	defer imageFile.Close()
	_, err = io.Copy(imageFile, imageResponse.Body)

	if err != nil {
		job.status = JobStatusErrorImageFileWrite
		return job, err
	}

	job.status = JobStatusSuccess
	return job, nil
}

func worker(jobs Jobs, results Results) {
	for job := range jobs {
		jobResult, err := doWork(job)
		result := Result{job: jobResult, err: err}
		results <- result
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())

	flag.Uint64Var(&indexStarting, "indexStarting", 1355412729, "index-starting")
	flag.Uint64Var(&indexFinal, "indexFinal", 1355412729, "index-final")
	flag.Uint64Var(&workerCount, "workerCount", 10, "workerCount")
	flag.StringVar(&imagesPath, "images", "images", "path to save file")
	flag.StringVar(&proxiesFilePath, "proxies", "proxies.csv", "path to file")
	flag.StringVar(&userAgentsFilePath, "userAgents", "user-agents.csv", "path to file")
}

func main() {
	flag.Parse()

	proxiesErr := proxies.AddFromFile(proxiesFilePath)
	if proxiesErr != nil {
		fmt.Println(proxiesErr)
		return
	}

	userAgentsErr := userAgents.AddFromFile(userAgentsFilePath)
	if userAgentsErr != nil {
		fmt.Println(userAgentsErr)
		return
	}

	jobs := make(Jobs, workerCount)
	results := make(Results, workerCount)

	for i := uint64(1); i <= workerCount; i++ {
		go worker(jobs, results)
	}

	go func() {
		for i := indexStarting; i <= indexFinal; i++ {
			job := Job{index: i, status: JobStatusNotProcessed}
			jobs <- job
		}
	}()

	indexesProcessed := 1
	indexesNumber := indexFinal - indexStarting + 1
	for result := range results {
		fmt.Printf(
			"(%d of %d, index: %d, status: %d) [ERROR] %v\n",
			indexesProcessed,
			indexesNumber,
			result.job.index,
			result.job.status,
			result.err,
		)
		indexesProcessed++
	}
}
