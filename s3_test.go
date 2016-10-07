package veneur

import (
	"encoding/json"
	"os"
	"path"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
)

const S3TestBucket = "stripe-test-veneur"

type mockS3Client struct {
	s3iface.S3API
	putObject func(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return m.putObject(input)
}

// TestS3Post tests that we can correctly post a sequence of
// DDMetrics to S3
func TestS3Post(t *testing.T) {
	RemoteResponseChan := make(chan struct{}, 1)
	defer func() {
		select {
		case <-RemoteResponseChan:
			// all is safe
			return
		case <-time.After(DefaultServerTimeout):
			assert.Fail(t, "Global server did not complete all responses before test terminated!")
		}
	}()

	client := &mockS3Client{}
	f, err := os.Open(path.Join("fixtures", "aws", "PutObject", "2016", "10", "07", "1475863542.json"))
	assert.NoError(t, err)
	defer f.Close()

	client.putObject = func(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
		var data []DDMetric
		assert.NoError(t, err)
		json.NewDecoder(input.Body).Decode(&data)
		assert.Equal(t, 6, len(data))
		assert.Equal(t, "a.b.c.max", data[0].Name)
		RemoteResponseChan <- struct{}{}
		return &s3.PutObjectOutput{ETag: aws.String("912ec803b2ce49e4a541068d495ab570")}, nil
	}

	svc = client

	err = s3Post("testbox", f)
	assert.NoError(t, err)
}

func TestS3Path(t *testing.T) {
	const hostname = "testingbox-9f23c"

	start := time.Now()

	path := s3Path(hostname)

	// We expect paths to follow this format
	// <year>/<month/<day>/<hostname>/<timestamp>.json
	// so we should be able to parse the output with this expectation
	re := regexp.MustCompile(`(\d{4}?)/(\d{2}?)/(\d{2}?)/([\w\-]+?)/(\d+?).json`)
	results := re.FindStringSubmatch(*path)

	year, err := strconv.Atoi(results[1])
	assert.NoError(t, err)
	month, err := strconv.Atoi(results[2])
	assert.NoError(t, err)
	day, err := strconv.Atoi(results[3])
	assert.NoError(t, err)

	sameYear := year == int(time.Now().Year()) ||
		year == int(start.Year())
	sameMonth := month == int(time.Now().Month()) ||
		month == int(start.Month())
	sameDay := day == int(time.Now().Day()) ||
		day == int(start.Day())

	// we may have started the tests a split-second before midnight
	assert.True(t, sameYear, "Expected year %s and received %s", start.Year(), year)
	assert.True(t, sameMonth, "Expected month %s and received %s", start.Month(), month)
	assert.True(t, sameDay, "Expected day %d and received %s", start.Day(), day)
}

func TestS3PostNoCredentials(t *testing.T) {
	svc = nil

	f, err := os.Open(path.Join("fixtures", "aws", "PutObject", "2016", "10", "07", "1475863542.json"))
	assert.NoError(t, err)
	defer f.Close()

	// this should not panic
	err = s3Post("testbox", f)
	assert.Equal(t, S3ClientUninitializedError, err)
}
