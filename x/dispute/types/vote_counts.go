package types

func (c *VoteCounts) bucket(choice VoteEnum) *uint64 {
	switch choice {
	case VoteEnum_VOTE_SUPPORT:
		return &c.Support
	case VoteEnum_VOTE_AGAINST:
		return &c.Against
	case VoteEnum_VOTE_INVALID:
		return &c.Invalid
	default:
		return nil
	}
}

// Add increments the bucket for choice. Unknown choices are rejected.
func (c *VoteCounts) Add(choice VoteEnum, amount uint64) error {
	b := c.bucket(choice)
	if b == nil {
		return ErrInvalidVoteChoice
	}
	*b += amount
	return nil
}

// Subtract decrements the bucket for choice. Out-of-range choices decrement
// Invalid, matching pre-upgrade Vote=3 parking.
func (c *VoteCounts) Subtract(choice VoteEnum, amount uint64) error {
	b := c.bucket(choice)
	if b == nil {
		b = &c.Invalid
	}
	if *b < amount {
		return ErrVoteCountUnderflow
	}
	*b -= amount
	return nil
}
