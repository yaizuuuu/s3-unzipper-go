package main

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	sampleFile = "sample.zip"
	testFile = "testdata/" + sampleFile
)

var (
	srcBucket = os.Getenv("ZIPPED_ARTIFACT_BUCKET") + "-dev"
)

func TestHandler(t *testing.T)  {
	events := events.S3Event{
		Records: []events.S3EventRecord{
			{
				S3: events.S3Entity{
					Bucket: events.S3Bucket{Name: srcBucket},
					Object: events.S3Object{Key: sampleFile},
				},
			},
		},
	}

	ctx := context.Background()
	lc := &lambdacontext.LambdaContext{
		AwsRequestID: "test request",
	}
	ctx = lambdacontext.NewContext(ctx, lc)

	err := handler(ctx, events)
	if err != nil {
		t.Error(err)
	}
}

func setup()  {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	svc := s3.New(sess)

	destBucket = os.Getenv("UNZIPPED_ARTIFACT_BUCKET") + "-dev"
	for _, b := range []string{srcBucket, destBucket} {
		if !bucketExists(svc, b) {
			_, err := svc.CreateBucket(&s3.CreateBucketInput{
				Bucket: aws.String(b),
			})
			if err != nil {
				panic(err)
			}
		}
	}

	file, err := os.Open(testFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(srcBucket),
		Key: aws.String(sampleFile),
		Body: file,
	})
	if err != nil {
		panic(err)
	}
}

func teardown()  {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	svc := s3.New(sess)

	for _, b := range []string{srcBucket, destBucket} {
		iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
			Bucket: aws.String(b),
		})

		if err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter); err != nil {
			panic(err)
		}

		_, err := svc.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: aws.String(b),
		})
		if err != nil {
			panic(err)
		}
	}
}

func bucketExists(svc *s3.S3, bucket string) bool {
	input := &s3.HeadBucketInput{Bucket: aws.String(bucket)}
	_, err := svc.HeadBucket(input)
	if err != nil {
		return false
	}

	return true
}

func TestMain(m *testing.M)  {
	setup()
	exitCode := m.Run()
	teardown()
	os.Exit(exitCode)
}
