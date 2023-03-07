package fleet_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/internal/acctest"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibana/fleet"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourcePackagePolicy(t *testing.T) {
	// Generate a random agent policy name because Fleet requires
	// policy names to be unique.
	agentPolicySuffix := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		//PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             checkResourceFleetPackagePolicyDestroy,
		ProtoV5ProviderFactories: acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceKibanaFleetPackagePolicyCreate(agentPolicySuffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_agent_policy.test", "name", "terraform test "+agentPolicySuffix),
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_package_policy.test", "name", "Windows Security log"),
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_package_policy.test", "namespace", "default"),
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_package_policy.test", "package.0.name", "winlog"),
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_package_policy.test", "package.0.version", "1.10.0"),
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_package_policy.test", "description", "Collect event logs from the Security channel."),

					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_package_policy.test", "input.0.policy_template", "winlogs"),
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_package_policy.test", "input.0.enabled", "true"),
					func(state *terraform.State) error {
						t.Log("TEST_STATE\n", state.RootModule().String())
						return nil
					},
				),
			},
		},
	})
}

func testAccResourceKibanaFleetPackagePolicyCreate(agentPolicySuffix string) string {
	return fmt.Sprintf(`
provider "elasticstack" {
  kibana {}
}

resource "elasticstack_kibana_fleet_agent_policy" "test" {
	name        = "terraform test %s"
    namespace   = "default"
    description = "Collect windows event logs."
}

resource "elasticstack_kibana_fleet_package_policy" "test" {
  name            = "Windows Security log"
  agent_policy_id = elasticstack_kibana_fleet_agent_policy.test.id
  description     = "Collect event logs from the Security channel."

  package {
    name    = "winlog"
    version = "1.10.0"
  }

  input {
    policy_template = "winlogs"
    type            = "httpjson"

    vars_json = jsonencode({
      url = "https://server.example.com:8089"
    })

    stream {
      data_stream = "winlog"
      vars_json = jsonencode({
        interval = "10s"
        search   = "search sourcetype=\"XmlWinEventLog:ChannelName\""
      })
    }
  }

  input {
    policy_template = "winlogs"
    type            = "winlog"

    stream {
      data_stream = "winlog"
      vars_json = jsonencode({
        channel                 = "Security"
        "data_stream.dataset"   = "winlog.security"
        preserve_original_event = false
        ignore_older            = "72h"
        language                = 0
        event_id                = "4624,4625"
        tags                    = ["security"]
      })
    }
  }
}
`, agentPolicySuffix)
}

func checkResourceFleetPackagePolicyDestroy(s *terraform.State) error {
	client, err := clients.NewAcceptanceTestingClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticstack_kibana_fleet_package_policy" {
			continue
		}

		packagePolicy, diag := fleet.ReadPackagePolicy(context.Background(), client.GetKibanaClient(), rs.Primary.ID)
		if diag.HasError() {
			return fmt.Errorf(diag[0].Summary)
		}

		if packagePolicy != nil {
			return fmt.Errorf("package policy id=%v still exists, but it should have been removed", rs.Primary.ID)
		}
	}
	return nil
}
