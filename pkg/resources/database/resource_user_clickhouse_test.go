package database_test

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type TemplateModelClickhouseUser struct {
	ResourceName string
	ServiceName  string
	Username     string
	Zone         string
	Password     string
	Plan         string
}

func TestAccClickhouseUser_basic(t *testing.T) {
	t.Parallel()

	userTpl, err := template.ParseFiles("testdata/resource_user_clickhouse.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	svcName := acctest.RandomWithPrefix(testutils.Prefix)

	userData := TemplateModelClickhouseUser{
		ResourceName: "test_user",
		ServiceName:  svcName,
		Username:     acctest.RandomWithPrefix(testutils.TestUsername),
		Zone:         testutils.TestZoneName,
		Plan:         "standard-1",
	}

	buf := &bytes.Buffer{}
	err = userTpl.Execute(buf, userData)
	if err != nil {
		t.Fatal(err)
	}
	config := buf.String()

	fullResourceName := "exoscale_dbaas_clickhouse_user.test_user"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             resource.ComposeTestCheckFunc(),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "username", userData.Username),
					resource.TestCheckResourceAttr(fullResourceName, "service", svcName),
					resource.TestCheckResourceAttr(fullResourceName, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttrSet(fullResourceName, "password"),
					resource.TestCheckResourceAttrSet(fullResourceName, "user_uuid"),
				),
			},
		},
	})
}

func TestAccClickhouseUser_withPassword(t *testing.T) {
	t.Parallel()

	userTpl, err := template.ParseFiles("testdata/resource_user_clickhouse.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	svcName := acctest.RandomWithPrefix(testutils.Prefix)

	userData := TemplateModelClickhouseUser{
		ResourceName: "test_user_pw",
		ServiceName:  svcName,
		Username:     acctest.RandomWithPrefix(testutils.TestUsername),
		Zone:         testutils.TestZoneName,
		Password:     "MyS3cureP@ssw0rd!",
		Plan:         "standard-1",
	}

	buf := &bytes.Buffer{}
	err = userTpl.Execute(buf, userData)
	if err != nil {
		t.Fatal(err)
	}
	config := buf.String()

	fullResourceName := "exoscale_dbaas_clickhouse_user.test_user_pw"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             resource.ComposeTestCheckFunc(),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "username", userData.Username),
					resource.TestCheckResourceAttr(fullResourceName, "password", userData.Password),
					resource.TestCheckResourceAttrSet(fullResourceName, "user_uuid"),
				),
			},
		},
	})
}