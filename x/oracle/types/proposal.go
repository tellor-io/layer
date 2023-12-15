package types

import (
	"fmt"
	"strings"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	SupportedQueryChange = "SupportedQueryChange"
)

var _ govtypes.Content = &SupportedQueryChangeProposal{}

func init() {
	govtypes.RegisterProposalType(SupportedQueryChange)
}

func (cp *SupportedQueryChangeProposal) GetTitle() string { return cp.Title }

func (cp *SupportedQueryChangeProposal) GetDescription() string { return cp.Description }

func (cp *SupportedQueryChangeProposal) ProposalRoute() string { return RouterKey }

func (cp *SupportedQueryChangeProposal) ProposalType() string { return SupportedQueryChange }

func (cp SupportedQueryChangeProposal) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, `Parameter Change Proposal:
  Title:       %s
  Description: %s
  Changes:
`, cp.Title, cp.Description)

	for _, q := range cp.Changes {
		fmt.Fprintf(&b, `    Supported queries Change:
      Query data: %s
`, q.QueryData)
	}

	return b.String()
}

func (cp *SupportedQueryChangeProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(cp)
	if err != nil {
		return err
	}

	return ValidateChanges(cp.Changes)
}

func ValidateChanges(queryData []QueryChange) error {
	if len(queryData) == 0 {
		return fmt.Errorf("submitted query data list is empty")
	}

	return nil
}
