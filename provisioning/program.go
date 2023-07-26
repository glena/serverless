package provisioning

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecr"
	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AWSConfiguration struct {
	Region    string
	AccessKey string
	SecretKey string
}

type Provisioning struct {
	Configuration AWSConfiguration
}

func (me *Provisioning) deploy(ctx *pulumi.Context, name string, script string) error {
	imageName := name
	serviceName := name + "-" + uuid.New().String()
	deploymentName := serviceName

	repo, err := ecr.NewRepository(ctx, imageName, nil)
	if err != nil {
		return err
	}

	registryInfo := repo.RegistryId.ApplyT(func(id string) (docker.Registry, error) {
		creds, err := ecr.GetCredentials(ctx, &ecr.GetCredentialsArgs{RegistryId: id})
		if err != nil {
			return docker.Registry{}, err
		}
		decoded, err := base64.StdEncoding.DecodeString(creds.AuthorizationToken)
		if err != nil {
			return docker.Registry{}, err
		}
		parts := strings.Split(string(decoded), ":")
		if len(parts) != 2 {
			return docker.Registry{}, errors.New("invalid credentials")
		}
		return docker.Registry{
			Server:   &creds.ProxyEndpoint,
			Username: &parts[0],
			Password: &parts[1],
		}, nil
	}).(docker.RegistryOutput)

	image, err := docker.NewImage(ctx, imageName, &docker.ImageArgs{

		Build: &docker.DockerBuildArgs{
			Dockerfile: pulumi.String("./provisioning/Dockerfile"),
			Platform:   pulumi.String("linux/amd64"),
			Args: pulumi.StringMap{
				"script": pulumi.String(script),
			},
		},
		ImageName: repo.RepositoryUrl,
		Registry:  registryInfo,
	})

	if err != nil {
		return err
	}

	appLabels := pulumi.StringMap{
		"app": pulumi.String(deploymentName),
	}

	deployment, err := appsv1.NewDeployment(ctx, deploymentName, &appsv1.DeploymentArgs{
		Spec: appsv1.DeploymentSpecArgs{
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: appLabels,
			},
			Replicas: pulumi.Int(1),
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: appLabels,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String(deploymentName),
							Image: image.ImageName,
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									ContainerPort: pulumi.Int(80),
								},
							},
						}},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	feType := "LoadBalancer"

	template := deployment.Spec.ApplyT(func(v *appsv1.DeploymentSpec) *corev1.PodTemplateSpec {
		return &v.Template
	}).(corev1.PodTemplateSpecPtrOutput)

	meta := template.ApplyT(func(v *corev1.PodTemplateSpec) *metav1.ObjectMeta { return v.Metadata }).(metav1.ObjectMetaPtrOutput)

	service := &corev1.ServiceArgs{
		Metadata: meta,
		Spec: &corev1.ServiceSpecArgs{
			Type:     pulumi.String(feType),
			Selector: appLabels,
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(80),
					Protocol:   pulumi.String("TCP"),
				},
			},
		},
	}

	frontend, err := corev1.NewService(ctx, serviceName, service)

	if err != nil {
		return err
	}

	url := frontend.Status.ApplyT(func(val *corev1.ServiceStatus) string {
		return *val.LoadBalancer.Ingress[0].Hostname
	}).(pulumi.StringOutput)

	ctx.Export("url", url)
	return nil
}

func (me *Provisioning) Provision(name string, script string) (string, error) {
	ctx := context.Background()

	// create or select a stack matching the specified name and project.
	// this will set up a workspace with everything necessary to run our inline program (deployFunc)
	s, err := auto.UpsertStackInlineSource(ctx, name, "onboarding-faas", func(ctx *pulumi.Context) error {
		return me.deploy(ctx, name, script)
	})

	if err != nil {
		fmt.Printf("Failed to upsert stack: %v\n", err)
		return "", err
	}

	fmt.Printf("Created/Selected stack %q\n", name)

	w := s.Workspace()

	w.SetAllConfig(ctx, name, auto.ConfigMap{
		"aws:region": auto.ConfigValue{
			Value:  me.Configuration.Region,
			Secret: false,
		},
		"aws:accessKey": auto.ConfigValue{
			Value:  me.Configuration.AccessKey,
			Secret: false,
		},
		"aws:secretKey": auto.ConfigValue{
			Value:  me.Configuration.SecretKey,
			Secret: true,
		},
	})

	fmt.Println("Installing the AWS plugin")

	// for inline source programs, we must manage plugins ourselves
	err = w.InstallPlugin(ctx, "aws", "v4.0.0")
	if err != nil {
		fmt.Printf("Failed to install program plugins: %v\n", err)
		return "", err
	}

	fmt.Println("Successfully installed AWS plugin")

	// set stack configuration specifying the AWS region to deploy
	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "us-west-2"})

	fmt.Println("Successfully set config")
	fmt.Println("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		return "", err
	}

	fmt.Println("Refresh succeeded!")

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	// run the update to deploy our s3 website
	res, err := s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		return "", err
	}

	// get the URL from the stack outputs
	url, ok := res.Outputs["url"].Value.(string)
	if !ok {
		fmt.Println("Failed to unmarshall output URL")
		return "", errors.New("failed to unmarshall output URL")
	}

	fmt.Printf("URL: %s\n", url)

	fmt.Println("Update succeeded!")

	return url, nil
}
