package database_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

type TemplateModelKafkaTopicACL struct {
	ResourceName string

	Zone       string
	Service    string
	Username   string
	Topic      string
	Permission string
}

type TemplateModelKafkaSchemaRegistryACL struct {
	ResourceName string

	Zone       string
	Service    string
	Username   string
	Resource   string
	Permission string
}

func testResourceKafkaACL(t *testing.T) {
	t.Parallel()

	serviceTpl, err := template.ParseFiles("testdata/resource_kafka.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	topicACLTpl, err := template.ParseFiles("testdata/resource_kafka_topic_acl.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	schemaRegistryACLTpl, err := template.ParseFiles("testdata/resource_kafka_schema_registry_acl.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	serviceFullResourceName := "exoscale_dbaas.test"
	serviceData := TemplateModelKafka{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "startup-4",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		EnableSchemaRegistry:  true,
	}

	topicACLFullResourceName := "exoscale_dbaas_kafka_topic_acl.test"
	topicACLDataCreate := TemplateModelKafkaTopicACL{
		ResourceName: "test",
		Zone:         serviceData.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
		Username:     "acl-user",
		Topic:        "orders-*",
		Permission:   "read",
	}
	topicACLDataUpdate := topicACLDataCreate
	topicACLDataUpdate.Permission = "readwrite"

	schemaRegistryACLFullResourceName := "exoscale_dbaas_kafka_schema_registry_acl.test"
	schemaRegistryACLData := TemplateModelKafkaSchemaRegistryACL{
		ResourceName: "test",
		Zone:         serviceData.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
		Username:     "acl-user",
		Resource:     "Subject:orders-*",
		Permission:   "schema_registry_read",
	}

	render := func(topicACLData TemplateModelKafkaTopicACL) string {
		buf := &bytes.Buffer{}
		if err := serviceTpl.Execute(buf, &serviceData); err != nil {
			t.Fatal(err)
		}
		if err := topicACLTpl.Execute(buf, &topicACLData); err != nil {
			t.Fatal(err)
		}
		if err := schemaRegistryACLTpl.Execute(buf, &schemaRegistryACLData); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	configCreate := render(topicACLDataCreate)
	configUpdate := render(topicACLDataUpdate)

	var createdTopicACLID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("kafka", serviceData.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(topicACLFullResourceName, "acl_id"),
					resource.TestCheckResourceAttrSet(schemaRegistryACLFullResourceName, "acl_id"),
					func(s *terraform.State) error {
						createdTopicACLID = s.RootModule().Resources[topicACLFullResourceName].Primary.Attributes["acl_id"]

						acls, err := getKafkaACLs(serviceData.Name)
						if err != nil {
							return err
						}

						if err := checkKafkaTopicACL(acls, s, topicACLFullResourceName, &topicACLDataCreate); err != nil {
							return err
						}

						return checkKafkaSchemaRegistryACL(acls, s, schemaRegistryACLFullResourceName, &schemaRegistryACLData)
					},
				),
			},
			{
				// The API has no update: a permission change replaces the entry
				Config: configUpdate,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(topicACLFullResourceName, plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectResourceAction(schemaRegistryACLFullResourceName, plancheck.ResourceActionNoop),
					},
				},
				Check: func(s *terraform.State) error {
					acls, err := getKafkaACLs(serviceData.Name)
					if err != nil {
						return err
					}

					for _, acl := range acls.TopicAcl {
						if string(acl.ID) == createdTopicACLID {
							return fmt.Errorf("replaced topic ACL entry %q still exists", createdTopicACLID)
						}
					}

					return checkKafkaTopicACL(acls, s, topicACLFullResourceName, &topicACLDataUpdate)
				},
			},
			{
				// An entry deleted outside Terraform is created again
				PreConfig: func() {
					acls, err := getKafkaACLs(serviceData.Name)
					if err != nil {
						t.Fatal(err)
					}

					for _, acl := range acls.TopicAcl {
						if acl.Username == topicACLDataUpdate.Username && acl.Topic == topicACLDataUpdate.Topic {
							if err := deleteKafkaTopicACL(serviceData.Name, string(acl.ID)); err != nil {
								t.Fatal(err)
							}
						}
					}
				},
				Config: configUpdate,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(topicACLFullResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: func(s *terraform.State) error {
					acls, err := getKafkaACLs(serviceData.Name)
					if err != nil {
						return err
					}

					return checkKafkaTopicACL(acls, s, topicACLFullResourceName, &topicACLDataUpdate)
				},
			},
			{
				// Import
				ResourceName:            topicACLFullResourceName,
				ImportStateIdFunc:       kafkaACLImportStateID(topicACLFullResourceName),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			{
				ResourceName:            schemaRegistryACLFullResourceName,
				ImportStateIdFunc:       kafkaACLImportStateID(schemaRegistryACLFullResourceName),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func kafkaACLImportStateID(resourceFullName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceFullName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceFullName)
		}

		return fmt.Sprintf("%s/%s@%s",
			rs.Primary.Attributes["service"],
			rs.Primary.Attributes["acl_id"],
			rs.Primary.Attributes["zone"],
		), nil
	}
}

func kafkaACLTestClient(ctx context.Context) (*v3.Client, error) {
	client, err := testutils.APIClientV3()
	if err != nil {
		return nil, err
	}

	return utils.SwitchClientZone(ctx, client, testutils.TestZoneName)
}

func getKafkaACLs(service string) (*v3.DBAASKafkaAcls, error) {
	ctx := context.Background()

	client, err := kafkaACLTestClient(ctx)
	if err != nil {
		return nil, err
	}

	return client.GetDBAASKafkaAclConfig(ctx, service)
}

func deleteKafkaTopicACL(service, aclID string) error {
	ctx := context.Background()

	client, err := kafkaACLTestClient(ctx)
	if err != nil {
		return err
	}

	op, err := client.DeleteDBAASKafkaTopicAclConfig(ctx, service, aclID)
	if err != nil {
		return err
	}

	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	return err
}

func checkKafkaTopicACL(acls *v3.DBAASKafkaAcls, s *terraform.State, resourceFullName string, data *TemplateModelKafkaTopicACL) error {
	aclID := s.RootModule().Resources[resourceFullName].Primary.Attributes["acl_id"]

	for _, acl := range acls.TopicAcl {
		if string(acl.ID) != aclID {
			continue
		}

		if acl.Username != data.Username || acl.Topic != data.Topic || string(acl.Permission) != data.Permission {
			return fmt.Errorf("topic ACL entry %q: expected %s/%s/%s, got %s/%s/%s", aclID,
				data.Username, data.Topic, data.Permission, acl.Username, acl.Topic, acl.Permission)
		}

		return nil
	}

	return fmt.Errorf("topic ACL entry %q not found", aclID)
}

func checkKafkaSchemaRegistryACL(acls *v3.DBAASKafkaAcls, s *terraform.State, resourceFullName string, data *TemplateModelKafkaSchemaRegistryACL) error {
	aclID := s.RootModule().Resources[resourceFullName].Primary.Attributes["acl_id"]

	for _, acl := range acls.SchemaRegistryAcl {
		if string(acl.ID) != aclID {
			continue
		}

		if acl.Username != data.Username || acl.Resource != data.Resource || string(acl.Permission) != data.Permission {
			return fmt.Errorf("schema registry ACL entry %q: expected %s/%s/%s, got %s/%s/%s", aclID,
				data.Username, data.Resource, data.Permission, acl.Username, acl.Resource, acl.Permission)
		}

		return nil
	}

	return fmt.Errorf("schema registry ACL entry %q not found", aclID)
}
