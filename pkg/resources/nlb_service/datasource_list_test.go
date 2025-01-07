package nlb_service_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testListDataSource(t *testing.T) {
	buf := &bytes.Buffer{}

	// template testdata
	tpl, err := template.ParseFiles("../../testutils/testdata/datasource_template.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	templateModel := testutils.DataSourceTemplateModel{
		ResourceName: "test",
		Zone:         testutils.TestZoneName,
		Name:         testutils.TestInstanceTemplateName,
	}
	err = tpl.Execute(buf, &templateModel)
	if err != nil {
		t.Fatal(err)
	}
	buf.WriteString("\n")

	// ipool testdata
	tpl, err = template.ParseFiles("../../testutils/testdata/resource_instance_pool.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	ipoolModel := testutils.ResourceInstancePoolModel{
		ResourceName: "test",
		Zone:         testutils.TestZoneName,
		Name:         acctest.RandomWithPrefix(testutils.Prefix),
		Size:         2,
		TemplateID:   "data.exoscale_template.test.id",
		Type:         "standard.medium",
		DiskSize:     20,
	}
	err = tpl.Execute(buf, &ipoolModel)
	if err != nil {
		t.Fatal(err)
	}
	buf.WriteString("\n")

	// nlb testdata
	tpl, err = template.ParseFiles("../../testutils/testdata/resource_nlb.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	nlbModel := testutils.ResourceNLBModel{
		ResourceName: "test",
		Zone:         testutils.TestZoneName,
		Name:         acctest.RandomWithPrefix(testutils.Prefix),
	}
	err = tpl.Execute(buf, &nlbModel)
	if err != nil {
		t.Fatal(err)
	}
	buf.WriteString("\n")

	// nlb_service testdata
	tpl, err = template.ParseFiles("../../testutils/testdata/resource_nlb_service.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	nlbServiceModel := testutils.ResourceNLBServiceModel{
		ResourceName:    "test",
		Zone:            testutils.TestZoneName,
		Name:            acctest.RandomWithPrefix(testutils.Prefix),
		NLBID:           "exoscale_nlb.test.id",
		InstancePoolID:  "exoscale_instance_pool.test.id",
		Port:            8080,
		TargetPort:      8080,
		HealthcheckPort: 8081,
	}
	err = tpl.Execute(buf, &nlbServiceModel)
	if err != nil {
		t.Fatal(err)
	}
	buf.WriteString("\n")

	configBase := buf.String()
	fullResourceName := "data.exoscale_nlb_service_list.test"

	// datasource by name
	buf = &bytes.Buffer{}
	tpl, err = template.ParseFiles("../../testutils/testdata/datasource_nlb_service_list.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	model := testutils.DataSourceNlbServiceListModel{
		ResourceName: "test",
		Zone:         testutils.TestZoneName,
		Name:         "exoscale_nlb.test.name",
		RawConfig:    "depends_on = [exoscale_nlb_service.test]",
	}
	err = tpl.Execute(buf, &model)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configBase + buf.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "zone", model.Zone),
					resource.TestCheckResourceAttr(fullResourceName, "services.#", "1"),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.port",
						fmt.Sprintf("%d", nlbServiceModel.Port),
					),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.target_port",
						fmt.Sprintf("%d", nlbServiceModel.TargetPort),
					),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.name",
						nlbServiceModel.Name,
					),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.healthcheck.port",
						fmt.Sprintf("%d", nlbServiceModel.HealthcheckPort),
					),
				),
			},
		},
	})

	// datasource by id
	buf = &bytes.Buffer{}
	model.Name = ""
	model.ID = "exoscale_nlb.test.id"
	err = tpl.Execute(buf, &model)
	if err != nil {
		t.Fatal(err)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configBase + buf.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "zone", model.Zone),
					resource.TestCheckResourceAttr(fullResourceName, "services.#", "1"),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.port",
						fmt.Sprintf("%d", nlbServiceModel.Port),
					),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.target_port",
						fmt.Sprintf("%d", nlbServiceModel.TargetPort),
					),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.name",
						nlbServiceModel.Name,
					),
					resource.TestCheckResourceAttr(
						fullResourceName,
						"services.0.healthcheck.port",
						fmt.Sprintf("%d", nlbServiceModel.HealthcheckPort),
					),
				),
			},
		},
	})
}
