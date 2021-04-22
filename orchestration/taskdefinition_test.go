package orchestration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
)

func (m *mockCWLClient) CreateLogGroupWithContext(ctx context.Context, input *cloudwatchlogs.CreateLogGroupInput, opts ...request.Option) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.CreateLogGroupOutput{}, nil
}

func (m *mockCWLClient) PutRetentionPolicyWithContext(ctx context.Context, input *cloudwatchlogs.PutRetentionPolicyInput, opts ...request.Option) (*cloudwatchlogs.PutRetentionPolicyOutput, error) {
	return &cloudwatchlogs.PutRetentionPolicyOutput{}, nil
}

func (m *mockECSClient) RegisterTaskDefinitionWithContext(ctx aws.Context, input *ecs.RegisterTaskDefinitionInput, opts ...request.Option) (*ecs.RegisterTaskDefinitionOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	output := &ecs.RegisterTaskDefinitionOutput{
		Tags: input.Tags,
		TaskDefinition: &ecs.TaskDefinition{
			Compatibilities:         input.RequiresCompatibilities,
			ContainerDefinitions:    input.ContainerDefinitions,
			Cpu:                     input.Cpu,
			ExecutionRoleArn:        input.ExecutionRoleArn,
			Family:                  input.Family,
			InferenceAccelerators:   input.InferenceAccelerators,
			IpcMode:                 input.IpcMode,
			Memory:                  input.Memory,
			NetworkMode:             input.NetworkMode,
			PidMode:                 input.PidMode,
			PlacementConstraints:    input.PlacementConstraints,
			ProxyConfiguration:      input.ProxyConfiguration,
			RequiresAttributes:      []*ecs.Attribute{},
			RequiresCompatibilities: input.RequiresCompatibilities,
			Revision:                aws.Int64(1),
			Status:                  aws.String("ACTIVE"),
			TaskDefinitionArn:       aws.String("arn:aws:ecs:us-east-1:0123456789:task-definition/" + aws.StringValue(input.Family) + ":1"),
			TaskRoleArn:             input.TaskRoleArn,
			Volumes:                 input.Volumes,
		},
	}

	return output, nil
}

func (m *mockIAMClient) TagRoleWithContext(ctx context.Context, input *iam.TagRoleInput, opts ...request.Option) (*iam.TagRoleOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.TagRoleOutput{}, nil
}

func TestOrchestrator_processTaskDefinitionCreate(t *testing.T) {
	t.Log("testing processTaskDefinitionCreate")

	type fields struct {
		org     string
		cwlerr  error
		ecserr  error
		iamerr  error
		rgtaerr error
		smerr   error
		sderr   error
	}
	type args struct {
		ctx   context.Context
		input *ServiceOrchestrationInput
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ecs.TaskDefinition
		// TODO test rollback function
		// want1   rollbackFunc
		wantErr bool
	}{
		{
			name: "nil input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "empty input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx:   context.TODO(),
				input: &ServiceOrchestrationInput{},
			},
			wantErr: true,
		},
		{
			name: "example input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
				input: &ServiceOrchestrationInput{
					Cluster: &ecs.CreateClusterInput{
						ClusterName: aws.String("clu1"),
					},
					Service: &ecs.CreateServiceInput{},
					TaskDefinition: &ecs.RegisterTaskDefinitionInput{
						ContainerDefinitions: []*ecs.ContainerDefinition{
							{
								Name:  aws.String("haxserver"),
								Image: aws.String("nginx:alpine"),
								PortMappings: []*ecs.PortMapping{
									{ContainerPort: aws.Int64(80)},
									{ContainerPort: aws.Int64(443)},
								},
							},
						},
						Cpu:    aws.String("256"),
						Family: aws.String("datfam"),
						Memory: aws.String("512"),
					},
					Tags: []*Tag{
						{
							Key:   aws.String("Application"),
							Value: aws.String("derpderpderp"),
						},
					},
				},
			},
			want: &ecs.TaskDefinition{
				Compatibilities: aws.StringSlice([]string{"FARGATE"}),
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Image: aws.String("nginx:alpine"),
						LogConfiguration: &ecs.LogConfiguration{
							LogDriver: aws.String("awslogs"),
							Options: map[string]*string{
								"awslogs-group":         aws.String("clu1"),
								"awslogs-stream-prefix": aws.String("datfam"),
								"awslogs-region":        aws.String("us-east-1"),
								"awslogs-create-group":  aws.String("true"),
							},
						},
						Name: aws.String("haxserver"),
						PortMappings: []*ecs.PortMapping{
							{ContainerPort: aws.Int64(80)},
							{ContainerPort: aws.Int64(443)},
						},
					},
				},
				Cpu:                     aws.String("256"),
				Family:                  aws.String("datfam"),
				Memory:                  aws.String("512"),
				ExecutionRoleArn:        aws.String("arn:aws:iam::12345678910:role/clu1-ecsTaskExecution"),
				NetworkMode:             aws.String("awsvpc"),
				RequiresAttributes:      []*ecs.Attribute{},
				RequiresCompatibilities: aws.StringSlice([]string{"FARGATE"}),
				Revision:                aws.Int64(1),
				Status:                  aws.String("ACTIVE"),
				TaskDefinitionArn:       aws.String("arn:aws:ecs:us-east-1:0123456789:task-definition/datfam:1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newMockOrchestrator(t, tt.fields.org,
				tt.fields.cwlerr, tt.fields.ecserr, tt.fields.iamerr,
				tt.fields.rgtaerr, tt.fields.smerr, tt.fields.sderr)
			got, _, err := o.processTaskDefinitionCreate(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.processTaskDefinitionCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Orchestrator.processTaskDefinitionCreate() got = %v, want %v", got, tt.want)
			}

			if !tt.wantErr {
				td := fmt.Sprintf("%s:%d", aws.StringValue(got.Family), aws.Int64Value(got.Revision))
				if gottd := aws.StringValue(tt.args.input.Service.TaskDefinition); gottd != td {
					t.Errorf("Orchestrator.processTaskDefinitionCreate() gottd = %v, want %v", gottd, td)
				}
			}

			// if !reflect.DeepEqual(got1, tt.want1) {
			// 	t.Errorf("Orchestrator.processTaskDefinitionCreate() got1 = %v, want %v", got1, tt.want1)
			// }
		})
	}
}

func TestOrchestrator_processTaskTaskDefinitionCreate(t *testing.T) {
	t.Log("testing processTaskTaskDefinitionCreate")

	type fields struct {
		org     string
		cwlerr  error
		ecserr  error
		iamerr  error
		rgtaerr error
		smerr   error
		sderr   error
	}
	type args struct {
		ctx   context.Context
		input *TaskDefCreateOrchestrationInput
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ecs.TaskDefinition
		// TODO test rollback function
		// want1   rollbackFunc
		wantErr bool
	}{
		{
			name: "nil input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "empty input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx:   context.TODO(),
				input: &TaskDefCreateOrchestrationInput{},
			},
			wantErr: true,
		},
		{
			name: "example input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
				input: &TaskDefCreateOrchestrationInput{
					Cluster: &ecs.CreateClusterInput{
						ClusterName: aws.String("clu1"),
					},
					TaskDefinition: &ecs.RegisterTaskDefinitionInput{
						ContainerDefinitions: []*ecs.ContainerDefinition{
							{
								Name:  aws.String("haxserver"),
								Image: aws.String("nginx:alpine"),
								PortMappings: []*ecs.PortMapping{
									{ContainerPort: aws.Int64(80)},
									{ContainerPort: aws.Int64(443)},
								},
							},
						},
						Cpu:    aws.String("256"),
						Family: aws.String("datfam"),
						Memory: aws.String("512"),
					},
					Tags: []*Tag{
						{
							Key:   aws.String("Application"),
							Value: aws.String("derpderpderp"),
						},
					},
				},
			},
			want: &ecs.TaskDefinition{
				Compatibilities: aws.StringSlice([]string{"FARGATE"}),
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Image: aws.String("nginx:alpine"),
						LogConfiguration: &ecs.LogConfiguration{
							LogDriver: aws.String("awslogs"),
							Options: map[string]*string{
								"awslogs-group":         aws.String("clu1"),
								"awslogs-stream-prefix": aws.String("datfam"),
								"awslogs-region":        aws.String("us-east-1"),
								"awslogs-create-group":  aws.String("true"),
							},
						},
						Name: aws.String("haxserver"),
						PortMappings: []*ecs.PortMapping{
							{ContainerPort: aws.Int64(80)},
							{ContainerPort: aws.Int64(443)},
						},
					},
				},
				Cpu:                     aws.String("256"),
				Family:                  aws.String("datfam"),
				Memory:                  aws.String("512"),
				ExecutionRoleArn:        aws.String("arn:aws:iam::12345678910:role/clu1-ecsTaskExecution"),
				NetworkMode:             aws.String("awsvpc"),
				RequiresAttributes:      []*ecs.Attribute{},
				RequiresCompatibilities: aws.StringSlice([]string{"FARGATE"}),
				Revision:                aws.Int64(1),
				Status:                  aws.String("ACTIVE"),
				TaskDefinitionArn:       aws.String("arn:aws:ecs:us-east-1:0123456789:task-definition/datfam:1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newMockOrchestrator(t, tt.fields.org,
				tt.fields.cwlerr, tt.fields.ecserr, tt.fields.iamerr,
				tt.fields.rgtaerr, tt.fields.smerr, tt.fields.sderr)
			got, _, err := o.processTaskTaskDefinitionCreate(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.processTaskTaskDefinitionCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Orchestrator.processTaskTaskDefinitionCreate() got = %v, want %v", got, tt.want)
			}

			// if !reflect.DeepEqual(got1, tt.want1) {
			// 	t.Errorf("Orchestrator.processTaskTaskDefinitionCreate() got1 = %v, want %v", got1, tt.want1)
			// }
		})
	}
}

func TestOrchestrator_processTaskDefinitionUpdate(t *testing.T) {
	t.Log("testing processTaskDefinitionUpdate")

	type fields struct {
		org     string
		cwlerr  error
		ecserr  error
		iamerr  error
		rgtaerr error
		smerr   error
		sderr   error
	}
	type args struct {
		ctx    context.Context
		input  *ServiceOrchestrationUpdateInput
		active *ServiceOrchestrationUpdateOutput
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantInput *ServiceOrchestrationUpdateInput
	}{
		{
			name: "nil input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "empty input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx:    context.TODO(),
				input:  &ServiceOrchestrationUpdateInput{},
				active: &ServiceOrchestrationUpdateOutput{},
			},
			wantErr: true,
		},
		{
			name: "example basic input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
				input: &ServiceOrchestrationUpdateInput{
					ClusterName: "clu1",
					TaskDefinition: &ecs.RegisterTaskDefinitionInput{
						ContainerDefinitions: []*ecs.ContainerDefinition{
							{
								Name:  aws.String("haxserver"),
								Image: aws.String("nginx:alpine"),
								PortMappings: []*ecs.PortMapping{
									{ContainerPort: aws.Int64(80)},
									{ContainerPort: aws.Int64(443)},
								},
							},
						},
						Cpu:    aws.String("256"),
						Family: aws.String("datfam"),
						Memory: aws.String("512"),
					},
					Tags: []*Tag{
						{
							Key:   aws.String("Application"),
							Value: aws.String("derpderpderp"),
						},
					},
					Service: &ecs.UpdateServiceInput{},
				},
				active: &ServiceOrchestrationUpdateOutput{},
			},
			wantInput: &ServiceOrchestrationUpdateInput{
				ClusterName: "clu1",
				TaskDefinition: &ecs.RegisterTaskDefinitionInput{
					ContainerDefinitions: []*ecs.ContainerDefinition{
						{
							Name:  aws.String("haxserver"),
							Image: aws.String("nginx:alpine"),
							PortMappings: []*ecs.PortMapping{
								{ContainerPort: aws.Int64(80)},
								{ContainerPort: aws.Int64(443)},
							},
							LogConfiguration: &ecs.LogConfiguration{
								LogDriver: aws.String("awslogs"),
								Options: map[string]*string{
									"awslogs-group":         aws.String("clu1"),
									"awslogs-stream-prefix": aws.String("datfam"),
									"awslogs-region":        aws.String("us-east-1"),
									"awslogs-create-group":  aws.String("true"),
								},
							},
						},
					},
					Cpu:                     aws.String("256"),
					ExecutionRoleArn:        aws.String("arn:aws:iam::12345678910:role/clu1-ecsTaskExecution"),
					Family:                  aws.String("datfam"),
					Memory:                  aws.String("512"),
					NetworkMode:             aws.String("awsvpc"),
					RequiresCompatibilities: aws.StringSlice([]string{"FARGATE"}),
				},
				Tags: []*Tag{
					{
						Key:   aws.String("Application"),
						Value: aws.String("derpderpderp"),
					},
				},
				Service: &ecs.UpdateServiceInput{
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:0123456789:task-definition/datfam:1"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newMockOrchestrator(t, tt.fields.org,
				tt.fields.cwlerr, tt.fields.ecserr, tt.fields.iamerr,
				tt.fields.rgtaerr, tt.fields.smerr, tt.fields.sderr)
			if err := o.processTaskDefinitionUpdate(tt.args.ctx, tt.args.input, tt.args.active); (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.processTaskDefinitionUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(tt.args.input, tt.wantInput) {
					t.Errorf("Orchestrator.processTaskDefinitionUpdate() input = %+v, wantInput %+v", tt.args.input, tt.wantInput)
				}
			}
		})
	}
}

func TestOrchestrator_defaultLogConfiguration(t *testing.T) {
	t.Log("testing defaultLogConfiguration")

	type fields struct {
		org     string
		cwlerr  error
		ecserr  error
		iamerr  error
		rgtaerr error
		smerr   error
		sderr   error
	}
	type args struct {
		ctx          context.Context
		logGroup     string
		streamPrefix string
		tags         []*Tag
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ecs.LogConfiguration
		wantErr bool
	}{
		{
			name: "empty inputs",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "empty inputs",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "empty log group",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx:          context.TODO(),
				streamPrefix: "myprefix",
				tags: []*Tag{
					{
						Key:   aws.String("key"),
						Value: aws.String("value"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "example input",
			fields: fields{
				org: "myorg",
			},
			args: args{
				ctx:          context.TODO(),
				logGroup:     "mygroup",
				streamPrefix: "myprefix",
				tags: []*Tag{
					{
						Key:   aws.String("key"),
						Value: aws.String("value"),
					},
				},
			},
			want: &ecs.LogConfiguration{
				LogDriver: aws.String("awslogs"),
				Options: map[string]*string{
					"awslogs-group":         aws.String("mygroup"),
					"awslogs-stream-prefix": aws.String("myprefix"),
					"awslogs-region":        aws.String("us-east-1"),
					"awslogs-create-group":  aws.String("true"),
				},
			},
		},
		{
			name: "example input, log group exists",
			fields: fields{
				org:    "myorg",
				cwlerr: awserr.New(cloudwatchlogs.ErrCodeResourceAlreadyExistsException, "exists", errors.New("exists")),
			},
			args: args{
				ctx:          context.TODO(),
				logGroup:     "mygroup",
				streamPrefix: "myprefix",
				tags: []*Tag{
					{
						Key:   aws.String("key"),
						Value: aws.String("value"),
					},
				},
			},
			want: &ecs.LogConfiguration{
				LogDriver: aws.String("awslogs"),
				Options: map[string]*string{
					"awslogs-group":         aws.String("mygroup"),
					"awslogs-stream-prefix": aws.String("myprefix"),
					"awslogs-region":        aws.String("us-east-1"),
					"awslogs-create-group":  aws.String("true"),
				},
			},
		},
		{
			name: "example input, other error",
			fields: fields{
				org:    "myorg",
				cwlerr: errors.New("boom"),
			},
			args: args{
				ctx:          context.TODO(),
				logGroup:     "mygroup",
				streamPrefix: "myprefix",
				tags: []*Tag{
					{
						Key:   aws.String("key"),
						Value: aws.String("value"),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newMockOrchestrator(t, tt.fields.org,
				tt.fields.cwlerr, tt.fields.ecserr, tt.fields.iamerr,
				tt.fields.rgtaerr, tt.fields.smerr, tt.fields.sderr)
			got, err := o.defaultLogConfiguration(tt.args.ctx, tt.args.logGroup, tt.args.streamPrefix, tt.args.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.defaultLogConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Orchestrator.defaultLogConfiguration() = %v, want %v", got, tt.want)
			}
		})
	}
}
