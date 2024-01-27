# Getting started with developing and running the code in this repository

## Configure Windows Development Machine

Use macOS or Linux.

If you must use Windows, the following commands might help you set up a Windows machine for the development the project. Using an IDE such as VS Code is entirely optional and not included in the steps below

Open PowerShell:

```shell
winget install -e --id Git.Git
winget install -e --id Hashicorp.Terraform
winget install -e --id Chocolatey.Chocolatey
```

Close and open PowerShell as Administrator:

```shell
choco upgrade chocolatey
```

Close and open PowerShell as Administrator:

```shell
choco install mob
choco install make
```

Close PowerShell

Note: if you're having repeated issues with ZScaler and pip, [take a look at the instructions to add the ZScaler certificate to stop pip erroring](https://elsevier.atlassian.net/wiki/spaces/TEN/pages/59535175745/SSL+encryption+Application+Development+Environment+and+Guest+OS+compatibility+requirements#Python-/-PyCharm-/-Conda) and [this if running under AWS Workspace](https://elsevier.atlassian.net/wiki/spaces/TEN/pages/59535175745/SSL+encryption+Application+Development+Environment+and+Guest+OS+compatibility+requirements#Python/PiP)

## Install Go and Docker

The best way to install Go is installing the [latest version from the Go website](https://go.dev/doc/install).

For Docker, the best way again is to get the [latest version of Docker Desktop from the Docker website](https://www.docker.com/products/docker-desktop/).

# Deployment

The Go programme [](terraform.go) is a wrapper to help you test the Terraform files on the command line. You should only ever run this against non-production.

## Prerequisite

The script uses an S3 bucket to hold it a copy of the status of the services deployed in AWS. In Terraform parlance this is called the _state_ files, and because they are on S3 as opposed to your local machine, they are called _remote state_ files.

The S3 bucket needs to be created before the Terraform commands can be run, either directly or via the Fabric script.

In the case of this project, for non-production testing at present, this remote state bucket is
_860378428577-eu-west-1-terraform-deployments_ and contains the object _tfstate/nonprod/cmi-schedulers-dsr-automation.json_ for
non-production. This was manually created and you will need to do this in each subsequent account you wish to deploy in.

## AWS Authentication

To use the script, first authenticate using https://elsevier-sso.awsapps.com/start#/ and click the "Command line or programmatic access" link:

![AWS Console Login](assets/aws-login-step1.png)

The copy the _export_ lines from Option 1 and paste into the command line:

![AWS Console Command Line Environment Vars](assets/aws-login-step-2.png)

For Linux, macOS and GitBash under Windows (not the Makefile does not work under PowerShell) copy the __macOS and Linux__ section variables in this format:

```shell
export AWS_ACCESS_KEY_ID=""
export AWS_SECRET_ACCESS_KEY=""
export AWS_SESSION_TOKEN=""
```

## Running the Go terraform program

There are separate files for [non-prod](terraform/environments/nonprod.tfvars) and [prod](terraform/environments/prod.tfvars) environments.  These contain the URL of the Claims Issuer.  For example, in non-prod, in SIT:

```terraform
claims_issuer = "https://sit.business.api.elsevier.systems" 
```

You will need to change these to fit with the local environment and [workspace](https://elsevier.atlassian.net/wiki/spaces/IN/pages/119601022841096/DBS+API-M+-+Workspaces) you're working with.

There is a [Go program](terraform.go) that _helps_ you to run Terraform commands included in the repository.  This program takes the following parameters:

```shell
go run terraform.go --help
Usage of ...../b001/exe/terraform:
  -account-number uint
    	Account number of AWS deployment target
  -app-name string
    	Application name: e.g. jwt-authorizer (default "jwt-authorizer")
  -build string
    	When running tfop plan or apply, which Lambda functions to build: all, none, <name-of-lambda> (default "all")
  -environment string
    	Target environment = prod, nonprod, etc (default "nonprod")
  -region string
    	Target region: e.g. us-east-1, eu-west-1 (default "us-east-1")
  -tfop string
    	Terraform operation = init|plan|apply|destroy
  -vpc-id string
    	The target VPC for the services (required)
```

If you're running for the first time, you need to initialise the local copy of the Terrafrom state using the Go program (the examples use an imaginary AWS account number _123456789012_; substitute this with your own target account).  For example:

```shell
go run terraform.go --account-number=123456789012 --tfop=init
```

__Note: you only need to run the _init_ command the first time, or when switching between production and non-production environments.__

To run the Terraform plan:

```shell
go run terraform.go --account-number=123456789012 --tfop=plan --vpc-id=vpc-a07b34c9
```

And to apply:

```shell
go run terraform.go --account-number=123456789012 --tfop=apply --vpc-id=vpc-a07b34c9
```

And you will need to confirm the changes by answering "yes".

#### Selective building of Lambdas

When developing Terraform files and testing them, you can prevent the Lambda from building everytime by adding the _--build=none_ flag.  For example:

```shell
go run terraform.go --account-number=123456789012 --tfop=apply --build=none --vpc-id=vpc-a07b34c9
```

You can also choose a specific Lambda to build, by specifying it instead of _none_ or _all_, for example:

```shell
go run terraform.go --account-number=123456789012 --tfop=apply --build=jwt-authorizer --vpc-id=vpc-a07b34c9
```

Would build only the [JWT Authorizer Lambda](lambdas/jwt-authorizer) and deploy it to AWS.

### FAQ

#### I get "Backend configuration changed" whilst initialising or another operation

You get an error similar to the following:

```shell
$ go run terraform.go --account-number=123456789012 --tfop=init 
2024/01/16 16:30:06 error running Init: exit status 1

Error: Backend configuration changed
```

This should be resolved by removing the directory `terraform/.terraform` and re-running the _init_ process.