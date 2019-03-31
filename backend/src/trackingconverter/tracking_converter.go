package main

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

const (
	awsRegion       string = "us-east-1"
	functionName    string = "tracking_converter"
	tempPersonIndex        = 0
	loggingName            = "lambda"
)

var svc *rekognition.Rekognition

type ResponseBody struct {
	Response string `json:"response"`
}

type RekSNSNotification struct {
	JobID  string `json:"JobId"`
	Status string `json:"Status"`
}

func process(notification *RekSNSNotification) error {
	log := getLogger("process")
	var (
		maxResults      int64 = 100
		paginationToken *string
		finished        = false
		totalCount      = 0
		count           = 0
	)
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

		log.Info(results.VideoMetadata)

		for _, p := range results.Persons {
			totalCount++

			person := p.Person
			if person == nil || *person.Index != tempPersonIndex {
				continue
			}
			// if person == nil {
			// 	continue
			// }
			count++
			log.Printf("Person (index=%d)", *person.Index)

			boundingBox := person.BoundingBox
			if boundingBox == nil {
				continue
			}

			addTrackingFrame(*p.Timestamp, *boundingBox.Width, *boundingBox.Left)
			log.Println("	Timestamp: ", *p.Timestamp)
			log.Println("	Bounding Box")
			log.Printf("		Top: %f", *boundingBox.Top)
			log.Printf("		Left: %f", *boundingBox.Left)
			log.Printf("		Width: %f", *boundingBox.Width)
			log.Printf("		Height: %f", *boundingBox.Height)
		}

		if results.NextToken == nil {
			finished = true
		} else {
			paginationToken = results.NextToken
		}
	}

	log.Info("ProjectID: ", project.ID)
	log.Info("Number of PersonDetection objects: ", totalCount)
	log.WithField("index", tempPersonIndex).Info("Number of PersonDetection objects for index: ", count)

	err := saveClip()
	if err != nil {
		log.Error("error saving clip ", err)
		return err
	}

	return nil
}

func handle(ctx context.Context, snsEvent events.SNSEvent) (events.APIGatewayProxyResponse, error) {
	log := getLogger("handle")
	log.Info("Post: ", snsEvent)
	log.Info("context ", ctx)
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}

	var notification *RekSNSNotification
	jsonParseError := json.Unmarshal([]byte(snsEvent.Records[0].SNS.Message), notification)
	if jsonParseError != nil {
		log.Error(jsonParseError)
		return events.APIGatewayProxyResponse{500, headers, nil, "Internal Server Error", false}, nil
	}

	log.Info("SNS event received: ", notification)

	process(notification)

	code := 200

	// TODO change response
	response, jsonBuildError := json.Marshal(ResponseBody{Response: "TODO"})
	if jsonBuildError != nil {
		log.Error(jsonBuildError)
		response = []byte("Internal Server Error")
		code = 500
	}

	return events.APIGatewayProxyResponse{code, headers, nil, string(response), false}, nil
}

// func main() {
// 	session, err := session.NewSession(&aws.Config{
// 		Region: aws.String(awsRegion)},
// 	)
// 	svc = rekognition.New(session)

// 	log := getLogger("main")

// 	if err != nil {
// 		log.Panic("Error initiating session ", err)
// 	} else {
// 		log.Info("Successfully initiated session")
// 		lambda.Start(handle)
// 	}
// }

func getLogger(method string) *log.Entry {
	return log.WithFields(log.Fields{
		"method": fmt.Sprintf("%s#%s", loggingName, method),
	})
}

func main() {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)
	if err != nil {
		log.Error(err)
		panic(err)
	}

	svc = rekognition.New(session)

	err = process(&RekSNSNotification{JobID: "51a3a9bed1dca4015708e18b24c884ecde6212fb738870500bbd440ad284e2f1"})
	if err != nil {
		log.Error(err)
	}
}
