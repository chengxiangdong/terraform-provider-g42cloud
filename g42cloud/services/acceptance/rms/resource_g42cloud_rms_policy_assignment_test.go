package rms

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/chnsz/golangsdk/openstack/rms/v1/policyassignments"

	"github.com/g42cloud-terraform/terraform-provider-g42cloud/g42cloud/services/acceptance"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/config"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/services/rms"
)

func getPolicyAssignmentResourceFunc(cfg *config.Config, state *terraform.ResourceState) (interface{}, error) {
	client, err := cfg.RmsV1Client(acceptance.G42_REGION_NAME)
	if err != nil {
		return nil, fmt.Errorf("error creating RMS v1 client: %s", err)
	}

	return policyassignments.Get(client, acceptance.G42_DOMAIN_ID, state.Primary.ID)
}

// Test the builtin policy (resource type) assignment.
func TestAccPolicyAssignment_basic(t *testing.T) {
	var (
		obj policyassignments.Assignment

		rName       = "g42cloud_rms_policy_assignment.test"
		name        = acceptance.RandomAccResourceNameWithDash()
		basicConfig = testAccPolicyAssignment_ecsConfig(name)
	)

	rc := acceptance.InitResourceCheck(
		rName,
		&obj,
		getPolicyAssignmentResourceFunc,
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acceptance.TestAccPreCheck(t)
			acceptance.TestAccPrecheckDomainId(t)
		},
		ProviderFactories: acceptance.TestAccProviderFactories,
		// Test to delete policy assignment in enabled status.
		CheckDestroy: rc.CheckResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyAssignment_basic(basicConfig, name, "Disabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "type", rms.AssignmentTypeBuiltin),
					resource.TestCheckResourceAttr(rName, "description", "An ECS is noncompliant if its flavor is "+
						"not in the specified flavor list (filter by resource ID)."),
					resource.TestCheckResourceAttr(rName, "name", name),
					resource.TestCheckResourceAttrPair(rName, "policy_definition_id",
						"data.g42cloud_rms_policy_definitions.test", "definitions.0.id"),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.region", acceptance.G42_REGION_NAME),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.resource_provider", "ecs"),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.resource_type", "cloudservers"),
					resource.TestCheckResourceAttrPair(rName, "policy_filter.0.resource_id",
						"g42cloud_compute_instance.test", "id"),
					resource.TestCheckResourceAttr(rName, "status", "Disabled"),
					resource.TestCheckResourceAttrSet(rName, "parameters.listOfAllowedFlavors"),
					resource.TestCheckResourceAttrSet(rName, "created_at"),
					resource.TestCheckResourceAttrSet(rName, "updated_at"),
				),
			},
			{
				Config: testAccPolicyAssignment_basic(basicConfig, name, "Enabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "status", "Enabled"),
				),
			},
			{
				Config: testAccPolicyAssignment_basicUpdate(basicConfig, name, "Enabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "type", rms.AssignmentTypeBuiltin),
					resource.TestCheckResourceAttr(rName, "description", "An ECS is noncompliant if its flavor is "+
						"not in the specified flavor list (filter by resource tag)."),
					resource.TestCheckResourceAttr(rName, "name", name),
					resource.TestCheckResourceAttrPair(rName, "policy_definition_id",
						"data.g42cloud_rms_policy_definitions.test", "definitions.0.id"),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.region", acceptance.G42_REGION_NAME),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.resource_provider", "ecs"),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.resource_type", "cloudservers"),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.tag_key", "foo"),
					resource.TestCheckResourceAttr(rName, "policy_filter.0.tag_value", "bar"),
					resource.TestCheckResourceAttr(rName, "status", "Enabled"),
					resource.TestCheckResourceAttrSet(rName, "parameters.listOfAllowedFlavors"),
					resource.TestCheckResourceAttrSet(rName, "created_at"),
					resource.TestCheckResourceAttrSet(rName, "updated_at"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPolicyAssignment_ecsConfig(name string) string {
	return fmt.Sprintf(`
data "g42cloud_availability_zones" "test" {}

data "g42cloud_compute_flavors" "test" {
  availability_zone = data.g42cloud_availability_zones.test.names[0]
  cpu_core_count    = 2
  memory_size       = 4
}

data "g42cloud_images_images" "test" {
  flavor_id = data.g42cloud_compute_flavors.test.ids[0]

  os         = "Ubuntu"
  visibility = "public"
}

resource "g42cloud_vpc" "test" {
  name = "%[1]s"
  cidr = "192.168.0.0/16"
}

resource "g42cloud_vpc_subnet" "test" {
  name       = "%[1]s"
  vpc_id     = g42cloud_vpc.test.id
  cidr       = cidrsubnet(g42cloud_vpc.test.cidr, 4, 1)
  gateway_ip = cidrhost(cidrsubnet(g42cloud_vpc.test.cidr, 4, 1), 1)
}

resource "g42cloud_networking_secgroup" "test" {
  name                 = "%[1]s"
  delete_default_rules = true
}

resource "g42cloud_compute_keypair" "test" {
  name = "%[1]s"
}

resource "g42cloud_compute_instance" "test" {
  name              = "%[1]s"
  flavor_id         = data.g42cloud_compute_flavors.test.ids[0]
  image_id          = data.g42cloud_images_images.test.images[0].id
  security_groups   = [g42cloud_networking_secgroup.test.name]
  availability_zone = data.g42cloud_availability_zones.test.names[0]
  key_pair          = g42cloud_compute_keypair.test.name

  network {
    uuid = g42cloud_vpc_subnet.test.id
  }

  tags = {
    foo = "bar"
  }
}
`, name)
}

func testAccPolicyAssignment_basic(basicConfig, name, status string) string {
	return fmt.Sprintf(`
%[1]s

data "g42cloud_rms_policy_definitions" "test" {
  name = "allowed-ecs-flavors"
}

resource "g42cloud_rms_policy_assignment" "test" {
  name                 = "%[2]s"
  description          = "An ECS is noncompliant if its flavor is not in the specified flavor list (filter by resource ID)."
  policy_definition_id = try(data.g42cloud_rms_policy_definitions.test.definitions[0].id, "")
  status               = "%[3]s"

  policy_filter {
    region            = "%[4]s"
    resource_provider = "ecs"
    resource_type     = "cloudservers"
    resource_id       = g42cloud_compute_instance.test.id
  }

  parameters = {
    listOfAllowedFlavors = "[\"${data.g42cloud_compute_flavors.test.ids[0]}\"]"
  }
}
`, basicConfig, name, status, acceptance.G42_REGION_NAME)
}

func testAccPolicyAssignment_basicUpdate(basicConfig, name, status string) string {
	return fmt.Sprintf(`
%[1]s

data "g42cloud_rms_policy_definitions" "test" {
  name = "allowed-ecs-flavors"
}

resource "g42cloud_rms_policy_assignment" "test" {
  name                 = "%[2]s"
  description          = "An ECS is noncompliant if its flavor is not in the specified flavor list (filter by resource tag)."
  policy_definition_id = try(data.g42cloud_rms_policy_definitions.test.definitions[0].id, "")
  status               = "%[3]s"

  policy_filter {
    region            = "%[4]s"
    resource_provider = "ecs"
    resource_type     = "cloudservers"
    tag_key           = "foo"
    tag_value         = "bar"
  }

  parameters = {
    listOfAllowedFlavors = "[\"${data.g42cloud_compute_flavors.test.ids[0]}\"]"
  }
}
`, basicConfig, name, status, acceptance.G42_REGION_NAME)
}

// Test the builtin policy (period type) assignment.
func TestAccPolicyAssignment_period(t *testing.T) {
	var (
		obj policyassignments.Assignment

		rName       = "g42cloud_rms_policy_assignment.test"
		name        = acceptance.RandomAccResourceNameWithDash()
		basicConfig = testAccPolicyAssignment_periodConfig(name)
	)

	rc := acceptance.InitResourceCheck(
		rName,
		&obj,
		getPolicyAssignmentResourceFunc,
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acceptance.TestAccPreCheck(t)
			acceptance.TestAccPrecheckDomainId(t)
		},
		ProviderFactories: acceptance.TestAccProviderFactories,
		// Test to delete policy assignment in disabled status.
		CheckDestroy: rc.CheckResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyAssignment_period(basicConfig, name, "Disabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "type", rms.AssignmentTypeBuiltin),
					resource.TestCheckResourceAttr(rName, "description", "An account is noncompliant if none of its "+
						"CTS trackers track specified OBS buckets."),
					resource.TestCheckResourceAttr(rName, "name", name),
					resource.TestCheckResourceAttrPair(rName, "policy_definition_id",
						"data.g42cloud_rms_policy_definitions.test", "definitions.0.id"),
					resource.TestCheckResourceAttr(rName, "period", "One_Hour"),
					resource.TestCheckResourceAttr(rName, "status", "Disabled"),
					resource.TestCheckResourceAttrSet(rName, "parameters.trackBucket"),
					resource.TestCheckResourceAttrSet(rName, "created_at"),
					resource.TestCheckResourceAttrSet(rName, "updated_at"),
				),
			},
			{
				Config: testAccPolicyAssignment_period(basicConfig, name, "Enabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "status", "Enabled"),
				),
			},
			{
				Config: testAccPolicyAssignment_periodUpdate(basicConfig, name, "Enabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "type", rms.AssignmentTypeBuiltin),
					resource.TestCheckResourceAttr(rName, "description", ""),
					resource.TestCheckResourceAttr(rName, "name", name),
					resource.TestCheckResourceAttrPair(rName, "policy_definition_id",
						"data.g42cloud_rms_policy_definitions.test", "definitions.0.id"),
					resource.TestCheckResourceAttr(rName, "period", "Six_Hours"),
					resource.TestCheckResourceAttr(rName, "status", "Enabled"),
					resource.TestCheckResourceAttrSet(rName, "parameters.trackBucket"),
					resource.TestCheckResourceAttrSet(rName, "created_at"),
					resource.TestCheckResourceAttrSet(rName, "updated_at"),
				),
			},
			{
				Config: testAccPolicyAssignment_periodUpdate(basicConfig, name, "Disabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "status", "Disabled"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPolicyAssignment_periodConfig(name string) string {
	return fmt.Sprintf(`
resource "g42cloud_obs_bucket" "complian" {
  bucket        = "%[1]s"
  storage_class = "STANDARD"
  acl           = "private"
  force_destroy = true
}

resource "g42cloud_obs_bucket" "non_complian" {
  bucket        = "%[1]s-non-complian"
  storage_class = "STANDARD"
  acl           = "private"
  force_destroy = true
}

resource "g42cloud_cts_tracker" "test" {
  bucket_name = g42cloud_obs_bucket.complian.bucket
  file_prefix = "cts-updated"
  lts_enabled = false
  enabled     = false
}
`, name)
}

func testAccPolicyAssignment_period(periodConfig, name, status string) string {
	return fmt.Sprintf(`
%[1]s

data "g42cloud_rms_policy_definitions" "test" {
  name = "cts-obs-bucket-track"
}

resource "g42cloud_rms_policy_assignment" "test" {
  name                 = "%[2]s"
  description          = "An account is noncompliant if none of its CTS trackers track specified OBS buckets."
  period               = "One_Hour"
  policy_definition_id = try(data.g42cloud_rms_policy_definitions.test.definitions[0].id, "")
  status               = "%[3]s"

  parameters = {
    trackBucket = "\"${g42cloud_obs_bucket.complian.bucket}\""
  }
}
`, periodConfig, name, status)
}

func testAccPolicyAssignment_periodUpdate(periodConfig, name, status string) string {
	return fmt.Sprintf(`
%[1]s

data "g42cloud_rms_policy_definitions" "test" {
  name = "cts-obs-bucket-track"
}

# Set the description to an empty value.
resource "g42cloud_rms_policy_assignment" "test" {
  name                 = "%[2]s"
  status               = "%[3]s"
  period               = "Six_Hours"
  policy_definition_id = try(data.g42cloud_rms_policy_definitions.test.definitions[0].id, "")

  parameters = {
    trackBucket = "\"${g42cloud_obs_bucket.non_complian.bucket}\""
  }
}
`, periodConfig, name, status)
}

// Test the custom policy assignment.
func TestAccPolicyAssignment_custom(t *testing.T) {
	var (
		obj policyassignments.Assignment

		rName        = "g42cloud_rms_policy_assignment.test"
		name         = acceptance.RandomAccResourceNameWithDash()
		customConfig = testAccPolicyAssignment_customConfig(name)
	)

	rc := acceptance.InitResourceCheck(
		rName,
		&obj,
		getPolicyAssignmentResourceFunc,
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acceptance.TestAccPreCheck(t)
			acceptance.TestAccPrecheckDomainId(t)
		},
		ProviderFactories: acceptance.TestAccProviderFactories,
		CheckDestroy:      rc.CheckResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyAssignment_custom(customConfig, name, "Disabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "type", rms.AssignmentTypeCustom),
					resource.TestCheckResourceAttr(rName, "description", "The ECS instances that do not conform to "+
						"the custom function logic are considered non-compliant"),
					resource.TestCheckResourceAttr(rName, "name", name),
					resource.TestCheckResourceAttr(rName, "status", "Disabled"),
					resource.TestCheckResourceAttr(rName, "parameters.string_test", "\"string_value\""),
					resource.TestCheckResourceAttr(rName, "parameters.array_test", "[\"array_element\"]"),
					resource.TestCheckResourceAttr(rName, "parameters.object_test", "{\"terraform_version\":\"1.xx.x\"}"),
					resource.TestCheckResourceAttrSet(rName, "created_at"),
					resource.TestCheckResourceAttrSet(rName, "updated_at"),
				),
			},
			{
				Config: testAccPolicyAssignment_custom(customConfig, name, "Enabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "status", "Enabled"),
				),
			},
			{
				Config: testAccPolicyAssignment_customUpdate(customConfig, name, "Enabled"),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "name", name),
					resource.TestCheckResourceAttr(rName, "status", "Enabled"),
					resource.TestCheckResourceAttr(rName, "parameters.string_test", "\"update_string_value\""),
					resource.TestCheckResourceAttr(rName, "parameters.update_array_test", "[\"array_element\"]"),
					resource.TestCheckResourceAttr(rName, "parameters.object_test", "{\"update_terraform_version\":\"1.xx.xx\"}"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPolicyAssignment_customConfig(name string) string {
	customConfig := testAccPolicyAssignment_ecsConfig(name)

	return fmt.Sprintf(`
%[1]s

resource "g42cloud_fgs_function" "test" {
  name                  = "%[2]s"
  code_type             = "inline"
  handler               = "index.handler"
  runtime               = "Node.js10.16"
  functiongraph_version = "v2"
  app                   = "default"
  enterprise_project_id = "0"
  memory_size           = 128
  timeout               = 3
}
`, customConfig, name)
}

func testAccPolicyAssignment_custom(customConfig, name, status string) string {
	return fmt.Sprintf(`
%[1]s

resource "g42cloud_rms_policy_assignment" "test" {
  name        = "%[2]s"
  description = "The ECS instances that do not conform to the custom function logic are considered non-compliant"
  status      = "%[3]s"

  custom_policy {
    function_urn = "${g42cloud_fgs_function.test.urn}:${g42cloud_fgs_function.test.version}"
    auth_type    = "agency"
    auth_value   = {
      agency_name = "\"rms_admin_trust\""
    }
  }

  parameters = {
    string_test = "\"string_value\""
    array_test  = "[\"array_element\"]"
    object_test = "{\"terraform_version\":\"1.xx.x\"}"
  }
}
`, customConfig, name, status)
}

func testAccPolicyAssignment_customUpdate(customConfig, name, status string) string {
	return fmt.Sprintf(`
%[1]s

resource "g42cloud_rms_policy_assignment" "test" {
  name        = "%[2]s"
  description = "The ECS instances that do not conform to the custom function logic are considered non-compliant"
  status      = "%[3]s"

  custom_policy {
    function_urn = "${g42cloud_fgs_function.test.urn}:${g42cloud_fgs_function.test.version}"
    auth_type    = "agency"
    auth_value   = {
      agency_name = "\"rms_admin_trust\""
    }
  }

  parameters = {
    string_test       = "\"update_string_value\""
    update_array_test = "[\"array_element\"]"
    object_test       = "{\"update_terraform_version\":\"1.xx.xx\"}"
  }
}
`, customConfig, name, status)
}
