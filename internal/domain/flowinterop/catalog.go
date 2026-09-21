package flowinterop

import "github.com/matiasleandrokruk/fenix/internal/domain/tool"

// ContractVersion is the Fenix-facing semantic contract for Salesforce Flow interoperability.
const ContractVersion = "1"

// Operation identifies one semantic Flow interoperability capability.
type Operation string

const (
	OperationImport   Operation = "salesforce.flow.import"
	OperationExport   Operation = "salesforce.flow.export"
	OperationValidate Operation = "salesforce.flow.validate"
	OperationCompare  Operation = "salesforce.flow.compare"
)

// ArtifactFormat identifies an artifact representation without importing Mermaid2SF FlowIR types into Fenix.
type ArtifactFormat string

const (
	FormatMermaid           ArtifactFormat = "mermaid"
	FormatSalesforceFlowXML ArtifactFormat = "salesforce_flow_xml"
	FormatFlowIRV2          ArtifactFormat = "flowir_v2"
)

// CapabilitySpec binds a semantic operation to the W1 governed capability boundary.
type CapabilitySpec struct {
	Descriptor    tool.CapabilityDescriptor
	InputFormats  []ArtifactFormat
	OutputFormats []ArtifactFormat
}

// Catalog returns only capabilities backed by Mermaid2SF's current production semantic paths.
// Round-trip remains correctness evidence rather than a runtime capability.
func Catalog() []CapabilitySpec {
	return []CapabilitySpec{
		{
			Descriptor: tool.CapabilityDescriptor{
				Name:            string(OperationImport),
				Version:         ContractVersion,
				Operation:       "import",
				SideEffectClass: tool.SideEffectTransform,
			},
			InputFormats:  []ArtifactFormat{FormatSalesforceFlowXML},
			OutputFormats: []ArtifactFormat{FormatFlowIRV2, FormatMermaid},
		},
		{
			Descriptor: tool.CapabilityDescriptor{
				Name:            string(OperationExport),
				Version:         ContractVersion,
				Operation:       "export",
				SideEffectClass: tool.SideEffectTransform,
			},
			InputFormats:  []ArtifactFormat{FormatMermaid, FormatFlowIRV2},
			OutputFormats: []ArtifactFormat{FormatSalesforceFlowXML, FormatFlowIRV2},
		},
		{
			Descriptor: tool.CapabilityDescriptor{
				Name:            string(OperationValidate),
				Version:         ContractVersion,
				Operation:       "validate",
				SideEffectClass: tool.SideEffectVerify,
			},
			InputFormats:  []ArtifactFormat{FormatMermaid, FormatFlowIRV2},
			OutputFormats: nil,
		},
		{
			Descriptor: tool.CapabilityDescriptor{
				Name:            string(OperationCompare),
				Version:         ContractVersion,
				Operation:       "compare",
				SideEffectClass: tool.SideEffectVerify,
			},
			InputFormats: []ArtifactFormat{
				FormatMermaid,
				FormatSalesforceFlowXML,
				FormatFlowIRV2,
			},
			OutputFormats: nil,
		},
	}
}

// Lookup returns the semantic capability contract for an operation.
func Lookup(operation Operation) (CapabilitySpec, bool) {
	for _, spec := range Catalog() {
		if spec.Descriptor.Name == string(operation) {
			return spec, true
		}
	}
	return CapabilitySpec{}, false
}
