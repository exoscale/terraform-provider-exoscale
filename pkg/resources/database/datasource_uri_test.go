package database_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type DataSourceURIModel struct {
	ResourceName string

	Name string
	Type string
	Zone string
}

func testDataSourceURI(t *testing.T) {
	tplData, err := template.ParseFiles("testdata/datasource.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "data.exoscale_database_uri.test"
	data := DataSourceURIModel{
		ResourceName: "test",
		Name:         "exoscale_database.test.name",
		Zone:         testutils.TestZoneName,
	}

	// Test database Pg URI
	tplResourcePg, err := template.ParseFiles("testdata/resource_pg.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	resourcePg := TemplateModelPg{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
	}
	buf := &bytes.Buffer{}
	err = tplResourcePg.Execute(buf, &resourcePg)
	if err != nil {
		t.Fatal(err)
	}
	part := buf.String()

	data.Type = "pg"
	buf = &bytes.Buffer{}
	err = tplData.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf("%s\n%s", part, buf.String())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("pg", resourcePg.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
				),
			},
		},
	})

	// Test database Mysql URI
	tplResourceMysql, err := template.ParseFiles("testdata/resource_mysql.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	resourceMysql := TemplateModelMysql{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
	}
	buf = &bytes.Buffer{}
	err = tplResourceMysql.Execute(buf, &resourceMysql)
	if err != nil {
		t.Fatal(err)
	}
	part = buf.String()

	data.Type = "mysql"
	buf = &bytes.Buffer{}
	err = tplData.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	config = fmt.Sprintf("%s\n%s", part, buf.String())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("mysql", resourceMysql.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
				),
			},
		},
	})

	// Test database Redis URI
	tplResourceRedis, err := template.ParseFiles("testdata/resource_redis.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	resourceRedis := TemplateModelRedis{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
	}
	buf = &bytes.Buffer{}
	err = tplResourceRedis.Execute(buf, &resourceRedis)
	if err != nil {
		t.Fatal(err)
	}
	part = buf.String()

	data.Type = "redis"
	buf = &bytes.Buffer{}
	err = tplData.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	config = fmt.Sprintf("%s\n%s", part, buf.String())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("redis", resourceRedis.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
				),
			},
		},
	})

	// Test database Kafka URI
	tplResourceKafka, err := template.ParseFiles("testdata/resource_kafka.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	resourceKafka := TemplateModelKafka{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "business-4",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
	}
	buf = &bytes.Buffer{}
	err = tplResourceKafka.Execute(buf, &resourceKafka)
	if err != nil {
		t.Fatal(err)
	}
	part = buf.String()

	data.Type = "kafka"
	buf = &bytes.Buffer{}
	err = tplData.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	config = fmt.Sprintf("%s\n%s", part, buf.String())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("kafka", resourceKafka.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
				),
			},
		},
	})

	// Test database Opensearch URI
	tplResourceOpensearch, err := template.ParseFiles("testdata/resource_opensearch.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	resourceOpensearch := TemplateModelOpensearch{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Dashboards:            &TemplateModelOpensearchDashboards{Enabled: false},
	}
	buf = &bytes.Buffer{}
	err = tplResourceOpensearch.Execute(buf, &resourceOpensearch)
	if err != nil {
		t.Fatal(err)
	}
	part = buf.String()

	data.Type = "opensearch"
	buf = &bytes.Buffer{}
	err = tplData.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	config = fmt.Sprintf("%s\n%s", part, buf.String())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("opensearch", resourceOpensearch.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
				),
			},
		},
	})

	// Test database Grafana URI
	tplResourceGrafana, err := template.ParseFiles("testdata/resource_grafana.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	resourceGrafana := TemplateModelGrafana{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
	}
	buf = &bytes.Buffer{}
	err = tplResourceGrafana.Execute(buf, &resourceGrafana)
	if err != nil {
		t.Fatal(err)
	}
	part = buf.String()

	data.Type = "grafana"
	buf = &bytes.Buffer{}
	err = tplData.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	config = fmt.Sprintf("%s\n%s", part, buf.String())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("grafana", resourceGrafana.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
				),
			},
		},
	})
}
