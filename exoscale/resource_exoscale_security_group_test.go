package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var (
	testAccResourceSecurityGroupName                   = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSecurityGroupExternalSources        = []string{"1.1.1.1/32", "2.2.2.2/32"}
	testAccResourceSecurityGroupExternalSourcesUpdated = []string{"2.2.2.2/32", "3.3.3.3/32"}
	testAccResourceSecurityGroupDescription            = acctest.RandString(10)

	testAccResourceSecurityGroupConfigCreate = fmt.Sprintf(`
resource "exoscale_security_group" "test" {
  name             = "%s"
  external_sources = ["%s"]
  description      = "%s"
}
`,
		testAccResourceSecurityGroupName,
		strings.Join(testAccResourceSecurityGroupExternalSources, `","`),
		testAccResourceSecurityGroupDescription,
	)

	testAccResourceSecurityGroupConfigUpdate = fmt.Sprintf(`
resource "exoscale_security_group" "test" {
  name             = "%s"
  external_sources = ["%s"]
  description      = "%s"
}
`,
		testAccResourceSecurityGroupName,
		strings.Join(testAccResourceSecurityGroupExternalSourcesUpdated, `","`),
		testAccResourceSecurityGroupDescription,
	)
)

func TestAccResourceSecurityGroup(t *testing.T) {
	var (
		r             = "exoscale_security_group.test"
		securityGroup egoscale.SecurityGroup
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSecurityGroupDestroy(&securityGroup),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceSecurityGroupConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists(r, &securityGroup),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceSecurityGroupDescription, *securityGroup.Description)
						a.ElementsMatch(testAccResourceSecurityGroupExternalSources, *securityGroup.ExternalSources)
						a.Equal(testAccResourceSecurityGroupName, *securityGroup.Name)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSecurityGroupAttrDescription:            validateString(testAccResourceSecurityGroupDescription),
						resSecurityGroupAttrExternalSources + ".0": validateString(testAccResourceSecurityGroupExternalSources[0]),
						resSecurityGroupAttrExternalSources + ".1": validateString(testAccResourceSecurityGroupExternalSources[1]),
						resSecurityGroupAttrName:                   validateString(testAccResourceSecurityGroupName),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceSecurityGroupConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists(r, &securityGroup),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.ElementsMatch(testAccResourceSecurityGroupExternalSourcesUpdated, *securityGroup.ExternalSources)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSecurityGroupAttrDescription:            validateString(testAccResourceSecurityGroupDescription),
						resSecurityGroupAttrExternalSources + ".0": validateString(testAccResourceSecurityGroupExternalSourcesUpdated[0]),
						resSecurityGroupAttrExternalSources + ".1": validateString(testAccResourceSecurityGroupExternalSourcesUpdated[1]),
						resSecurityGroupAttrName:                   validateString(testAccResourceSecurityGroupName),
					})),
				),
			},
			{
				// Import
				ResourceName:      r,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSecurityGroupAttrDescription:            validateString(testAccResourceSecurityGroupDescription),
							resSecurityGroupAttrExternalSources + ".0": validateString(testAccResourceSecurityGroupExternalSourcesUpdated[0]),
							resSecurityGroupAttrExternalSources + ".1": validateString(testAccResourceSecurityGroupExternalSourcesUpdated[1]),
							resSecurityGroupAttrName:                   validateString(testAccResourceSecurityGroupName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSecurityGroupExists(r string, securityGroup *egoscale.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))
		res, err := client.GetSecurityGroup(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*securityGroup = *res
		return nil
	}
}

func testAccCheckResourceSecurityGroupDestroy(securityGroup *egoscale.SecurityGroup) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		_, err := client.GetSecurityGroup(ctx, testZoneName, *securityGroup.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Security Group still exists")
	}
}
