package database_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type TemplateModelValkeyUser struct {
	ResourceName string
	Username     string
	Service      string
	Zone         string
}

func testResourceValkeyUser(t *testing.T) {
	t.Parallel()

	svcTpl, err := template.ParseFiles("testdata/resource_valkey.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	userTpl, err := template.ParseFiles("testdata/resource_user_valkey.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	svcData := TemplateModelValkey{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
	}

	userData := TemplateModelValkeyUser{
		ResourceName: "test_user",
		Username:     acctest.RandomWithPrefix(testutils.TestUsername),
		Service:      "exoscale_database.test.name",
		Zone:         testutils.TestZoneName,
	}

	svcBuf := &bytes.Buffer{}
	if err = svcTpl.Execute(svcBuf, &svcData); err != nil {
		t.Fatal(err)
	}

	userBuf := &bytes.Buffer{}
	if err = userTpl.Execute(userBuf, &userData); err != nil {
		t.Fatal(err)
	}

	config := fmt.Sprintf("%s\n%s", svcBuf.String(), userBuf.String())

	fullUserResourceName := "exoscale_dbaas_valkey_user.test_user"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("valkey", svcData.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullUserResourceName, "username", userData.Username),
					resource.TestCheckResourceAttr(fullUserResourceName, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttrSet(fullUserResourceName, "password"),
					resource.TestCheckResourceAttrSet(fullUserResourceName, "type"),
					resource.TestCheckResourceAttrSet(fullUserResourceName, "id"),
				),
			},
			{
				// Import
				ResourceName: fullUserResourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%s@%s", svcData.Name, userData.Username, testutils.TestZoneName), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}


