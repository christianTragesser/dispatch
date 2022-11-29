package dispatch

import (
	"fmt"
	"os"
	"testing"
)

func TestGetNodeSize(t *testing.T) {
	// input string for node size
	// return AWS EC2 type and error
	tests := []struct {
		expectedReturn string
		name           string
		input          string
		err            error
	}{
		{
			name:           "Invalid",
			input:          "x",
			expectedReturn: "",
			err:            fmt.Errorf("invalid node size: x"),
		},
		{
			name:           "Default",
			input:          "S",
			expectedReturn: smallEC2,
			err:            nil,
		},
		{
			name:           "Get l",
			input:          "l",
			expectedReturn: largeEC2,
			err:            nil,
		},
		{
			name:           "Get medium",
			input:          "medium",
			expectedReturn: mediumEC2,
			err:            nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size, err := getNodeSize(test.input)

			if size != test.expectedReturn {
				t.Errorf("getNodeSize unit test failure\n got: '%v', want: '%v', error: '%v'", size, test.expectedReturn, err)
			}
		})
	}
}

func TestSetAWSRegion(t *testing.T) {
	// return discovered or determined region
	tests := []struct {
		expectedReturn string
		name           string
		input          string
	}{
		{
			name:           "Default",
			input:          "",
			expectedReturn: defaultRegion,
		},
		{
			name:           "US West 2",
			input:          "us-west-2",
			expectedReturn: "us-west-2",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.input != "" {
				os.Setenv("AWS_REGION", test.input)
			}

			region := setAWSRegion()

			if region != test.expectedReturn {
				t.Errorf("setAWSRegion unit test failure\n got: '%v', want: '%v'", region, test.expectedReturn)
			}
		})
	}
}
