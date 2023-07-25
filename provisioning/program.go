package provisioning

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecr"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func deploy(ctx *pulumi.Context) error {
	isMinikube := false//config.GetBool(ctx, "isMinikube")
	customImageName := "node-app"

	image, err := ecr.NewImage(ctx, customImageName, &ecr.ImageArgs{
		// RepositoryUrl: repo.url,
		Dockerfile:    pulumi.String("./" + customImageName),
	})

	appLabels := pulumi.StringMap{
		"app": pulumi.String(customImageName),
	}

	deployment, err := appsv1.NewDeployment(ctx, customImageName, &appsv1.DeploymentArgs{
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
							Name:  pulumi.String(customImageName),
							Image: image.ImageUri,
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
	if isMinikube {
		feType = "ClusterIP"
	}

	template := deployment.Spec.ApplyT(func(v *appsv1.DeploymentSpec) *corev1.PodTemplateSpec {
		return &v.Template
	}).(corev1.PodTemplateSpecPtrOutput)

	meta := template.ApplyT(func(v *corev1.PodTemplateSpec) *metav1.ObjectMeta { return v.Metadata }).(metav1.ObjectMetaPtrOutput)

	service := &corev1.ServiceArgs{
		Metadata: meta,
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String(feType),
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

	frontend, err := corev1.NewService(ctx, customImageName, service)

	var ip pulumi.StringOutput

	if isMinikube {
		ip = frontend.Spec.ApplyT(func(val *corev1.ServiceSpec) string {
			if val.ClusterIP != nil {
				return *val.ClusterIP
			}
			return ""
		}).(pulumi.StringOutput)
	} else {
		ip = frontend.Status.ApplyT(func(val *corev1.ServiceStatus) string {
			if val.LoadBalancer.Ingress[0].Ip != nil {
				return *val.LoadBalancer.Ingress[0].Ip
			}
			return *val.LoadBalancer.Ingress[0].Hostname
		}).(pulumi.StringOutput)
	}

	ctx.Export("ip", ip)
	return nil
}

func Provision(destroy bool) error {
	ctx := context.Background()

	projectName := "faas"
	// we use a simple stack name here, but recommend using auto.FullyQualifiedStackName for maximum specificity.
	stackName := "dev"
	// stackName := auto.FullyQualifiedStackName("myOrgOrUser", projectName, stackName)

	// create or select a stack matching the specified name and project.
	// this will set up a workspace with everything necessary to run our inline program (deployFunc)
	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, deploy)

	fmt.Printf("Created/Selected stack %q\n", stackName)

	w := s.Workspace()

	fmt.Println("Installing the AWS plugin")

	// for inline source programs, we must manage plugins ourselves
	err = w.InstallPlugin(ctx, "aws", "v4.0.0")
	if err != nil {
		fmt.Printf("Failed to install program plugins: %v\n", err)
		return err
	}

	fmt.Println("Successfully installed AWS plugin")

	// set stack configuration specifying the AWS region to deploy
	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "us-west-2"})

	fmt.Println("Successfully set config")
	fmt.Println("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		return err
	}

	fmt.Println("Refresh succeeded!")

	if destroy {
		fmt.Println("Starting stack destroy")

		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
		}
		fmt.Println("Stack successfully destroyed")
		return err
	}

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	// run the update to deploy our s3 website
	res, err := s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		return err
	}

	fmt.Println("Update succeeded!")

	// get the URL from the stack outputs
	url, ok := res.Outputs["websiteUrl"].Value.(string)
	if !ok {
		fmt.Println("Failed to unmarshall output URL")
		return errors.New("Failed to unmarshall output URL")
	}

	fmt.Printf("URL: %s\n", url)

	return nil
}
