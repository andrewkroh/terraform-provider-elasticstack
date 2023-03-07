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

func TestAccResourceAgentPolicy(t *testing.T) {
	// Generate a random agent policy name because Fleet requires
	// policy names to be unique.
	agentPolicySuffix := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		//PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             checkResourceFleetAgentPolicyDestroy,
		ProtoV5ProviderFactories: acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceKibanaFleetAgentPolicyCreate(agentPolicySuffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_kibana_fleet_agent_policy.test", "name", "terraform test "+agentPolicySuffix),
				),
			},
		},
	})
}

func testAccResourceKibanaFleetAgentPolicyCreate(agentPolicySuffix string) string {
	return fmt.Sprintf(`
provider "elasticstack" {
  kibana {}
}

resource "elasticstack_kibana_fleet_agent_policy" "test" {
	name        = "terraform test %s"
    namespace   = "default"
    description = "Custom policy for Agents."
}

`, agentPolicySuffix)
}

func checkResourceFleetAgentPolicyDestroy(s *terraform.State) error {
	client, err := clients.NewAcceptanceTestingClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticstack_kibana_fleet_agent_policy" {
			continue
		}

		packagePolicy, diag := fleet.ReadAgentPolicy(context.Background(), client.GetKibanaClient(), rs.Primary.ID)
		if diag.HasError() {
			return fmt.Errorf(diag[0].Summary)
		}

		if packagePolicy != nil {
			return fmt.Errorf("agent policy id=%v still exists, but it should have been removed", rs.Primary.ID)
		}
	}
	return nil
}
