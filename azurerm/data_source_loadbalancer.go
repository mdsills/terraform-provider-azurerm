package azurerm

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func dataSourceArmLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceArmLoadBalancerRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": resourceGroupNameSchema(),

			"location": locationForDataSourceSchema(),

			"sku": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"frontend_ip_configuration": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"private_ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"public_ip_address_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"private_ip_address_allocation": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"load_balancer_rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"inbound_nat_rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"zones": zonesSchemaComputed(),
					},
				},
			},

			"private_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_ip_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"tags": tagsForDataSourceSchema(),
		},
	}
}

func dataSourceArmLoadBalancerRead(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	client := meta.(*ArmClient).loadBalancerClient
	ctx := meta.(*ArmClient).StopContext

	resp, err := client.Get(ctx, resourceGroup, name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return fmt.Errorf("Load Balancer %q was not found in Resource Group %q", name, resourceGroup)
		}

		return fmt.Errorf("Error retrieving Load Balancer %s: %s", name, err)
	}

	d.SetId(*resp.ID)

	if location := resp.Location; location != nil {
		d.Set("location", azureRMNormalizeLocation(*location))
	}

	if sku := resp.Sku; sku != nil {
		d.Set("sku", string(sku.Name))
	}

	if props := resp.LoadBalancerPropertiesFormat; props != nil {
		if feipConfigs := props.FrontendIPConfigurations; feipConfigs != nil {
			if err := d.Set("frontend_ip_configuration", flattenLoadBalancerDataSourceFrontendIpConfiguration(feipConfigs)); err != nil {
				return fmt.Errorf("Error flattening `frontend_ip_configuration`: %+v", err)
			}

			privateIpAddress := ""
			privateIpAddresses := make([]string, 0)
			for _, config := range *feipConfigs {
				if feipProps := config.FrontendIPConfigurationPropertiesFormat; feipProps != nil {
					if ip := feipProps.PrivateIPAddress; ip != nil {
						if privateIpAddress == "" {
							privateIpAddress = *feipProps.PrivateIPAddress
						}

						privateIpAddresses = append(privateIpAddresses, *feipProps.PrivateIPAddress)
					}
				}
			}

			d.Set("private_ip_address", privateIpAddress)
			d.Set("private_ip_addresses", privateIpAddresses)
		}
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func flattenLoadBalancerDataSourceFrontendIpConfiguration(ipConfigs *[]network.FrontendIPConfiguration) []interface{} {
	result := make([]interface{}, 0)
	if ipConfigs == nil {
		return result
	}

	for _, config := range *ipConfigs {
		ipConfig := make(map[string]interface{})
		if config.Name != nil {
			ipConfig["name"] = *config.Name
		}

		zones := make([]string, 0)
		if zs := config.Zones; zs != nil {
			zones = *zs
		}
		ipConfig["zones"] = zones

		if props := config.FrontendIPConfigurationPropertiesFormat; props != nil {
			ipConfig["private_ip_address_allocation"] = props.PrivateIPAllocationMethod

			if subnet := props.Subnet; subnet != nil {
				ipConfig["subnet_id"] = *subnet.ID
			}

			if pip := props.PrivateIPAddress; pip != nil {
				ipConfig["private_ip_address"] = *pip
			}

			if pip := props.PublicIPAddress; pip != nil {
				ipConfig["public_ip_address_id"] = *pip.ID
			}

			loadBalancingRules := make([]interface{}, 0)
			if rules := props.LoadBalancingRules; rules != nil {
				for _, rule := range *rules {
					loadBalancingRules = append(loadBalancingRules, *rule.ID)
				}

			}
			ipConfig["load_balancer_rules"] = schema.NewSet(schema.HashString, loadBalancingRules)

			inboundNatRules := make([]interface{}, 0)
			if rules := props.InboundNatRules; rules != nil {
				for _, rule := range *rules {
					inboundNatRules = append(inboundNatRules, *rule.ID)
				}
			}
			ipConfig["inbound_nat_rules"] = schema.NewSet(schema.HashString, inboundNatRules)
		}

		result = append(result, ipConfig)
	}
	return result
}
