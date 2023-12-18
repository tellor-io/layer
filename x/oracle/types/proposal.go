package types

import (
	"fmt"
	"strings"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	CycleListChange = "CycleListChange"
)

var _ govtypes.Content = &CycleListChangeProposal{}

func init() {
	govtypes.RegisterProposalType(CycleListChange)
}

func (cp *CycleListChangeProposal) GetTitle() string { return cp.Title }

func (cp *CycleListChangeProposal) GetDescription() string { return cp.Description }

func (cp *CycleListChangeProposal) ProposalRoute() string { return RouterKey }

func (cp *CycleListChangeProposal) ProposalType() string { return CycleListChange }

func (cp CycleListChangeProposal) String() string {
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

func (cp *CycleListChangeProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(cp)
	if err != nil {
		return err
	}

	return ValidateChanges(cp.Changes)
}

func ValidateChanges(queryData []CycleListQuery) error {
	if len(queryData) == 0 {
		return fmt.Errorf("submitted query data list is empty")
	}

	return nil
}
