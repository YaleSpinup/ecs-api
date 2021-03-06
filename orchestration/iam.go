package orchestration

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/YaleSpinup/apierror"
	im "github.com/YaleSpinup/ecs-api/iam"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

var assumeRolePolicyDoc []byte

// DefaultTaskExecutionPolicy generates the default policy for ECS task execution
func (o *Orchestrator) DefaultTaskExecutionPolicy(path string) im.PolicyDoc {
	log.Debugf("generating default task execution policy for %s", path)

	return im.PolicyDoc{
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
					fmt.Sprintf("arn:aws:kms:*:*:key/%s", o.IAM.DefaultKmsKeyID),
				},
			},
		},
	}
}

// DefaultTaskExecutionRole generates the default role (if it doesn't exist) for ECS task execution and returns the ARN
func (o *Orchestrator) DefaultTaskExecutionRole(ctx context.Context, path, role string, tags []*Tag) (string, error) {
	if path == "" || role == "" {
		return "", apierror.New(apierror.ErrBadRequest, "invalid path", nil)
	}

	log.Infof("generating default task execution role %s/%s", path, role)

	defaultPolicy := o.DefaultTaskExecutionPolicy(path)

	var roleArn string
	if out, err := o.IAM.GetRole(ctx, role); err != nil {
		if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrNotFound {
			return "", err
		}

		log.Debugf("unable to find role %s, creating", role)

		output, err := o.createDefaultTaskExecutionRole(ctx, path, role)
		if err != nil {
			return "", err
		}

		roleArn = output

		log.Infof("created role %s with ARN: %s", role, roleArn)
	} else {
		roleArn = aws.StringValue(out.Arn)

		log.Infof("role %s exists with ARN: %s", role, roleArn)

		currentDoc, err := o.IAM.GetRolePolicy(ctx, role, "ECSTaskAccessPolicy")
		if err != nil {
			if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrNotFound {
				return "", err
			}

			log.Infof("inline policy for role %s is not found, updating", role)

		} else {
			currentPolicy, err := im.PolicyFromDocument(currentDoc)
			if err != nil {
				return "", err
			}

			// if the current policy matches the generated (default) policy, return
			// the role ARN otherwise, keep going and update the policy doc
			if awsutil.DeepEqual(defaultPolicy, currentPolicy) {
				log.Debugf("inline policy for role %s is up to date", role)
				return roleArn, nil
			}

			log.Infof("inline policy for role %s is out of date, updating", role)
		}

	}

	defaultPolicyDoc, err := im.DocumentFromPolicy(&defaultPolicy)
	if err != nil {
		log.Errorf("failed creating default IAM task execution policy for %s: %s", path, err.Error())
		return "", err
	}

	// attach default role policy to the role
	err = o.IAM.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(defaultPolicyDoc),
		PolicyName:     aws.String("ECSTaskAccessPolicy"),
		RoleName:       aws.String(role),
	})
	if err != nil {
		return "", err
	}

	// apply tags if any were passed
	if len(tags) > 0 {
		iamTags := make([]*iam.Tag, len(tags))
		for i, t := range tags {
			iamTags[i] = &iam.Tag{Key: t.Key, Value: t.Value}
		}

		if err := o.IAM.TagRole(ctx, role, iamTags); err != nil {
			return "", err
		}
	}

	return roleArn, nil
}

// createDefaultTaskExecutionRole handles creating the default task execution role.  it does not leverage the
// path for the role currently since we already have many container services with the "/" path.
// TODO: revisit moving to a non-default path for the task execution role
func (o *Orchestrator) createDefaultTaskExecutionRole(ctx context.Context, path, role string) (string, error) {
	if role == "" {
		return "", apierror.New(apierror.ErrBadRequest, "invalid role", nil)
	}

	log.Debugf("creating default task execution role %s", role)

	assumeRolePolicyDoc, err := assumeRolePolicy()
	if err != nil {
		log.Errorf("failed to generate default task execution role assume policy for %s: %s", path, err)
		return "", err
	}

	log.Debugf("generated assume role policy document: %s", assumeRolePolicyDoc)

	roleOutput, err := o.IAM.CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicyDoc),
		Description:              aws.String(fmt.Sprintf("ECS task execution role for %s", path)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String(role),
	})
	if err != nil {
		return "", err
	}

	return aws.StringValue(roleOutput.Arn), nil
}

// assumeRolePolicy generates the policy document to allow the ecs service to assume a role
func assumeRolePolicy() (string, error) {
	if assumeRolePolicyDoc != nil {
		return string(assumeRolePolicyDoc), nil
	}

	policyDoc, err := json.Marshal(im.PolicyDoc{
		Version: "2012-10-17",
		Statement: []im.PolicyStatement{
			{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: map[string][]string{
					"Service": {"ecs-tasks.amazonaws.com"},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	// cache result since it doesn't change
	assumeRolePolicyDoc = policyDoc

	return string(policyDoc), nil
}

func (o *Orchestrator) deleteDefaultTaskExecutionRole(ctx context.Context, role string) error {
	policies, err := o.IAM.ListRolePolicies(ctx, role)
	if err != nil {
		return err
	}

	for _, p := range policies {
		if err := o.IAM.DeleteRolePolicy(ctx, role, p); err != nil {
			return err
		}
	}

	if err := o.IAM.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(role),
	}); err != nil {
		return err
	}

	return nil
}
