package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	cluster1Name                = acctest.RandomWithPrefix(testPrefix + "-cluster")
	cluster2Name                = acctest.RandomWithPrefix(testPrefix + "-cluster-2")
	affinityGroupName           = acctest.RandomWithPrefix(testPrefix + "-affinity-group")
	securityGroupName           = acctest.RandomWithPrefix(testPrefix + "-security-group")
	nodepool1Name               = acctest.RandomWithPrefix(testPrefix + "-nodepool")
	nodepool2Name               = acctest.RandomWithPrefix(testPrefix + "-nodepool-2")
	testAccSKSDataSourcesConfig = fmt.Sprintf(`
locals {
  my_zone = %q
}

data "exoscale_security_group" "default" {
  name = "default"
}

resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone = local.my_zone
  name = %q
  labels = {
    "customer" = "your-telecom"
  }
}

resource "exoscale_sks_cluster" "my_sks_cluster_2" {
  zone = local.my_zone
  name = %q
  labels = {
    "customer" = "your-telecom"
  }
}

resource "exoscale_anti_affinity_group" "my_sks_anti_affinity_group" {
  name = %q
}

resource "exoscale_security_group" "my_sks_security_group" {
  name = %q
}

resource "exoscale_security_group_rule" "kubelet" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "Kubelet"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 10250
  end_port          = 10250
  # (beetwen worker nodes only)
  user_security_group_id = exoscale_security_group.my_sks_security_group.id
}

resource "exoscale_security_group_rule" "calico_vxlan" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "VXLAN (Calico)"
  type              = "INGRESS"
  protocol          = "UDP"
  start_port        = 4789
  end_port          = 4789
  user_security_group_id = exoscale_security_group.my_sks_security_group.id
}

resource "exoscale_security_group_rule" "nodeport_tcp" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "Nodeport TCP services"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 30000
  end_port          = 32767
  cidr = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "nodeport_udp" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "Nodeport UDP services"
  type              = "INGRESS"
  protocol          = "UDP"
  start_port        = 30000
  end_port          = 32767
  cidr = "0.0.0.0/0"
}

resource "exoscale_sks_nodepool" "my_sks_nodepool" {
  zone       = local.my_zone
  cluster_id = exoscale_sks_cluster.my_sks_cluster.id
  name       = %q

  instance_type = "standard.medium"
  size          = 3

  anti_affinity_group_ids = [
    exoscale_anti_affinity_group.my_sks_anti_affinity_group.id,
  ]
  security_group_ids = [
    data.exoscale_security_group.default.id,
    resource.exoscale_security_group.my_sks_security_group.id,
  ]
}

resource "exoscale_sks_nodepool" "my_sks_nodepool_2" {
  zone       = local.my_zone
  cluster_id = exoscale_sks_cluster.my_sks_cluster_2.id
  name       = %q

  instance_type = "standard.medium"
  size          = 3

  anti_affinity_group_ids = [
    exoscale_anti_affinity_group.my_sks_anti_affinity_group.id,
  ]
  security_group_ids = [
    data.exoscale_security_group.default.id,
    resource.exoscale_security_group.my_sks_security_group.id,
  ]
}
`, testZoneName, cluster1Name, cluster2Name, affinityGroupName, securityGroupName, nodepool1Name, nodepool2Name)
)

func TestAccSKSDataSources(t *testing.T) {
	type testCase struct {
		Config               string
		DataSourceIdentifier string
		DataSourceName       string
		Attributes           testAttrs
	}

	zone := testZoneName
	dsId := dsSKSClusterIdentifier
	dsName := "my_cluster_ds"
	testCases := []testCase{
		{
			Config: fmt.Sprintf(`
				data %q %q {
				  zone = %q
				  name = exoscale_sks_cluster.my_sks_cluster.name
				}
				`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"name": validateString(cluster1Name),
			},
		},
		{
			Config: fmt.Sprintf(`
		data %q %q {
		  zone = %q
		  id = exoscale_sks_cluster.my_sks_cluster.id
		}
		`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"name": validateString(cluster1Name),
			},
		},
	}

	dsId = dsSKSClustersListIdentifier
	dsName = "my_cluster_list"
	testCases = append(testCases, []testCase{
		{
			Config: fmt.Sprintf(`
		data %q %q {
		  zone = %q
		  name = %q
		}
		`, dsId, dsName, zone, cluster1Name),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"clusters.#":      validateString("1"),
				"clusters.0.name": validateString(cluster1Name),
			},
		},
		{
			Config: fmt.Sprintf(`
		data %q %q {
		  zone = %q
		  labels = {
		    "customer" = "/.*telecom.*/"
		}
		}
		`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"clusters.#": validateString("2"),
			},
		},
	}...,
	)

	dsId = dsSKSNodepoolsListIdentifier
	dsName = "my_nodepool_list"
	testCases = append(testCases, []testCase{
		{
			Config: fmt.Sprintf(`
		data %q %q {
		  zone = %q
		  size = 3
		  name = "/.*nodepool-2/"
		}
		`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"nodepools.#":      validateString("1"),
				"nodepools.0.name": validateString(nodepool2Name),
			},
		},
		{
			Config: fmt.Sprintf(`
		data %q %q {
		  zone = %q
		  name = "/.*-nodepool.*/"
		}
		`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"nodepools.#": validateString("2"),
			},
		},
	}...)

	resTC := resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSKSDataSourcesConfig,
			},
		},
	}

	for _, c := range testCases {
		resTC.Steps = append(resTC.Steps, resource.TestStep{
			Config: fmt.Sprintf(`
%s

%s`, testAccSKSDataSourcesConfig, c.Config),
			Check: resource.ComposeTestCheckFunc(
				testAccSKSDataSourcesAttributes("data."+c.DataSourceIdentifier+"."+c.DataSourceName, c.Attributes)),
		})
	}

	resource.Test(t, resTC)
}

func testAccSKSDataSourcesAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("data source not found in the state")
	}
}
