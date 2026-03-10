package templates

import (
	"bytes"
	"strings"
	"text/template"

	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

type EBSCSIDriverTemplateData struct {
	AWSArnPrefix string
	Region       string
	ProviderID   string
	OIDCDomain   string
	STSDomain    string
}

type NodeInstanceRoleTemplateData struct {
	AWSArnPrefix string
	EC2Service   string
}

func getAWSDNSSuffix(region string) string {
	if p, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region); ok {
		return p.DNSSuffix()
	}
	return endpoints.AwsPartition().DNSSuffix()
}

func getOIDCDomain(region string, ipFamily *string) string {
	if p, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region); ok {
		if IsIPv6(ipFamily) {
			ep, err := p.EndpointFor("iam", region,
				func(o *endpoints.Options) {
					o.UseDualStackEndpoint =
						endpoints.DualStackEndpointStateEnabled
				})
			if err == nil {
				dnsSuffix := strings.TrimPrefix(
					ep.URL,
					"https://iam."+region+".",
				)
				return "oidc-eks." + region + "." + dnsSuffix
			}
		}
		return "oidc.eks." + region + "." + p.DNSSuffix()
	}
	return "oidc.eks." + region + ".amazonaws.com"
}

func getEC2ServiceEndpoint(region string) string {
	return "ec2." + getAWSDNSSuffix(region)
}

func getArnPrefixForRegion(region string) string {
	if p, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region); ok {
		return "arn:" + p.ID()
	}
	return "arn:" + endpoints.AwsPartition().ID()
}

func GetServiceRoleTemplate(region string) (string, error) {
	tmpl, err := template.New("serviceRole").Parse(ServiceRoleTemplate)
	if err != nil {
		return "", err
	}

	// Create the data for the template
	data := struct {
		AWSArnPrefix string
	}{
		AWSArnPrefix: getArnPrefixForRegion(region),
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GetNodeInstanceRoleTemplate(region string, ipFamily *string) (string, error) {
	nodeInstanceRoleTmpl := NodeInstanceRoleTemplate
	if IsIPv6(ipFamily) {
		nodeInstanceRoleTmpl = NodeInstanceRoleIPv6Template
	}

	tmpl, err := template.New("nodeInstanceRole").Parse(nodeInstanceRoleTmpl)
	if err != nil {
		return "", err
	}

	// Create the data for the template
	data := NodeInstanceRoleTemplateData{
		AWSArnPrefix: getArnPrefixForRegion(region),
		EC2Service:   getEC2ServiceEndpoint(region),
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GetEBSCSIDriverTemplate(region string, providerID string, ipFamily *string) (string, error) {
	tmpl, err := template.New("ebsrole").Parse(EBSCSIDriverTemplate)
	if err != nil {
		return "", err
	}

	data := EBSCSIDriverTemplateData{
		AWSArnPrefix: getArnPrefixForRegion(region),
		OIDCDomain:   getOIDCDomain(region, ipFamily),
		STSDomain:    getAWSDNSSuffix(region),
		Region:       region,
		ProviderID:   providerID,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func IsIPv6(ipFamily *string) bool {
	return ipFamily != nil && strings.EqualFold(*ipFamily, string(ekstypes.IpFamilyIpv6))
}
