package openshot

import (
	"fmt"
	"httputils"
)

const (
	filesEndpoint        = baseURL + "/projects/%d/files/"
	fileEndpoint         = baseURL + "/files/%d/"
	s3DefaultFilesFolder = "files/"
	s3DefaultBucket      = "fancamgenerator"
)

// GetProjects returns a list of all projects created
func (o *OpenShot) GetFiles(projectID int) (*Files, error) {
	log := getLogger("GetFiles")
	var files Files
	httputils.Get(log, fmt.Sprintf(filesEndpoint, projectID), nil, &files)
	return &files, nil
}

// CreateFile adds file to openshot from location on s3. The projectURL of the
// given file (if empty) is overriden with one matching the specified projectID.
// The URL (if empty) is overriden with "files/NAME"
func (o *OpenShot) CreateFile(projectID int, file *FileUploadS3) (*File, error) {
	log := getLogger("CreateFile")
	setDefaults(file, projectID)
	var createdFile File
	httputils.Post(log, fmt.Sprintf(filesEndpoint, projectID), file, &createdFile)
	return &createdFile, nil
}

func createFileStruct(testFileName string) *FileUploadS3 {
	return &FileUploadS3{JSON: FileS3Info{Name: testFileName}}
}

func setDefaults(file *FileUploadS3, projectID int) {
	if file.ProjectURL == "" {
		file.ProjectURL = projectURL(projectID)
	}
	if file.JSON.URL == "" {
		file.JSON.URL = s3DefaultFilesFolder + file.JSON.Name
	}
	if file.JSON.Bucket == "" {
		file.JSON.Bucket = s3DefaultBucket
	}
}

// DeleteFile deletes the file from openshot and associated storage
func (o *OpenShot) DeleteFile(fileID int) error {
	log := getLogger("DeleteFile")
	return httputils.Delete(log, fmt.Sprintf(fileEndpoint, fileID), nil, nil)
}