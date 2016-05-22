# Building

Set ```GOPATH``` to this directory, and then run ```make```

Ensure that the PATH is set to include the resulting bin directory,
and then you can run the terraform command that will produce the
exoscale plugin.

# TODO List/Missing features

## Security Groups
* There is currently no API in place to allow for listing what ingress/egress rules are in place for a security group once the group is created

## S3 Support
* Due to the AWS library in use, CORS is not supported
* Due to the AWS library in use, per-object K/V pairs are not supported