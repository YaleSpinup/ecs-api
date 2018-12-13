package common

import (
	"bytes"
	"reflect"
	"testing"
)

var testConfig = []byte(
	`{
		"listenAddress": ":8000",
		"accounts": {
		  "provider1": {
			"region": "us-east-1",
			"akid": "key1",
			"secret": "secret1",
			"defaultSgs": ["sg-xxxxxx", "sg-yyyyyy"],
			"defaultSubnets": ["subnet-xxxxxxx", "subnet-yyyyyy"],
			"defaultExecutionRoleArn": "arn:aws:iam::1111111111111:role/ecsTaskExecutionRole"
		  },
		  "provider2": {
			"region": "us-west-1",
			"akid": "key2",
			"secret": "secret2"
		  }
		},
		"token": "SEKRET",
		"logLevel": "info"
	}`)

func TestReadConfig(t *testing.T) {
	expectedConfig := Config{
		ListenAddress: ":8000",
		Accounts: map[string]Account{
			"provider1": Account{
				Region:                  "us-east-1",
				Akid:                    "key1",
				Secret:                  "secret1",
				DefaultSgs:              []string{"sg-xxxxxx", "sg-yyyyyy"},
				DefaultSubnets:          []string{"subnet-xxxxxxx", "subnet-yyyyyy"},
				DefaultExecutionRoleArn: "arn:aws:iam::1111111111111:role/ecsTaskExecutionRole",
			},
			"provider2": Account{
				Region: "us-west-1",
				Akid:   "key2",
				Secret: "secret2",
			},
		},
		Token:    "SEKRET",
		LogLevel: "info",
	}

	actualConfig, err := ReadConfig(bytes.NewReader(testConfig))
	if err != nil {
		t.Error("Failed to read config", err)
	}

	if !reflect.DeepEqual(actualConfig, expectedConfig) {
		t.Errorf("Expected config to be %+v\n got %+v", expectedConfig, actualConfig)
	}
}
