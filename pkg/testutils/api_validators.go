package testutils

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ssgreg/repeat"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
)

func CheckAntiAffinityGroupExists(r string, res *egoscale.AntiAffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName),
		)
		data, err := client.GetAntiAffinityGroup(ctx, TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*res = *data
		return nil
	}
}

func CheckAntiAffinityGroupDestroy(res *egoscale.AntiAffinityGroup) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if res == nil {
			return nil
		}

		client, err := APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName),
		)

		_, err = client.GetAntiAffinityGroup(ctx, TestZoneName, *res.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Anti-Affinity Group still exists")
	}
}

func CheckInstanceExists(r string, testInstance *egoscale.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName),
		)

		res, err := client.GetInstance(ctx, TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*testInstance = *res
		return nil
	}
}

func CheckInstanceDestroy(testInstance *egoscale.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testInstance == nil {
			return nil
		}

		client, err := APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName),
		)

		_, err = client.GetInstance(ctx, TestZoneName, *testInstance.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Compute testInstance still exists")
	}
}

func CheckSecurityGroupExists(r string, securityGroup *egoscale.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))
		res, err := client.GetSecurityGroup(ctx, TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*securityGroup = *res
		return nil
	}
}

func CheckSecurityGroupDestroy(securityGroup *egoscale.SecurityGroup) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if securityGroup == nil {
			return nil
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		_, err = client.GetSecurityGroup(ctx, TestZoneName, *securityGroup.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Security Group still exists")
	}
}

func CheckPrivateNetworkExists(r string, privateNetwork *egoscale.PrivateNetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))
		res, err := client.GetPrivateNetwork(ctx, TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*privateNetwork = *res
		return nil
	}
}

func CheckPrivateNetworkDestroy(privateNetwork *egoscale.PrivateNetwork) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if privateNetwork == nil {
			return nil
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		_, err = client.GetPrivateNetwork(ctx, TestZoneName, *privateNetwork.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Private Network still exists")
	}
}

func CheckElasticIPExists(r string, elasticIP *egoscale.ElasticIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		res, err := client.GetElasticIP(ctx, TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*elasticIP = *res
		return nil
	}
}

func CheckElasticIPDestroy(elasticIP *egoscale.ElasticIP) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if elasticIP == nil {
			return nil
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		_, err = client.GetElasticIP(ctx, TestZoneName, *elasticIP.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Elastic IP still exists")
	}
}

func CheckSSHKeyExists(r string, sshKey *egoscale.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		res, err := client.GetSSHKey(ctx, TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*sshKey = *res
		return nil
	}
}

func CheckSSHKeyDestroy(sshKey *egoscale.SSHKey) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if sshKey == nil {
			return nil
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		_, err = client.GetSSHKey(ctx, TestZoneName, *sshKey.Name)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("SSH Key still exists")
	}
}

func CheckInstancePoolExists(r string, pool *egoscale.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		res, err := client.GetInstancePool(ctx, TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*pool = *res
		return nil
	}
}

func CheckInstancePoolDestroy(pool *egoscale.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if pool == nil {
			return nil
		}

		client, err := APIClient()
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(TestEnvironment(), TestZoneName))

		// The Exoscale API can be a bit slow to reflect the deletion operation
		// in the Instance Pool state, so we give it the benefit of the doubt
		// by retrying a few times before returning an error.
		return repeat.Repeat(
			repeat.Fn(func() error {
				pool, err := client.GetInstancePool(ctx, TestZoneName, *pool.ID)
				if err != nil {
					if errors.Is(err, exoapi.ErrNotFound) {
						return nil
					}
					return err
				}

				if *pool.State == "destroying" {
					return nil
				}

				return errors.New("Instance Pool still exists")
			}),
			repeat.StopOnSuccess(),
			repeat.LimitMaxTries(10),
			repeat.WithDelay(
				repeat.FixedBackoff(3*time.Second).Set(),
				repeat.SetContext(ctx),
			),
		)
	}
}
