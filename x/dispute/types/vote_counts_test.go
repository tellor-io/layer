package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/types"
)

func TestVoteCountsAddSubtract(t *testing.T) {
	c := &types.VoteCounts{}

	require.NoError(t, c.Add(types.VoteEnum_VOTE_SUPPORT, 10))
	require.NoError(t, c.Add(types.VoteEnum_VOTE_AGAINST, 20))
	require.NoError(t, c.Add(types.VoteEnum_VOTE_INVALID, 30))
	require.Equal(t, types.VoteCounts{Support: 10, Against: 20, Invalid: 30}, *c)

	require.NoError(t, c.Subtract(types.VoteEnum_VOTE_SUPPORT, 4))
	require.NoError(t, c.Subtract(types.VoteEnum_VOTE_AGAINST, 5))
	require.NoError(t, c.Subtract(types.VoteEnum_VOTE_INVALID, 6))
	require.Equal(t, types.VoteCounts{Support: 6, Against: 15, Invalid: 24}, *c)
}

func TestVoteCountsAddOutOfRange(t *testing.T) {
	c := &types.VoteCounts{}
	err := c.Add(types.VoteEnum(3), 1)
	require.ErrorIs(t, err, types.ErrInvalidVoteChoice)
	require.Equal(t, types.VoteCounts{}, *c)
}

func TestVoteCountsSubtractOutOfRangeDecrementsInvalid(t *testing.T) {
	c := &types.VoteCounts{Invalid: 100}
	require.NoError(t, c.Subtract(types.VoteEnum(3), 30))
	require.Equal(t, uint64(70), c.Invalid)
}

func TestVoteCountsSubtractUnderflow(t *testing.T) {
	c := &types.VoteCounts{Support: 5}
	err := c.Subtract(types.VoteEnum_VOTE_SUPPORT, 6)
	require.ErrorIs(t, err, types.ErrVoteCountUnderflow)
	require.Equal(t, uint64(5), c.Support)
}

func TestVoteCountsAddOverflow(t *testing.T) {
	c := &types.VoteCounts{Support: ^uint64(0)}
	err := c.Add(types.VoteEnum_VOTE_SUPPORT, 1)
	require.ErrorIs(t, err, types.ErrVoteCountOverflow)
	require.Equal(t, ^uint64(0), c.Support)

	c = &types.VoteCounts{Against: ^uint64(0) - 5}
	require.NoError(t, c.Add(types.VoteEnum_VOTE_AGAINST, 5))
	require.Equal(t, ^uint64(0), c.Against)
	err = c.Add(types.VoteEnum_VOTE_AGAINST, 1)
	require.ErrorIs(t, err, types.ErrVoteCountOverflow)
	require.Equal(t, ^uint64(0), c.Against)
}
