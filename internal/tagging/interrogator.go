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
	"sync"
)

type JobResult struct {
	Tags  []string
	Error error
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

func (interrogate *InterrogateForever) TagImage(imageFile multipart.File) (chan JobResult, func(), error) {

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
		interrogate.jobMutex.Lock()
		delete(interrogate.jobs, id)
		interrogate.jobMutex.Unlock()
	}
	go func() {

		imageFilename := fmt.Sprintf("%s.%s", id, extension)
		// Listen for output
		interrogate.jobMutex.Lock()
		interrogate.jobs[id] = responseChan
		interrogate.jobMutex.Unlock()

		// Create file
		err := interrogate.createJob(id, imageFile, imageFilename)
		if err != nil {
			responseChan <- JobResult{nil, err}
		}
	}()
	// block until it's ready, so that it doesn't risk sending a response before it's ready
	return responseChan, cancel, nil
}

func (interrogate *InterrogateForever) createJob(jobId string, imageFile multipart.File, imageFilename string) error {
	zipFilename := fmt.Sprintf("%s.zip", jobId)
	targetPath := filepath.Join(interrogate.InputPath, zipFilename)
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

func (interrogate *InterrogateForever) Start() {
	interrogate.jobs = make(map[string]chan JobResult)
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
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("error:", err)
				}
			}
		}()

		err = watcher.Add(interrogate.OutputPath)
		if err != nil {
			log.Fatal(err)
		}
		// block go routine forever
		<-make(chan struct{})
	}()
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
