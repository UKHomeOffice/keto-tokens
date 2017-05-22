### **Keto Tokens**

----

Is a small utility service used to either produce or consume kubernetes registration tokens by the compute nodes. The service effectively discovers auto scaling group by filters, iterate the group membership and looks for compute nodes in need of registration token. The one-time are then applied the tags in the instance and consumed by the compute node.

#### **Command Line Usage**

```shell
[jest@starfury keto-tokens]$ bin/keto-tokens --help
NAME:
   keto-tokens - is a client/server used to generate and consume kubelet registration tokens

USAGE:
   keto-tokens [global options] command [command options] [arguments...]

VERSION:
   v0.0.1

AUTHOR:
   devops@digital.homeoffice.gov.uk

COMMANDS:
     server   starts the service, generating the registration tokens for kubelets
     client   retrieves a kubenetes registration tokens for compute kubelets
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   -c NAME, --cloud NAME  specify the cloud provider (aws, gce) NAME (default: "aws") [$CLOUD_PROVIDER]
   --verbose BOOL         switch on verbose logging mode BOOL [$VERBOSE]
   --help, -h             show help
   --version, -v          print the version
```

#### **IAM Permissions**

For the **server** component the following permissions are required;

```JSON
{
    "Statement": [
        {
            "Action": [
                "autoscaling:DescribeAutoScalingGroups",
                "ec2:CreateTags",
                "ec2:DescribeTags",
                "ec2:DescribeInstances"
            ],
            "Resource": "*",
            "Effect": "Allow"
        }
    ]
}
```
And for the client, we need to be able to read our own tags and the permission to update them.
```JSON
{
    "Statement": [
        {
            "Action": [
                "ec2:CreateTags",
                "ec2:DescribeTags",
                "ec2:DescribeInstances"
            ],
            "Resource": "*",
            "Effect": "Allow"
        }
    ]
}
```
Note: the above it just a guideline, it would be preferable to lock permissions down to the specific instance - i.e. ensure the compute instance itself can describe it's own tags.
