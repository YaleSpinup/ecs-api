package iam

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var testPolicy = PolicyDoc{
	Version: "2012-10-17",
	Statement: []PolicyStatement{
		{
			Effect: "Allow",
			Action: []string{
				"ecr:GetAuthorizationToken",
				"logs:CreateLogGroup",
				"logs:CreateLogStream",
				"logs:PutLogEvents",
			},
			Resource: []string{"*"},
		},
		{
			Effect: "Allow",
			Action: []string{
				"secretsmanager:GetSecretValue",
				"ssm:GetParameters",
				"kms:Decrypt",
			},
			Resource: []string{
				"arn:aws:secretsmanager:*:*:secret:spinup/foobar/*",
				"arn:aws:ssm:*:*:parameter/foobar/*",
				"arn:aws:kms:*:*:key/1484468c-abb3-463f-a397-41529fece4c2",
			},
		},
	},
}

var testPolicyDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ecr:GetAuthorizationToken","logs:CreateLogGroup","logs:CreateLogStream","logs:PutLogEvents"],"Resource":["*"]},{"Effect":"Allow","Action":["secretsmanager:GetSecretValue","ssm:GetParameters","kms:Decrypt"],"Resource":["arn:aws:secretsmanager:*:*:secret:spinup/foobar/*","arn:aws:ssm:*:*:parameter/foobar/*","arn:aws:kms:*:*:key/1484468c-abb3-463f-a397-41529fece4c2"]}]}`

// mockIAMClient is a fake IAM client
type mockIAMClient struct {
	iamiface.IAMAPI
	t   *testing.T
	err error
}

func newMockIAMClient(t *testing.T, err error) iamiface.IAMAPI {
	return &mockIAMClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "iam.IAM" {
		t.Errorf("expected type to be 'iam.IAM', got %s", to)
	}
}

func TestDocumentFromPolicy(t *testing.T) {
	type args struct {
		policy *PolicyDoc
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "nil input",
			args: args{
				policy: nil,
			},
			wantErr: true,
		},
		{
			name: "example policy",
			args: args{
				policy: &testPolicy,
			},
			want: testPolicyDocument,
		},
		{
			name: "empty policy",
			args: args{
				policy: &PolicyDoc{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DocumentFromPolicy(tt.args.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("DocumentFromPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DocumentFromPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolicyFromDocument(t *testing.T) {
	type args struct {
		doc string
	}
	tests := []struct {
		name    string
		args    args
		want    *PolicyDoc
		wantErr bool
	}{
		{
			name: "empty input",
			args: args{
				doc: "",
			},
			wantErr: true,
		},
		{
			name: "example policy document",
			args: args{
				doc: testPolicyDocument,
			},
			want: &testPolicy,
		},
		{
			name: "empty policy document statements",
			args: args{
				doc: "{}",
			},
			wantErr: true,
		},
		{
			name: "bad JSON",
			args: args{
				doc: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["ecr:GetAuthorizationToken","logs:CreateLogGroup","logs:CreateLogStream","logs:PutLogEvents"],"Resource":["*"]}`,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PolicyFromDocument(tt.args.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("PolicyFromDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PolicyFromDocument() = %v, want %v", got, tt.want)
			}
		})
	}
}
