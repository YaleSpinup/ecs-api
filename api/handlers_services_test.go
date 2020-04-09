package api

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func TestParseLogQuery(t *testing.T) {
	type logQueryParseTest struct {
		query string
		input *cloudwatchlogs.GetLogEventsInput
		err   error
	}

	tests := []logQueryParseTest{
		// happy path
		{
			query: "",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   nil,
		},
		{
			query: "task=abcdefg&container=hijklmnop",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   nil,
		},
		{
			query: "limit=5",
			input: &cloudwatchlogs.GetLogEventsInput{
				Limit: aws.Int64(int64(5)),
			},
			err: nil,
		},
		{
			query: "limit=5&seq=abc12345",
			input: &cloudwatchlogs.GetLogEventsInput{
				Limit:         aws.Int64(int64(5)),
				NextToken:     aws.String("abc12345"),
				StartFromHead: aws.Bool(true),
			},
			err: nil,
		},
		{
			query: "start=1234567&end=7654321",
			input: &cloudwatchlogs.GetLogEventsInput{
				StartTime: aws.Int64(int64(1234567)),
				EndTime:   aws.Int64(int64(7654321)),
			},
			err: nil,
		},
		{
			query: "limit=5&seq=abc12345&start=1234567&end=7654321",
			input: &cloudwatchlogs.GetLogEventsInput{
				Limit:         aws.Int64(int64(5)),
				NextToken:     aws.String("abc12345"),
				StartFromHead: aws.Bool(true),
				StartTime:     aws.Int64(int64(1234567)),
				EndTime:       aws.Int64(int64(7654321)),
			},
			err: nil,
		},
		// errors
		{
			query: "limit=true",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   errors.New("strconv.ParseInt: parsing \"true\": invalid syntax"),
		},
		{
			query: "limit=abcdefghijklmnopqrstuvwxyz",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   errors.New("strconv.ParseInt: parsing \"abcdefghijklmnopqrstuvwxyz\": invalid syntax"),
		},
		{
			query: "start=true",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   errors.New("strconv.ParseInt: parsing \"true\": invalid syntax"),
		},
		{
			query: "start=abcdefghijklmnopqrstuvwxyz",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   errors.New("strconv.ParseInt: parsing \"abcdefghijklmnopqrstuvwxyz\": invalid syntax"),
		},
		{
			query: "end=true",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   errors.New("strconv.ParseInt: parsing \"true\": invalid syntax"),
		},
		{
			query: "end=abcdefghijklmnopqrstuvwxyz",
			input: &cloudwatchlogs.GetLogEventsInput{},
			err:   errors.New("strconv.ParseInt: parsing \"abcdefghijklmnopqrstuvwxyz\": invalid syntax"),
		},
	}

	for _, test := range tests {
		input := &cloudwatchlogs.GetLogEventsInput{}
		r := &http.Request{
			URL: &url.URL{
				RawQuery: test.query,
			},
		}

		t.Logf("testing raw query %s", test.query)

		if err := parseLogQuery(r, input); err != nil {
			if test.err == nil {
				t.Errorf("expected nil error, got %s", err)
			} else if test.err.Error() != err.Error() {
				t.Errorf("expected error %s, got %s", test.err, err)
			}
		} else {
			if !reflect.DeepEqual(input, test.input) {
				t.Errorf("expected %+v, got %+v", test.input, input)
			}
		}
	}
}
