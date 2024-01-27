package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
	"log"
	"os"
	"os/exec"
)

var awsAccountNumber = flag.Uint("account-number", 0, "Account number of AWS deployment target")
var environment = flag.String("environment", "nonprod", "Target environment = prod, nonprod, etc")
var awsRegion = flag.String("region", "us-east-1", "Target region: e.g. us-east-1, eu-west-1")
var appName = flag.String("app-name", "jwt-authorizer", "Application name: e.g. jwt-authorizer")
var tfOp = flag.String("tfop", "", "Terraform operation = init|plan|apply|destroy")
var build = flag.String("build", "all", "When running tfop plan or apply, which Lambda functions to build: all, none, <name-of-lambda>")
var vpcId = flag.String("vpc-id", "", "The target VPC for the services (required)")

func main() {
	flag.Parse()

	if shouldBuildLambdas() {
		buildLambdas()
	}

	runTerraformCommand()
}

func runTerraformCommand() {
	tf := setupTerraformExec(context.Background())
	var buf bytes.Buffer
	tf.SetStdout(&buf)
	tfWorkingBucket := fmt.Sprintf("%d-%s-terraform-deployments", *awsAccountNumber, *awsRegion)
	switch *tfOp {
	case "init":
		terraformInit(tf, tfWorkingBucket, *awsRegion)
	case "plan":
		terraformPlan(tf, tfWorkingBucket, *awsAccountNumber, *environment)
	case "apply":
		terraformApply(tf, tfWorkingBucket, *awsAccountNumber, *environment)
	case "destroy":
		log.Fatalf("Destroy needs implementing!")
	default:
		log.Fatalf("Bad operation: --tfop should be one of init, plan, apply, skip, or destroy")
	}
	log.Println(buf.String())
}

func shouldBuildLambdas() bool {
	return *build != "none" && (*tfOp == "plan" || *tfOp == "apply")
}

func setupTerraformExec(ctx context.Context) *tfexec.Terraform {
	log.Println("installing Terraform...")
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion("1.6")),
	}

	execPath, err := installer.Install(ctx)
	if err != nil {
		log.Fatalf("error installing Terraform: %s", err)
	}

	workingDir := "terraform"
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
	}
	return tf
}

func terraformInit(tf *tfexec.Terraform, tfWorkingBucket string, awsRegion string) {
	log.Println("initialising Terraform...")
	if err := tf.Init(context.Background(),
		tfexec.Upgrade(true),
		tfexec.BackendConfig(fmt.Sprintf("key=tfstate/%s/%s.json", *environment, *appName)),
		tfexec.BackendConfig(fmt.Sprintf("bucket=%s", tfWorkingBucket)),
		tfexec.BackendConfig(fmt.Sprintf("region=%s", awsRegion))); err != nil {
		log.Fatalf("error running Init: %s", err)
	}
}

func terraformPlan(tf *tfexec.Terraform, tfWorkingBucket string, awsAccountNumber uint, environment string) {
	log.Println("planning Terraform...")
	_, err := tf.Plan(context.Background(),
		tfexec.Refresh(true),
		tfexec.Var(fmt.Sprintf("terraform_working_bucket=%s", tfWorkingBucket)),
		tfexec.Var(fmt.Sprintf("account_number=%d", awsAccountNumber)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("vpc_id=%s", *vpcId)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	)
	if err != nil {
		log.Fatalf("error running Plan: %s", err)
	}
}

func terraformApply(tf *tfexec.Terraform, workingBucket string, awsAccountNumber uint, environment string) {
	log.Println("applying Terraform...")
	if err := tf.Apply(context.Background(),
		tfexec.Refresh(true),
		tfexec.Var(fmt.Sprintf("terraform_working_bucket=%s", workingBucket)),
		tfexec.Var(fmt.Sprintf("account_number=%d", awsAccountNumber)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("vpc_id=%s", *vpcId)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	); err != nil {
		log.Fatalf("error running Apply: %s", err)
	}
	displayTerraformOutputs(tf)
}

func displayTerraformOutputs(tf *tfexec.Terraform) {
	outputs, err := tf.Output(context.Background())
	if err != nil {
		log.Fatalf("Error outputting outputs: %v", err)
	}
	if len(outputs) > 0 {
		fmt.Println("Terraform outputs:")
	}
	for key := range outputs {
		if outputs[key].Sensitive {
			continue
		}
		fmt.Println(fmt.Sprintf("%s = %s\n", key, string(outputs[key].Value)))
	}
}

func buildLambdas() {
	log.Println("building Lambdas...")
	if *build == "all" {
		items, err := os.ReadDir("lambdas")
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range items {
			if item.IsDir() {
				buildLambda(item.Name())
			}
		}
	} else {
		buildLambda(*build)
	}
}

func buildLambda(lambdaName string) {
	log.Printf("running tests for %s Lambda...\n", lambdaName)
	runCmdIn(fmt.Sprintf("lambdas/%s", lambdaName), "make", "test")
	log.Printf("building %s Lambda...\n", lambdaName)
	runCmdIn(fmt.Sprintf("lambdas/%s", lambdaName), "make", "target")
}

func runCmdIn(dir string, command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		log.Fatalf("error running %s %s: %s", command, args, err)
	}
	return cmd
}
