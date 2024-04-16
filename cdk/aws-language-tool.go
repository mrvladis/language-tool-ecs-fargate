package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	appautoscaling "github.com/aws/aws-cdk-go/awscdk/v2/awsapplicationautoscaling"
	acm "github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	ecs "github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	elbv2 "github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	iam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	logs "github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	r53 "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	r53targets "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"

	// ecr "github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	ecrassets "github.com/aws/aws-cdk-go/awscdk/v2/awsecrassets"
	kms "github.com/aws/aws-cdk-go/awscdk/v2/awskms"
	s3 "github.com/aws/aws-cdk-go/awscdk/v2/awss3"

	"fmt"
	"os"

	// "strings"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type AwsLanguageToolStackProps struct {
	awscdk.StackProps
}

var hostedZoneID, hostedZoneName = os.Getenv("hostedZoneID"), os.Getenv("hostedZoneName")
var langToolDomainName = os.Getenv("langToolDomainName")
var langToolAPIName = os.Getenv("langToolAPIName")

// var langToolWebName = os.Getenv("langToolWebName")

// var oidClientSecret = awscdk.SecretValue_SecretsManager(jsii.String("LangToolsClientSecret"), &awscdk.SecretsManagerSecretOptions{})

func NewAwsLanguageToolStack(scope constructs.Construct, id *string, props *AwsLanguageToolStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, id, &sprops)

	// Create VPC for the Language Tool
	myVPC := ec2.NewVpc(stack, id, &ec2.VpcProps{
		//		AvailabilityZones:            &[]*string{},
		IpAddresses:           ec2.IpAddresses_Cidr(jsii.String("10.128.0.0/16")),
		CreateInternetGateway: jsii.Bool(true),
		EnableDnsHostnames:    jsii.Bool(true),
		EnableDnsSupport:      jsii.Bool(true),
		// This need to be investigated. I am not sure why doesn't this work.
		// FlowLogs:                     &map[string]*ec2.FlowLogOptions{
		// 	Destination: ec2.FlowLogDestination_ToCloudWatchLogs(&logs.LogGroupProps{}),
		// },

		// Destination: ec2.FlowLogDestination_ToCloudWatchLogs(&logs.LogGroupProps{
		// 	Retention: logs.RetentionDays_SIX_MONTHS,
		// 	RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		// 	}),

		// GatewayEndpoints:             &map[string]*ec2.GatewayVpcEndpointOptions{},
		// IpAddresses:                  nil,
		MaxAzs: jsii.Number(2),
		// NatGatewayProvider:           nil,
		NatGateways: jsii.Number(2),
		NatGatewaySubnets: &ec2.SubnetSelection{
			OnePerAz:   jsii.Bool(true),
			SubnetType: ec2.SubnetType_PUBLIC,
		},
		// ReservedAzs:                  new(float64),
		// RestrictDefaultSecurityGroup: new(bool),
		SubnetConfiguration: &[]*ec2.SubnetConfiguration{
			{
				Name:       jsii.String("Public"),
				SubnetType: ec2.SubnetType_PUBLIC,
				CidrMask:   jsii.Number(24),
			},
			{
				Name:       jsii.String("Private"),
				SubnetType: ec2.SubnetType_PRIVATE_WITH_EGRESS,
				CidrMask:   jsii.Number(24),
			},
			{
				Name:       jsii.String("PrivateIntra"),
				SubnetType: ec2.SubnetType_PRIVATE_ISOLATED,
				CidrMask:   jsii.Number(24),
			},
		},
		//VpcName:                      id,
		// VpnConnections:               &map[string]*ec2.VpnConnectionOptions{},
		// VpnGateway:                   new(bool),
		// VpnGatewayAsn:                new(float64),
		// VpnRoutePropagation:          &[]*ec2.SubnetSelection{},
	})

	// VPC Flow Logs Configuration
	myVpcRole := iam.NewRole(stack, jsii.String("vpcFlowLogsRole"+*id), &iam.RoleProps{
		AssumedBy: iam.NewServicePrincipal(jsii.String("vpc-flow-logs.amazonaws.com"), &iam.ServicePrincipalOpts{}),
		ManagedPolicies: &[]iam.IManagedPolicy{
			iam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("CloudWatchFullAccess")),
		},
	})

	myVPCLogGroup := logs.NewLogGroup(stack, jsii.String("vpcFlowLogGroup"), &logs.LogGroupProps{
		Retention:     logs.RetentionDays_SIX_MONTHS,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	myVPCLogStream := logs.NewLogStream(stack, jsii.String(*id+"vpcFlowLogStream"), &logs.LogStreamProps{
		LogGroup:      myVPCLogGroup,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	myVPCFlowLogs := ec2.NewFlowLog(stack, jsii.String(*id+"vpcFlowLogs"), &ec2.FlowLogProps{
		Destination:            ec2.FlowLogDestination_ToCloudWatchLogs(myVPCLogGroup, myVpcRole),
		ResourceType:           ec2.FlowLogResourceType_FromVpc(myVPC),
		TrafficType:            ec2.FlowLogTrafficType_ALL,
		MaxAggregationInterval: ec2.FlowLogMaxAggregationInterval_ONE_MINUTE,
	})

	awscdk.NewCfnOutput(stack, jsii.String("vpcId"), &awscdk.CfnOutputProps{
		Value: myVPC.VpcId(),
	})

	fmt.Println(myVPCLogStream, myVPCFlowLogs)

	// Create Certificate

	myHostedZone := r53.HostedZone_FromHostedZoneAttributes(stack, jsii.String("awesomebuilder"), &r53.HostedZoneAttributes{
		ZoneName:     jsii.String(hostedZoneName),
		HostedZoneId: jsii.String(hostedZoneID),
	})

	myALBCert := acm.NewCertificate(stack, jsii.String(*id+"_ALTCert"), &acm.CertificateProps{
		DomainName:      jsii.String(langToolDomainName),
		CertificateName: jsii.String(langToolDomainName),
		SubjectAlternativeNames: &[]*string{
			jsii.String(langToolAPIName + "." + langToolDomainName),
			// jsii.String(langToolWebName + "." + langToolDomainName),
		},
		Validation: acm.CertificateValidation_FromDns(
			myHostedZone,
		),
	})

	// Create Load Balancer for the Language Tool

	myALBsg := ec2.NewSecurityGroup(stack, jsii.String(*id+"_ALBSG"), &ec2.SecurityGroupProps{
		Vpc:               myVPC,
		AllowAllOutbound:  jsii.Bool(true),
		SecurityGroupName: id,
	})
	myALBsg.AddIngressRule(ec2.Peer_AnyIpv4(), ec2.Port_Tcp(jsii.Number(443)), jsii.String("Allow HTTPS Traffic"), jsii.Bool(false))

	myALB := elbv2.NewApplicationLoadBalancer(stack, jsii.String(*id+"_ALB"), &elbv2.ApplicationLoadBalancerProps{
		LoadBalancerName:   id,
		DeletionProtection: jsii.Bool(false),
		Vpc:                myVPC,
		InternetFacing:     jsii.Bool(true),
		VpcSubnets: &ec2.SubnetSelection{
			SubnetType: ec2.SubnetType_PUBLIC,
		},
		IpAddressType: elbv2.IpAddressType_IPV4,
		SecurityGroup: myALBsg,
	})

	myHTTPListener := myALB.AddRedirect(&elbv2.ApplicationLoadBalancerRedirectConfig{
		SourcePort:     jsii.Number(80),
		SourceProtocol: elbv2.ApplicationProtocol_HTTP,
		TargetPort:     jsii.Number(443),
		TargetProtocol: elbv2.ApplicationProtocol_HTTPS,
	})

	myELBTargetGroup := elbv2.NewApplicationTargetGroup(stack, jsii.String(*id+"ECSTargetGroup"), &elbv2.ApplicationTargetGroupProps{
		TargetGroupName: id,
		Vpc:             myVPC,
		Protocol:        elbv2.ApplicationProtocol_HTTP,
		TargetType:      elbv2.TargetType_IP,
		HealthCheck: &elbv2.HealthCheck{
			Enabled:                 jsii.Bool(true),
			Path:                    jsii.String("/v2/languages"),
			Interval:                awscdk.Duration_Seconds(jsii.Number(10)),
			Timeout:                 awscdk.Duration_Seconds(jsii.Number(5)),
			HealthyThresholdCount:   jsii.Number(2),
			UnhealthyThresholdCount: jsii.Number(2),
			HealthyHttpCodes:        jsii.String("200"),
			// Port: jsii.String("80"),
			Protocol: elbv2.Protocol_HTTP,
		},
	})

	myHTTPSListener := myALB.AddListener(jsii.String(*id+"_HTTPSListener"), &elbv2.BaseApplicationListenerProps{
		Port:     jsii.Number(443),
		Protocol: elbv2.ApplicationProtocol_HTTPS,
		Open:     jsii.Bool(true),
		Certificates: &[]elbv2.IListenerCertificate{
			elbv2.ListenerCertificate_FromCertificateManager(myALBCert),
		},
		SslPolicy: elbv2.SslPolicy_RECOMMENDED_TLS,
		DefaultAction: elbv2.ListenerAction_Forward(&[]elbv2.IApplicationTargetGroup{
			myELBTargetGroup,
		},
			&elbv2.ForwardOptions{
				StickinessDuration: awscdk.Duration_Minutes(jsii.Number(60)),
			},
		),
		// DefaultAction: elbv2.ListenerAction_AuthenticateOidc(
		// 	&elbv2.AuthenticateOidcOptions{
		// 		AuthorizationEndpoint: jsii.String("https://login.microsoftonline.com/a4758b3d-fddb-4a52-afbd-dae8ec30dd1e/oauth2/v2.0/authorize"),
		// 		ClientId:              jsii.String("e0b1e54a-69b3-4e0b-8795-7810bfc2be91"),
		// 		ClientSecret:          oidClientSecret,
		// 		Issuer:                jsii.String("https://login.microsoftonline.com/a4758b3d-fddb-4a52-afbd-dae8ec30dd1e/wsfed"),
		// 		TokenEndpoint:         jsii.String("https://login.microsoftonline.com/a4758b3d-fddb-4a52-afbd-dae8ec30dd1e/oauth2/v2.0/token"),
		// 		UserInfoEndpoint:      jsii.String("https://login.microsoftonline.com/a4758b3d-fddb-4a52-afbd-dae8ec30dd1e/v2.0/.well-known/openid-configuration"),
		// 		// Scope: jsii.String("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"),
		// 		// SessionCookieName: jsii.String("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"),
		// 		Next: elbv2.ListenerAction_Forward(
		// 			&[]elbv2.IApplicationTargetGroup{
		// 				myELBTargetGroup,
		// 			},
		// 			&elbv2.ForwardOptions{
		// 				StickinessDuration: awscdk.Duration_Minutes(jsii.Number(60)),
		// 			},
		// 		),
		// 	},
		// ),
	})

	// Protocol: elbv2.ApplicationProtocol_HTTPS,
	// 	Port:     jsii.String("443"),
	// 	StatusCode: jsii.String("HTTP_301"),
	// 	Host: jsii.String("#{host}"),
	// 	Path: jsii.String("/#{path}"),
	// 	Query: jsii.String("#{query}"),

	fmt.Println(myHTTPListener, myHTTPSListener)

	// //Create ECR registry if doesn't exist
	// 	myECRRepo := ecr.NewRepository(stack, jsii.String(strings.ToLower(*id+"ECR")), &ecr.RepositoryProps{
	// 		RepositoryName: jsii.String(strings.ToLower(*id+"ECR")),
	// 		ImageScanOnPush: jsii.Bool(true),
	// 	})

	// Create ECS Exec KMS Key and S3Bucket and logging
	myKMSKey := kms.NewKey(stack, jsii.String(*id+"ECSExecKey"), &kms.KeyProps{
		Description:       jsii.String(*id + "ECSExecKey"),
		EnableKeyRotation: jsii.Bool(true),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
	})
	myECSExecBucket := s3.NewBucket(stack, jsii.String(*id+"ecsexeclogs"), &s3.BucketProps{
		BlockPublicAccess: s3.BlockPublicAccess_BLOCK_ALL(),
		//		BucketName: jsii.String(strings.ToLower(*id+"ecsexeclogs")),
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		Versioned:     jsii.Bool(true),
		EncryptionKey: myKMSKey,
	})

	myECSLogGroup := logs.NewLogGroup(stack, jsii.String("ECSLogGroup"), &logs.LogGroupProps{
		Retention:     logs.RetentionDays_SIX_MONTHS,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	// Create ECS Service for the Language Tool
	myECSCluster := ecs.NewCluster(stack, jsii.String(*id+"ECSCluster"), &ecs.ClusterProps{
		ClusterName:       id,
		Vpc:               myVPC,
		ContainerInsights: jsii.Bool(true),
		ExecuteCommandConfiguration: &ecs.ExecuteCommandConfiguration{
			KmsKey: myKMSKey,
			LogConfiguration: &ecs.ExecuteCommandLogConfiguration{
				S3Bucket:                    myECSExecBucket,
				S3EncryptionEnabled:         jsii.Bool(true),
				S3KeyPrefix:                 jsii.String("exec-command-output"),
				CloudWatchLogGroup:          myECSLogGroup,
				CloudWatchEncryptionEnabled: jsii.Bool(true),
			},
			Logging: ecs.ExecuteCommandLogging_OVERRIDE,
		},
	})

	fmt.Println(myECSCluster)

	// Create ECS Execution Role
	myECSTangToolRole := iam.NewRole(stack, jsii.String(*id+"ECSExecutionRole"), &iam.RoleProps{
		AssumedBy: iam.NewServicePrincipal(jsii.String("ecs-tasks.amazonaws.com"), &iam.ServicePrincipalOpts{}),
		ManagedPolicies: &[]iam.IManagedPolicy{
			iam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AmazonECSTaskExecutionRolePolicy")),
		},
	})

	myECSExecPolicy := iam.NewManagedPolicy(stack, jsii.String(*id+"ECSExecPolicy"), &iam.ManagedPolicyProps{
		Description:       jsii.String(*id + "ECSExecPolicy"),
		ManagedPolicyName: jsii.String(*id + "ECSExecPolicy"),
		Document: iam.NewPolicyDocument(&iam.PolicyDocumentProps{
			Statements: &[]iam.PolicyStatement{
				iam.NewPolicyStatement(&iam.PolicyStatementProps{
					Effect: iam.Effect_ALLOW,
					Actions: &[]*string{
						jsii.String("ssmmessages:CreateControlChannel"),
						jsii.String("ssmmessages:CreateDataChannel"),
						jsii.String("ssmmessages:OpenControlChannel"),
						jsii.String("ssmmessages:OpenDataChannel"),
					},
					Resources: &[]*string{
						jsii.String("*"),
					},
				}),
				iam.NewPolicyStatement(&iam.PolicyStatementProps{
					Effect: iam.Effect_ALLOW,
					Actions: &[]*string{
						jsii.String("s3:PutObject"),
						jsii.String("s3:GetEncryptionConfiguration"),
					},
					Resources: &[]*string{
						myECSExecBucket.BucketArn(),
						jsii.String(*myECSExecBucket.BucketArn() + "/*"),
					},
				}),
				iam.NewPolicyStatement(&iam.PolicyStatementProps{
					Effect: iam.Effect_ALLOW,
					Actions: &[]*string{
						jsii.String("kms:Decrypt"),
					},
					Resources: &[]*string{
						myKMSKey.KeyArn(),
					},
				}),
			},
		}),
	})

	myECSTangToolRole.AddManagedPolicy(myECSExecPolicy)

	//Create Task Definition of the LangTool Container
	myECSTaskDefinition := ecs.NewTaskDefinition(stack, jsii.String(*id+"ECSTaskDefinition"), &ecs.TaskDefinitionProps{
		Compatibility: ecs.Compatibility_FARGATE,
		Cpu:           jsii.String("512"),
		MemoryMiB:     jsii.String("1024"),
		NetworkMode:   ecs.NetworkMode_AWS_VPC,
		TaskRole:      myECSTangToolRole,
		RuntimePlatform: &ecs.RuntimePlatform{
			OperatingSystemFamily: ecs.OperatingSystemFamily_LINUX(),
			CpuArchitecture:       ecs.CpuArchitecture_X86_64(),
		},
	})

	fmt.Println(myECSTaskDefinition)

	// Create Container Definition of the LangTool
	myImageAsset := ecrassets.NewDockerImageAsset(stack, jsii.String(*id+"ImageAsset"), &ecrassets.DockerImageAssetProps{
		Directory: jsii.String("../docker"),
		Platform:  ecrassets.Platform_LINUX_AMD64(),
		AssetName: id,
	})

	// ecr

	myECSContainerDefinition := ecs.NewContainerDefinition(stack, jsii.String(*id+"ECSTaskContainerDefinition"), &ecs.ContainerDefinitionProps{
		ContainerName: id,
		Image:         ecs.ContainerImage_FromDockerImageAsset(myImageAsset),
		Logging: ecs.LogDriver_AwsLogs(&ecs.AwsLogDriverProps{
			LogGroup:     myECSLogGroup,
			StreamPrefix: id,
		}),
		LinuxParameters: ecs.NewLinuxParameters(stack, jsii.String(*id+"LinuxParam"), &ecs.LinuxParametersProps{
			InitProcessEnabled: jsii.Bool(true),
		}),
		TaskDefinition: myECSTaskDefinition,
		// PortMappings: &[]*ecs.PortMapping{
		// 	ContainerPort: jsii.Number(8010),
		// },
	})

	myECSContainerDefinition.AddPortMappings(&ecs.PortMapping{
		ContainerPort: jsii.Number(8010),
	})

	// myECSTaskDefinition
	// Container(jsii.String("LangToolContainer"), myECSContainerDefinition.)

	fmt.Println(myECSContainerDefinition)
	//create LangTool service security group
	myECSLangToolSG := ec2.NewSecurityGroup(stack, jsii.String(*id+"_ECSLangToolsg"), &ec2.SecurityGroupProps{
		Vpc:               myVPC,
		AllowAllOutbound:  jsii.Bool(true),
		SecurityGroupName: jsii.String(*id + "_ECSLangToolsg"),
	})
	myECSLangToolSG.AddIngressRule(ec2.Peer_SecurityGroupId(myALBsg.SecurityGroupId(), nil), ec2.Port_AllTraffic(), jsii.String("Allow Traffic from LoadBalancer"), jsii.Bool(false))

	// Create LangTool ECS Service
	myECSService := ecs.NewFargateService(stack, jsii.String(*id+"ECSService"), &ecs.FargateServiceProps{
		ServiceName:       id,
		Cluster:           myECSCluster,
		TaskDefinition:    myECSTaskDefinition,
		DesiredCount:      jsii.Number(1),
		MinHealthyPercent: jsii.Number(100),
		MaxHealthyPercent: jsii.Number(200),
		AssignPublicIp:    jsii.Bool(false),
		VpcSubnets: &ec2.SubnetSelection{
			SubnetType: ec2.SubnetType_PRIVATE_WITH_EGRESS,
		},
		SecurityGroups: &[]ec2.ISecurityGroup{
			myECSLangToolSG,
		},
		EnableExecuteCommand: jsii.Bool(true),
	})

	fmt.Println(myECSService)
	// Configure Task Autoscaling

	myALTScalingTarget := myECSService.AutoScaleTaskCount(&appautoscaling.EnableScalingProps{
		MinCapacity: jsii.Number(1),
		MaxCapacity: jsii.Number(4),
	})

	myALTScalingTarget.ScaleOnCpuUtilization(jsii.String(*id+"CPUScaling"), &ecs.CpuUtilizationScalingProps{
		TargetUtilizationPercent: jsii.Number(70),
		ScaleInCooldown:          awscdk.Duration_Seconds(jsii.Number(60)),
		ScaleOutCooldown:         awscdk.Duration_Seconds(jsii.Number(60)),
		PolicyName:               jsii.String(*id + "CPUScaling"),
	})

	// Configure Target Group of the ALB
	myELBTargetGroup.AddTarget(myECSService)

	// create Route 53 Aliases
	myR53ApiRecord := r53.NewARecord(stack, jsii.String(*id+"ApiAlias"), &r53.ARecordProps{
		Zone:           myHostedZone,
		RecordName:     jsii.String(langToolAPIName + "." + langToolDomainName + "."),
		Target:         r53.RecordTarget_FromAlias(r53targets.NewLoadBalancerTarget(myALB)),
		Comment:        jsii.String("Alias for the API"),
		DeleteExisting: jsii.Bool(true),
	})

	// myR53WebRecord := r53.NewARecord(stack, jsii.String(*id+"WebAlias"), &r53.ARecordProps{
	// 	Zone:           myHostedZone,
	// 	RecordName:     jsii.String(langToolWebName + "." + langToolDomainName + "."),
	// 	Target:         r53.RecordTarget_FromAlias(r53targets.NewLoadBalancerTarget(myALB)),
	// 	Comment:        jsii.String("Alias for the Web"),
	// 	DeleteExisting: jsii.Bool(true),
	// })

	fmt.Println(myR53ApiRecord)

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewAwsLanguageToolStack(app, jsii.String("AwsLanguageTool"), &AwsLanguageToolStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
