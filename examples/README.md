# Test execution

To run these tests ensure that the terraform binary and plugin are in
your path.  Then for the appropriate plan, rename it from X.tf.orig to
X.tf, then run ``$ terraform plan`` to verify, ``$ terraform apply`` to
apply and ``$ terraform destory`` to clean up.

As a reminder, be sure to update the resource section for each plan to
include your Exoscale API tokens.
