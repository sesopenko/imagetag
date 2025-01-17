package tagging

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type JobResult struct {
	Tags  []string
	Error error
}

type ResultFile struct {
	JobId string   `json:"job_id"`
	Model string   `json:"model"`
	Tags  []string `json:"tags"`
	Error string   `json:"error"`
}

type jobSpec struct {
	ModelName          string `json:"model_name"`
	JobId              string `json:"job_id"`
	InputImageFilename string `json:"input_image_filename"`
}

type InterrogateForever struct {
	InputPath  string
	OutputPath string
	jobs       map[string]chan JobResult
	jobMutex   sync.Mutex
}

func BuildAndStart(inputPath string, outputPath string) *InterrogateForever {
	i := InterrogateForever{
		InputPath:  filepath.Clean(inputPath),
		OutputPath: filepath.Clean(outputPath),
	}
	i.Start()
	return &i
}

func (i *InterrogateForever) TagImage(imageFile multipart.File) (chan JobResult, func(), error) {

	mimeType, err := detectMimeType(imageFile)
	if err != nil {
		return nil, nil, err
	}
	extension, err := mimeToExtension(mimeType)
	if err != nil {
		return nil, nil, err
	}
	responseChan := make(chan JobResult)
	id := uuid.New().String()
	cancel := func() {
		i.jobMutex.Lock()
		delete(i.jobs, id)
		i.jobMutex.Unlock()
	}
	go func() {

		imageFilename := fmt.Sprintf("%s.%s", id, extension)
		// Listen for output
		i.jobMutex.Lock()
		i.jobs[id] = responseChan
		i.jobMutex.Unlock()

		// Create file
		err := i.createJob(id, imageFile, imageFilename)
		if err != nil {
			responseChan <- JobResult{nil, err}
		}
	}()
	// block until it's ready, so that it doesn't risk sending a response before it's ready
	return responseChan, cancel, nil
}

func (i *InterrogateForever) createJob(jobId string, imageFile multipart.File, imageFilename string) error {
	zipFilename := fmt.Sprintf("%s.zip", jobId)
	targetPath := filepath.Join(i.InputPath, zipFilename)
	zipFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("could not create zip file: %s", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	imageWriter, err := zipWriter.Create(imageFilename)
	if err != nil {
		return fmt.Errorf("could not create image file: %s", err)
	}
	if _, err := io.Copy(imageWriter, imageFile); err != nil {
		return fmt.Errorf("could not copy image file: %s", err)
	}

	// Add the job spec json to the zip
	validModel := "SmilingWolf/wd-vit-large-tagger-v3"
	job := jobSpec{
		ModelName:          validModel,
		JobId:              jobId,
		InputImageFilename: imageFilename,
	}

	jsonWriter, err := zipWriter.Create("job.json")
	if err != nil {
		return fmt.Errorf("could not create job.json: %s", err)
	}
	if err := json.NewEncoder(jsonWriter).Encode(job); err != nil {
		return fmt.Errorf("could not encode job: %s", err)
	}

	return nil

}

func (i *InterrogateForever) Start() {
	i.jobs = make(map[string]chan JobResult)
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					log.Println("event:", event)
					if event.Has(fsnotify.Write) {
						log.Println("modified file:", event.Name)
						i.HandleResponse(event.Name)
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("error:", err)
				}
			}
		}()

		err = watcher.Add(i.OutputPath)
		if err != nil {
			log.Fatal(err)
		}
		// block go routine forever
		<-make(chan struct{})
	}()
}

func (i *InterrogateForever) HandleResponse(filePath string) {

	filename := filepath.Base(filePath)
	parts := strings.Split(filename, ".")
	if len(parts) != 2 {
		log.Printf("file not named correctly: %s", filename)
		// todo: error and delete
	}
	id := parts[0]
	file, err := os.Open(filePath)
	if err != nil {
		// todo: delete it
		i.respondError(id, err)
		log.Printf("could not open file: %s", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	var resultFile ResultFile

	err = decoder.Decode(&resultFile)
	if err != nil {
		i.respondError(id, err)
		log.Printf("could not decode file: %s", err)
	}
	i.respondSuccess(id, resultFile.Tags)
	if err := os.Remove(filePath); err != nil {
		log.Printf("could not remove file: %s", err)
	}

}

func (i *InterrogateForever) respondSuccess(id string, tags []string) {
	response := JobResult{
		Tags:  tags,
		Error: nil,
	}
	i.SendResponse(id, response)
}

func (i *InterrogateForever) respondError(id string, err error) {
	response := JobResult{
		Error: err,
	}
	i.SendResponse(id, response)
}

func (i *InterrogateForever) SendResponse(id string, response JobResult) {
	i.jobMutex.Lock()
	ch, exists := i.jobs[id]
	if exists {

		ch <- response
		delete(i.jobs, id)
	}
	i.jobMutex.Unlock()
}

func detectMimeType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	mimeType := http.DetectContentType(buffer)

	// reset file pointer to beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("failed to reset file pointer: %v", err)
	}
	return mimeType, nil
}

func mimeToExtension(mimeType string) (string, error) {
	switch mimeType {
	case "image/png":
		return "png", nil
	case "image/jpeg":
		return "jpg", nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", mimeType)
	}
}
