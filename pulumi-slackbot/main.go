package main

import (
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get configuration
		cfg := config.New(ctx, "slackbot")
		slackBotToken := cfg.RequireSecret("slackBotToken")
		slackSigningSecret := cfg.RequireSecret("slackSigningSecret")
		claudeApiKey := cfg.RequireSecret("claudeApiKey")
		s3BucketName := cfg.Get("s3Bucket")
		if s3BucketName == "" {
			s3BucketName = "slackbot-claude-sessions"
		}
		workDirectory := cfg.Get("workDirectory")
		if workDirectory == "" {
			workDirectory = "/tmp/claude-sessions"
		}

		// Create S3 bucket for Claude session uploads
		bucket, err := s3.NewBucket(ctx, "claude-sessions-bucket", &s3.BucketArgs{
			Bucket: pulumi.String(s3BucketName),
			Versioning: &s3.BucketVersioningArgs{
				Enabled: pulumi.Bool(true),
			},
		})
		if err != nil {
			return err
		}

		// Create DynamoDB table for session storage
		sessionsTable, err := dynamodb.NewTable(ctx, "slackbot-sessions", &dynamodb.TableArgs{
			Name:           pulumi.String("slackbot-sessions"),
			BillingMode:    pulumi.String("PAY_PER_REQUEST"),
			HashKey:        pulumi.String("sessionId"),
			RangeKey:       pulumi.String("threadId"),
			StreamEnabled:  pulumi.Bool(false),
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("sessionId"),
					Type: pulumi.String("S"),
				},
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("threadId"),
					Type: pulumi.String("S"),
				},
			},
			Tags: pulumi.StringMap{
				"Environment": pulumi.String("production"),
				"Application": pulumi.String("slackbot"),
			},
		})
		if err != nil {
			return err
		}

		// Create IAM role for Lambda execution
		lambdaRole, err := iam.NewRole(ctx, "slackbot-lambda-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Action": "sts:AssumeRole",
						"Principal": {
							"Service": "lambda.amazonaws.com"
						},
						"Effect": "Allow",
						"Sid": ""
					}
				]
			}`),
		})
		if err != nil {
			return err
		}

		// Attach basic Lambda execution policy
		_, err = iam.NewRolePolicyAttachment(ctx, "slackbot-lambda-basic-execution", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		})
		if err != nil {
			return err
		}

		// Create IAM policy for DynamoDB and S3 access
		lambdaPolicy, err := iam.NewPolicy(ctx, "slackbot-lambda-policy", &iam.PolicyArgs{
			Description: pulumi.String("IAM policy for Slackbot Lambda function"),
			Policy: pulumi.All(sessionsTable.Arn, bucket.Arn).ApplyT(func(args []interface{}) (string, error) {
				tableArn := args[0].(string)
				bucketArn := args[1].(string)
				policy := map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []interface{}{
						map[string]interface{}{
							"Effect": "Allow",
							"Action": []string{
								"dynamodb:GetItem",
								"dynamodb:PutItem",
								"dynamodb:UpdateItem",
								"dynamodb:DeleteItem",
								"dynamodb:Query",
								"dynamodb:Scan",
							},
							"Resource": []string{
								tableArn,
								fmt.Sprintf("%s/*", tableArn),
							},
						},
						map[string]interface{}{
							"Effect": "Allow",
							"Action": []string{
								"s3:GetObject",
								"s3:PutObject",
								"s3:DeleteObject",
								"s3:ListBucket",
							},
							"Resource": []string{
								bucketArn,
								fmt.Sprintf("%s/*", bucketArn),
							},
						},
					},
				}
				policyJSON, err := json.Marshal(policy)
				if err != nil {
					return "", err
				}
				return string(policyJSON), nil
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}

		// Attach the policy to the role
		_, err = iam.NewRolePolicyAttachment(ctx, "slackbot-lambda-policy-attachment", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: lambdaPolicy.Arn,
		})
		if err != nil {
			return err
		}

		// Create Lambda function for Slackbot
		slackbotLambda, err := lambda.NewFunction(ctx, "slackbot-lambda", &lambda.FunctionArgs{
			Runtime:      pulumi.String("go1.x"),
			Code:         pulumi.NewFileArchive("./lambda/slackbot-lambda.zip"),
			Handler:      pulumi.String("main"),
			Role:         lambdaRole.Arn,
			Timeout:      pulumi.Int(30),
			MemorySize:   pulumi.Int(256),
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"SLACK_BOT_TOKEN":      slackBotToken,
					"SLACK_SIGNING_SECRET": slackSigningSecret,
					"CLAUDE_API_KEY":       claudeApiKey,
					"DYNAMODB_TABLE":       sessionsTable.Name,
					"S3_BUCKET":            bucket.Bucket,
					"WORK_DIRECTORY":       pulumi.String(workDirectory),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create Lambda function for Claude sessions
		claudeSessionLambda, err := lambda.NewFunction(ctx, "claude-session-lambda", &lambda.FunctionArgs{
			Runtime:      pulumi.String("go1.x"),
			Code:         pulumi.NewFileArchive("./lambda/claude-session-lambda.zip"),
			Handler:      pulumi.String("main"),
			Role:         lambdaRole.Arn,
			Timeout:      pulumi.Int(900), // 15 minutes max for Claude sessions
			MemorySize:   pulumi.Int(512),
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"CLAUDE_API_KEY":   claudeApiKey,
					"DYNAMODB_TABLE":   sessionsTable.Name,
					"S3_BUCKET":        bucket.Bucket,
					"WORK_DIRECTORY":   pulumi.String(workDirectory),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create API Gateway for Slack events
		slackApi, err := apigateway.NewRestApi(ctx, "slackbot-api", &apigateway.RestApiArgs{
			Name:        pulumi.String("slackbot-api"),
			Description: pulumi.String("API Gateway for Slack Events"),
		})
		if err != nil {
			return err
		}

		// Create API Gateway resource for Slack events
		slackResource, err := apigateway.NewResource(ctx, "slack-events-resource", &apigateway.ResourceArgs{
			RestApi:   slackApi.ID(),
			ParentId:  slackApi.RootResourceId,
			PathPart:  pulumi.String("slack"),
		})
		if err != nil {
			return err
		}

		// Create API Gateway method for Slack events
		slackMethod, err := apigateway.NewMethod(ctx, "slack-events-method", &apigateway.MethodArgs{
			RestApi:       slackApi.ID(),
			ResourceId:    slackResource.ID(),
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
		})
		if err != nil {
			return err
		}

		// Create API Gateway integration for Slack events
		_, err = apigateway.NewIntegration(ctx, "slack-events-integration", &apigateway.IntegrationArgs{
			RestApi:               slackApi.ID(),
			ResourceId:            slackResource.ID(),
			HttpMethod:            slackMethod.HttpMethod,
			IntegrationHttpMethod: pulumi.String("POST"),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   slackbotLambda.InvokeArn,
		})
		if err != nil {
			return err
		}

		// Create Lambda permission for API Gateway
		_, err = lambda.NewPermission(ctx, "slackbot-lambda-permission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  slackbotLambda.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("%s/*/*", slackApi.ExecutionArn),
		})
		if err != nil {
			return err
		}

		// Deploy API Gateway
		deployment, err := apigateway.NewDeployment(ctx, "slackbot-deployment", &apigateway.DeploymentArgs{
			RestApi: slackApi.ID(),
			StageName: pulumi.String("prod"),
		}, pulumi.DependsOn([]pulumi.Resource{slackMethod}))
		if err != nil {
			return err
		}

		// Create WebSocket API Gateway for Claude sessions
		websocketApi, err := apigatewayv2.NewApi(ctx, "claude-websocket-api", &apigatewayv2.ApiArgs{
			Name:                       pulumi.String("claude-websocket-api"),
			Description:                pulumi.String("WebSocket API for Claude sessions"),
			ProtocolType:               pulumi.String("WEBSOCKET"),
			RouteSelectionExpression:   pulumi.String("$request.body.action"),
		})
		if err != nil {
			return err
		}

		// Create WebSocket routes
		connectRoute, err := apigatewayv2.NewRoute(ctx, "claude-websocket-connect", &apigatewayv2.RouteArgs{
			ApiId:    websocketApi.ID(),
			RouteKey: pulumi.String("$connect"),
			Target:   pulumi.Sprintf("integrations/%s", "connect-integration"),
		})
		if err != nil {
			return err
		}

		disconnectRoute, err := apigatewayv2.NewRoute(ctx, "claude-websocket-disconnect", &apigatewayv2.RouteArgs{
			ApiId:    websocketApi.ID(),
			RouteKey: pulumi.String("$disconnect"),
			Target:   pulumi.Sprintf("integrations/%s", "disconnect-integration"),
		})
		if err != nil {
			return err
		}

		defaultRoute, err := apigatewayv2.NewRoute(ctx, "claude-websocket-default", &apigatewayv2.RouteArgs{
			ApiId:    websocketApi.ID(),
			RouteKey: pulumi.String("$default"),
			Target:   pulumi.Sprintf("integrations/%s", "default-integration"),
		})
		if err != nil {
			return err
		}

		// Create WebSocket integrations
		_, err = apigatewayv2.NewIntegration(ctx, "claude-websocket-connect-integration", &apigatewayv2.IntegrationArgs{
			ApiId:             websocketApi.ID(),
			IntegrationType:   pulumi.String("AWS_PROXY"),
			IntegrationUri:    claudeSessionLambda.InvokeArn,
			IntegrationMethod: pulumi.String("POST"),
		})
		if err != nil {
			return err
		}

		_, err = apigatewayv2.NewIntegration(ctx, "claude-websocket-disconnect-integration", &apigatewayv2.IntegrationArgs{
			ApiId:             websocketApi.ID(),
			IntegrationType:   pulumi.String("AWS_PROXY"),
			IntegrationUri:    claudeSessionLambda.InvokeArn,
			IntegrationMethod: pulumi.String("POST"),
		})
		if err != nil {
			return err
		}

		_, err = apigatewayv2.NewIntegration(ctx, "claude-websocket-default-integration", &apigatewayv2.IntegrationArgs{
			ApiId:             websocketApi.ID(),
			IntegrationType:   pulumi.String("AWS_PROXY"),
			IntegrationUri:    claudeSessionLambda.InvokeArn,
			IntegrationMethod: pulumi.String("POST"),
		})
		if err != nil {
			return err
		}

		// Create WebSocket deployment
		websocketDeployment, err := apigatewayv2.NewDeployment(ctx, "claude-websocket-deployment", &apigatewayv2.DeploymentArgs{
			ApiId: websocketApi.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{connectRoute, disconnectRoute, defaultRoute}))
		if err != nil {
			return err
		}

		// Create WebSocket stage
		websocketStage, err := apigatewayv2.NewStage(ctx, "claude-websocket-stage", &apigatewayv2.StageArgs{
			ApiId:        websocketApi.ID(),
			DeploymentId: websocketDeployment.ID(),
			Name:         pulumi.String("prod"),
		})
		if err != nil {
			return err
		}

		// Create Lambda permissions for WebSocket API
		_, err = lambda.NewPermission(ctx, "claude-websocket-lambda-permission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  claudeSessionLambda.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("%s/*/*", websocketApi.ExecutionArn),
		})
		if err != nil {
			return err
		}

		// Export important values
		ctx.Export("slackApiUrl", pulumi.Sprintf("https://%s.execute-api.%s.amazonaws.com/prod/slack", slackApi.ID(), aws.Region))
		ctx.Export("websocketApiUrl", pulumi.Sprintf("wss://%s.execute-api.%s.amazonaws.com/prod", websocketApi.ID(), aws.Region))
		ctx.Export("s3BucketName", bucket.Bucket)
		ctx.Export("dynamodbTableName", sessionsTable.Name)
		ctx.Export("slackbotLambdaArn", slackbotLambda.Arn)
		ctx.Export("claudeSessionLambdaArn", claudeSessionLambda.Arn)

		return nil
	})
}