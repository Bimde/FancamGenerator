package main

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/sns"
)

const (
	awsRegion    = "us-east-1"
	functionName = "tracking_converter"
	loggingName  = "lambda"
)

var (
	svc *rekognition.Rekognition
	topic *sns.SNS
)

type responseBody struct {
	Response string `json:"response"`
}

type rekSNSNotification struct {
	JobID  string `json:"JobId"`
	Status string `json:"Status"`
}

func process(notification *rekSNSNotification) error {
	log := getLogger("process")
	var (
		maxResults      int64 = 100
		paginationToken *string
		finished        = false
		totalCount      = 0
		count           = 0
		noPeople        = int64(0)
	)
	log.WithField("JobID", notification.JobID).Infof("Starting processing")
	for !finished {
		x := rekognition.GetPersonTrackingInput{
			JobId:      aws.String(notification.JobID),
			MaxResults: &maxResults,
			NextToken:  paginationToken,
		}
		results, err := svc.GetPersonTracking(&x)
		if err != nil {
			return err
		}

		log.Debug(results.VideoMetadata)

		for _, p := range results.Persons {
			totalCount++

			person := p.Person
			if person == nil {
				continue
			}
			if *person.Index > noPeople {
				noPeople = *person.Index
			}

			count++
			log.Debugf("Person (index=%d)", *person.Index)
			log.Debug("	Timestamp: ", *p.Timestamp)

			boundingBox := person.BoundingBox
			if boundingBox == nil {
				continue
			}

			GetClient(*person.Index).AddTrackingFrame(*p.Timestamp, *boundingBox.Width, *boundingBox.Left)
			log.Debug("	Bounding Box")
			log.Debugf("		Top: %f", *boundingBox.Top)
			log.Debugf("		Left: %f", *boundingBox.Left)
			log.Debugf("		Width: %f", *boundingBox.Width)
			log.Debugf("		Height: %f", *boundingBox.Height)
		}

		if results.NextToken == nil {
			finished = true
		} else {
			paginationToken = results.NextToken
		}
	}

	log.Info("Number of PersonDetection objects: ", totalCount)
	log.Info("Number of People: ", noPeople+1)

	exports := TriggerAllExportsTrimmed()
	NotifyExportCompletion(exports, nil)
	
	for _, e := range *exports {
		log.WithField("Project", e.ProjectURL).Infof("Export: %s", e.URL)
	}

	return nil
}

func handle(ctx context.Context, snsEvent events.SNSEvent) (events.APIGatewayProxyResponse, error) {
	log := getLogger("handle")
	log.Info("Post: ", snsEvent)
	log.Info("context ", ctx)
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}

	var notification rekSNSNotification
	jsonParseError := json.Unmarshal([]byte(snsEvent.Records[0].SNS.Message), &notification)
	if jsonParseError != nil {
		log.Error(jsonParseError)
		return events.APIGatewayProxyResponse{500, headers, nil, "Internal Server Error", false}, nil
	}

	log.Info("SNS event received: ", notification)

	err := process(&notification)
	if err != nil {
		log.Error("Error processing event: ", err)
		return events.APIGatewayProxyResponse{500, headers, nil, "Internal Server Error", false}, nil
	}

	code := 200

	response, jsonBuildError := json.Marshal(responseBody{Response: "Processing Successful"})
	if jsonBuildError != nil {
		log.Error(jsonBuildError)
		response = []byte("Internal Server Error")
		code = 500
	}

	return events.APIGatewayProxyResponse{code, headers, nil, string(response), false}, nil
}

func main() {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)
	svc = rekognition.New(session)
	topic = sns.New(session)

	log := getLogger("main")

	if err != nil {
		log.Panic("Error initiating session ", err)
	} else {
		log.Info("Successfully initiated session")
		lambda.Start(handle)
	}
}

func getLogger(method string) *log.Entry {
	return log.WithFields(log.Fields{
		"method": fmt.Sprintf("%s#%s", loggingName, method),
	})
}
