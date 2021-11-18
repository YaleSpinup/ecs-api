package orchestration

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/YaleSpinup/apierror"
	yiam "github.com/YaleSpinup/aws-go/services/iam"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

var assumeRolePolicyDoc []byte

// defaultTaskExecutionPolicy generates the default policy for ECS task execution
func defaultTaskExecutionPolicy(path, kms string) yiam.PolicyDocument {
	log.Debugf("generating default task execution policy for %s", path)

	return yiam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []yiam.StatementEntry{
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
					fmt.Sprintf("arn:aws:kms:*:*:key/%s", kms),
				},
			},
			{
				Effect: "Allow",
				Action: []string{
					"elasticfilesystem:ClientRootAccess",
					"elasticfilesystem:ClientWrite",
					"elasticfilesystem:ClientMount",
				},
				Resource: []string{"*"},
				Condition: yiam.Condition{
					"Bool": yiam.ConditionStatement{
						"elasticfilesystem:AccessedViaMountTarget": []string{"true"},
					},
					"StringEqualsIgnoreCase": yiam.ConditionStatement{
						"aws:ResourceTag/spinup:org": []string{
							"${aws:PrincipalTag/spinup:org}",
						},
						"aws:ResourceTag/spinup:spaceid": []string{
							"${aws:PrincipalTag/spinup:spaceid}",
						},
					},
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

	log.Infof("generating default task execution role %s/%s if it doesn't exist ", path, role)

	defaultPolicy := defaultTaskExecutionPolicy(path, o.IAM.DefaultKmsKeyID)

	var roleArn string
	if out, err := o.IAM.GetRole(ctx, role); err != nil {
		if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrNotFound {
			return "", err
		}

		log.Debugf("unable to find role %s/%s, creating", path, role)

		output, err := o.createDefaultTaskExecutionRole(ctx, path, role)
		if err != nil {
			return "", err
		}

		roleArn = output

		log.Infof("created role %s/%s with ARN: %s", path, role, roleArn)
	} else {
		roleArn = aws.StringValue(out.Arn)

		log.Infof("role %s exists with ARN: %s", role, roleArn)

		currentDoc, err := o.IAM.GetRolePolicy(ctx, role, "ECSTaskAccessPolicy")
		if err != nil {
			if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrNotFound {
				return "", err
			}

			log.Infof("inline policy for role %s/%s is not found, updating", path, role)

		} else {
			var currentPolicy yiam.PolicyDocument
			if err := json.Unmarshal([]byte(currentDoc), &currentPolicy); err != nil {
				log.Errorf("failed to unmarhsall policy from document: %s", err)
				return "", err
			}

			// if the current policy matches the generated (default) policy, return
			// the role ARN otherwise, keep going and update the policy doc
			if yiam.PolicyDeepEqual(defaultPolicy, currentPolicy) {
				log.Debugf("inline policy for role %s/%s is up to date", path, role)
				return roleArn, nil
			}

			log.Infof("inline policy for role %s/%s is out of date, updating", path, role)
		}

	}

	defaultPolicyDoc, err := json.Marshal(defaultPolicy)
	if err != nil {
		log.Errorf("failed creating default IAM task execution policy for %s: %s", path, err.Error())
		return "", err
	}

	// attach default role policy to the role
	err = o.IAM.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(defaultPolicyDoc)),
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

	policyDoc, err := json.Marshal(yiam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []yiam.StatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: yiam.Principal{
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
