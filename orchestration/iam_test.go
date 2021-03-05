package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	im "github.com/YaleSpinup/ecs-api/iam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
)

var orch = Orchestrator{
	IAM: im.IAM{
		DefaultKmsKeyID: "123",
	},
}

var (
	path     = "org/super-why"
	testTime = time.Now()
)

var defaultPolicyDoc = im.PolicyDoc{
	Version: "2012-10-17",
	Statement: []im.PolicyStatement{
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
				fmt.Sprintf("arn:aws:secretsmanager:*:*:secret:spinup/%s/*", path),
				fmt.Sprintf("arn:aws:ssm:*:*:parameter/%s/*", path),
				fmt.Sprintf("arn:aws:kms:*:*:key/%s", orch.IAM.DefaultKmsKeyID),
			},
		},
	},
}

var outdatedPolicyDoc = im.PolicyDoc{
	Version: "2012-10-17",
	Statement: []im.PolicyStatement{
		{
			Effect: "Allow",
			Action: []string{
				"ecr:*",
			},
			Resource: []string{"*"},
		},
	},
}

var testRoles = map[string]iam.Role{
	"super-why-ecsTaskExecution": {
		Arn:         aws.String("arn:aws:iam::12345678910:role/super-why-ecsTaskExecution"),
		CreateDate:  &testTime,
		Description: aws.String("role model"),
		Path:        aws.String("/"),
		RoleId:      aws.String("TESTROLEID123"),
		RoleName:    aws.String("super-why-ecsTaskExecution"),
	},
	"mr-rogers-ecsTaskExecution": {
		Arn:         aws.String("arn:aws:iam::12345678910:role/mr-rogers-ecsTaskExecution"),
		CreateDate:  &testTime,
		Description: aws.String("role model"),
		Path:        aws.String("/"),
		RoleId:      aws.String("TESTROLEID000"),
		RoleName:    aws.String("mr-rogers-ecsTaskExecution"),
	},
	"missingpolicy-ecsTaskExecution": {
		Arn:         aws.String("arn:aws:iam::12345678910:role/missingpolicy-ecsTaskExecution"),
		CreateDate:  &testTime,
		Description: aws.String("role model"),
		Path:        aws.String("/"),
		RoleId:      aws.String("TESTROLEID000"),
		RoleName:    aws.String("missingpolicy-ecsTaskExecution"),
	},
	"badpolicy-ecsTaskExecution": {
		Arn:         aws.String("arn:aws:iam::12345678910:role/org/badpolicy-ecsTaskExecution"),
		CreateDate:  &testTime,
		Description: aws.String("role model"),
		Path:        aws.String("/"),
		RoleId:      aws.String("TESTROLEID000"),
		RoleName:    aws.String("badpolicy-ecsTaskExecution"),
	},
}

func (m *mockIAMClient) GetRoleWithContext(ctx context.Context, input *iam.GetRoleInput, opts ...request.Option) (*iam.GetRoleOutput, error) {
	if m.err != nil {
		if aerr, ok := (m.err).(awserr.Error); ok {
			if aerr.Code() == "TestNoSuchEntity" {
				return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
			}
		}
		return nil, m.err
	}

	r, ok := testRoles[aws.StringValue(input.RoleName)]
	if !ok {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
	}

	return &iam.GetRoleOutput{Role: &r}, nil
}

func (m *mockIAMClient) CreateRoleWithContext(ctx context.Context, input *iam.CreateRoleInput, opts ...request.Option) (*iam.CreateRoleOutput, error) {
	role := iam.Role{
		Arn:         aws.String(fmt.Sprintf("arn:aws:iam::12345678910:role/%s", *input.RoleName)),
		CreateDate:  &testTime,
		Description: input.Description,
		Path:        input.Path,
		RoleId:      aws.String(strings.ToUpper(fmt.Sprintf("%sID123", *input.RoleName))),
		RoleName:    input.RoleName,
	}

	output := &iam.CreateRoleOutput{Role: &role}

	if m.err != nil {
		if aerr, ok := (m.err).(awserr.Error); ok {
			if aerr.Code() == "TestNoSuchEntity" {
				return output, nil
			}
		}
		return nil, m.err
	}

	return output, nil
}

func (m *mockIAMClient) GetRolePolicyWithContext(ctx context.Context, input *iam.GetRolePolicyInput, opts ...request.Option) (*iam.GetRolePolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.PolicyName) != "ECSTaskAccessPolicy" || aws.StringValue(input.RoleName) == "missingpolicy" {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "policy not found", nil)
	}

	if aws.StringValue(input.RoleName) == "badpolicy-ecsTaskExecution" {
		return &iam.GetRolePolicyOutput{
			PolicyDocument: aws.String("{}"),
			RoleName:       input.RoleName,
		}, nil
	}

	var p im.PolicyDoc
	if aws.StringValue(input.RoleName) == "super-why-ecsTaskExecution" {
		p = defaultPolicyDoc
	} else if aws.StringValue(input.RoleName) == "mr-rogers-ecsTaskExecution" {
		p = outdatedPolicyDoc
	} else {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "role not found", nil)
	}

	pdoc, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	return &iam.GetRolePolicyOutput{
		PolicyDocument: aws.String(string(pdoc)),
		RoleName:       input.RoleName,
	}, nil
}

func (m *mockIAMClient) PutRolePolicyWithContext(ctx context.Context, input *iam.PutRolePolicyInput, opts ...request.Option) (*iam.PutRolePolicyOutput, error) {
	var output = &iam.PutRolePolicyOutput{}

	if m.err != nil {
		if aerr, ok := (m.err).(awserr.Error); ok {
			if aerr.Code() == "TestNoSuchEntity" {
				return output, nil
			}
		}
		return nil, m.err
	}

	return output, nil
}

func TestOrchestrator_DefaultTaskExecutionPolicy(t *testing.T) {
	type fields struct {
		IAM im.IAM
	}
	type args struct {
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   im.PolicyDoc
	}{
		{
			name: "test example",
			fields: fields{
				IAM: im.IAM{
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				path: path,
			},
			want: defaultPolicyDoc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				IAM: tt.fields.IAM,
			}
			if got := o.DefaultTaskExecutionPolicy(tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Orchestrator.DefaultTaskExecutionPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrchestrator_DefaultTaskExecutionRole(t *testing.T) {
	type fields struct {
		IAM im.IAM
	}
	type args struct {
		ctx  context.Context
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "empty path",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "",
			},
			wantErr: true,
		},
		{
			name: "example path",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: path,
			},
			want: "arn:aws:iam::12345678910:role/super-why-ecsTaskExecution",
		},
		{
			name: "create missing role",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "missing",
			},
			want: "arn:aws:iam::12345678910:role/missing-ecsTaskExecution",
		},
		{
			name: "policy doc that needs updating",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "org/mr-rogers",
			},
			want: "arn:aws:iam::12345678910:role/mr-rogers-ecsTaskExecution",
		},
		{
			name: "existing role, missing pollicy document",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "org/missingpolicy",
			},
			want: "arn:aws:iam::12345678910:role/missingpolicy-ecsTaskExecution",
		},
		{
			name: "existing role, invalid pollicy document",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "org/badpolicy",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				IAM: tt.fields.IAM,
			}
			got, err := o.DefaultTaskExecutionRole(tt.args.ctx, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.DefaultTaskExecutionRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Orchestrator.DefaultTaskExecutionRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrchestrator_createDefaultTaskExecutionRole(t *testing.T) {
	type fields struct {
		IAM im.IAM
	}
	type args struct {
		ctx  context.Context
		path string
		role string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "empty path and role",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "",
				role: "",
			},
			wantErr: true,
		},
		{
			name: "empty path",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "",
				role: "super-why-ecsTaskExecution",
			},
			want: "arn:aws:iam::12345678910:role/super-why-ecsTaskExecution",
		},
		{
			name: "empty role",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "org/super-why",
				role: "",
			},
			wantErr: true,
		},
		{
			name: "example test role",
			fields: fields{
				IAM: im.IAM{
					Service:         newMockIAMClient(t, nil),
					DefaultKmsKeyID: "123",
				},
			},
			args: args{
				ctx:  context.TODO(),
				path: "org/super-why",
				role: "testrole",
			},
			want: "arn:aws:iam::12345678910:role/testrole",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				IAM: tt.fields.IAM,
			}
			got, err := o.createDefaultTaskExecutionRole(tt.args.ctx, tt.args.path, tt.args.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.defaultTaskExecutionRoleArn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Orchestrator.defaultTaskExecutionRoleArn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_assumeRolePolicy(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name: "assume role policy document",
			want: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["sts:AssumeRole"],"Principal":{"Service":["ecs-tasks.amazonaws.com"]}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := assumeRolePolicy()
			if (err != nil) != tt.wantErr {
				t.Errorf("assumeRolePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("assumeRolePolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_assumeRolePolicy(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := assumeRolePolicy()
		if err != nil {
			b.Errorf("expected nil error, got %s", err)
		}
	}
}
