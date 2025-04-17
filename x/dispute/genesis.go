package dispute

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	err := k.Params.Set(ctx, genState.Params)
	if err != nil {
		panic(err)
	}
	if genState.Dust.IsNil() {
		err = k.Dust.Set(ctx, math.ZeroInt())
		if err != nil {
			panic(err)
		}
	} else {
		err = k.Dust.Set(ctx, genState.Dust)
		if err != nil {
			panic(err)
		}
	}

	for _, data := range genState.Disputes {
		if err := k.Disputes.Set(ctx, data.DisputeId, *data.Dispute); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.Votes {
		if err := k.Votes.Set(ctx, data.DisputeId, *data.Vote); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.Voter {
		if err := k.Voter.Set(ctx, collections.Join(data.DisputeId, data.VoterAddress), *data.Voter); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.ReportersWithDelegatorsWhoVoted {
		if err := k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(data.ReporterAddress, data.DisputeId), data.VotedAmount); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.BlockInfo {
		if err := k.BlockInfo.Set(ctx, data.HashId, *data.BlockInfo); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.DisputeFeePayer {
		if err := k.DisputeFeePayer.Set(ctx, collections.Join(data.DisputeId, data.Payer), *data.PayerInfo); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.VoteCountsByGroup {
		if err := k.VoteCountsByGroup.Set(ctx, data.DisputeId, types.StakeholderVoteCounts{Users: *data.Users, Reporters: *data.Reporters, Team: *data.Team}); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	params, _ := k.Params.Get(ctx)
	genesis.Params = params

	iterDisputes, err := k.Disputes.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	disputes := make([]*types.DisputeStateEntry, 0)
	votes := make([]*types.VotesStateEntry, 0)
	voters := make([]*types.VoterStateEntry, 0)
	blockInfo := make([]*types.BlockInfoStateEntry, 0)
	disputeFeePayer := make([]*types.DisputeFeePayerStateEntry, 0)
	voteCountsByGroup := make([]*types.VoteCountsByGroupStateEntry, 0)
	reportersWithDelsWhoVoted := make([]*types.ReportersWithDelegatorsWhoVotedStateEntry, 0)

	for ; iterDisputes.Valid(); iterDisputes.Next() {
		dispute_id, err := iterDisputes.Key()
		if err != nil {
			panic(err)
		}

		dispute, err := iterDisputes.Value()
		if err != nil {
			panic(err)
		}
		// only add disputes that are open and in voting status to genesis. The rest will be migrated over separately
		if dispute.Open && dispute.DisputeStatus == types.Voting {
			disputes = append(disputes, &types.DisputeStateEntry{DisputeId: dispute_id, Dispute: &dispute})
			// add votes for this dispute to genesis
			v, err := k.Votes.Get(ctx, dispute.DisputeId)
			if err != nil {
				panic(err)
			}
			votes = append(votes, &types.VotesStateEntry{DisputeId: dispute.DisputeId, Vote: &v})

			// iterate through all voters for this dispute
			voterIter, err := k.Voter.Indexes.VotersById.MatchExact(ctx, dispute.DisputeId)
			if err != nil {
				panic(err)
			}

			keys, err := voterIter.PrimaryKeys()
			if err != nil {
				panic(err)
			}
			for _, key := range keys {
				voter, err := k.Voter.Get(ctx, key)
				if err != nil {
					panic(err)
				}
				voters = append(voters, &types.VoterStateEntry{DisputeId: key.K1(), VoterAddress: key.K2(), Voter: &voter})
			}
			err = voterIter.Close()
			if err != nil {
				panic(err)
			}

			// get block info for this dispute
			block, err := k.BlockInfo.Get(ctx, dispute.HashId)
			if err != nil {
				panic(err)
			}
			blockInfo = append(blockInfo, &types.BlockInfoStateEntry{HashId: dispute.HashId, BlockInfo: &block})

			// add the Dispute Fee Payers for this dispute
			rng := collections.NewPrefixedPairRange[uint64, []byte](dispute.DisputeId)
			iterDisputeFeePayer, err := k.DisputeFeePayer.Iterate(ctx, rng)
			if err != nil {
				panic(err)
			}
			for ; iterDisputeFeePayer.Valid(); iterDisputeFeePayer.Next() {
				keys, err := iterDisputeFeePayer.Key()
				if err != nil {
					panic(err)
				}
				payer := keys.K2()

				payerInfo, err := iterDisputeFeePayer.Value()
				if err != nil {
					panic(err)
				}
				disputeFeePayer = append(disputeFeePayer, &types.DisputeFeePayerStateEntry{DisputeId: dispute.DisputeId, Payer: payer, PayerInfo: &payerInfo})
			}

			// add the Vote Counts By Group for this dispute
			voteCountByGroup, err := k.VoteCountsByGroup.Get(ctx, dispute.DisputeId)
			if err != nil {
				panic(err)
			}
			voteCountsByGroup = append(voteCountsByGroup, &types.VoteCountsByGroupStateEntry{DisputeId: dispute.DisputeId, Users: &voteCountByGroup.Users, Reporters: &voteCountByGroup.Reporters, Team: &voteCountByGroup.Team})

			rngReportersDel := (&collections.PairRange[[]byte, uint64]{}).
				StartInclusive(uint64(3)).
				EndInclusive(uint64(4))
			iterReportersDelVoted, err := k.ReportersWithDelegatorsVotedBefore.Iterate(ctx, rngReportersDel)
			if err != nil {
				panic(err)
			}
			for ; iterReportersDelVoted.Valid(); iterReportersDelVoted.Next() {
				keys, err := iterReportersDelVoted.Key()
				if err != nil {
					panic(err)
				}
				reporterAddr := keys.K1()
				dispute_id := keys.K2()
				votedAmt, err := iterReportersDelVoted.Value()
				if err != nil {
					panic(err)
				}
				reportersWithDelsWhoVoted = append(reportersWithDelsWhoVoted, &types.ReportersWithDelegatorsWhoVotedStateEntry{ReporterAddress: reporterAddr, DisputeId: dispute_id, VotedAmount: votedAmt})
			}
			err = iterReportersDelVoted.Close()
			if err != nil {
				panic(err)
			}
		}
	}
	err = iterDisputes.Close()
	if err != nil {
		panic(err)
	}

	genesis.Disputes = disputes
	genesis.BlockInfo = blockInfo
	genesis.Votes = votes
	genesis.Voter = voters
	genesis.DisputeFeePayer = disputeFeePayer
	genesis.VoteCountsByGroup = voteCountsByGroup
	genesis.ReportersWithDelegatorsWhoVoted = reportersWithDelsWhoVoted

	Dust, err := k.Dust.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Dust = Dust

	// write all module data to second file to persist without making genesis file massive
	fmt.Println("Exporting module data")
	ExportModuleData(ctx, k)
	fmt.Println("Module data exported")
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}

func calculateFileChecksum(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type ModuleStateWriter struct {
	encoder       *json.Encoder
	file          *os.File
	first         bool
	tempFilename  string
	finalFilename string
}

func NewModuleStateWriter(filename string) (*ModuleStateWriter, error) {
	// Create a temporary file first
	tempFile := filename + ".temp"
	file, err := os.Create(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write the opening structure
	if _, err := file.Write([]byte("{\n")); err != nil {
		fmt.Println("Failed to write opening structure")
		file.Close()
		return nil, err
	}

	return &ModuleStateWriter{
		encoder:       json.NewEncoder(file),
		file:          file,
		first:         true,
		tempFilename:  tempFile,
		finalFilename: filename,
	}, nil
}

// For array fields
func (w *ModuleStateWriter) StartArraySection(name string) error {
	if !w.first {
		if _, err := w.file.Write([]byte(",\n")); err != nil {
			return err
		}
	}
	w.first = false

	// Write the field name and opening bracket
	_, err := w.file.Write([]byte(fmt.Sprintf("\"%s\": [\n", name)))
	return err
}

func (w *ModuleStateWriter) WriteArrayItem(item interface{}, isFirst bool) error {
	if !isFirst {
		if _, err := w.file.Write([]byte(",\n")); err != nil {
			return err
		}
	}
	return w.encoder.Encode(item)
}

func (w *ModuleStateWriter) EndArraySection() error {
	_, err := w.file.Write([]byte("\n]"))
	return err
}

// For single value fields
func (w *ModuleStateWriter) WriteValue(name string, value interface{}) error {
	if !w.first {
		if _, err := w.file.Write([]byte(",\n")); err != nil {
			return err
		}
	}
	w.first = false

	// Write the field name
	if _, err := w.file.Write([]byte(fmt.Sprintf("\"%s\": ", name))); err != nil {
		return err
	}

	// Encode the value
	return w.encoder.Encode(value)
}

func (w *ModuleStateWriter) Close() {
	// Only close the file if it hasn't been closed yet
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			panic(err)
		}
		w.file = nil
	}

	// Calculate checksum of the temporary file
	checksum, err := calculateFileChecksum(w.tempFilename)
	if err != nil {
		panic(err)
	}

	// Read the entire temporary file
	content, err := os.ReadFile(w.tempFilename)
	if err != nil {
		panic(err)
	}

	// Create the final file
	finalFile, err := os.Create(w.finalFilename)
	if err != nil {
		panic(err)
	}
	defer finalFile.Close()

	// Remove the final closing brace from the content
	content = content[:len(content)-2]

	// Write the original content without the final brace
	if _, err := finalFile.Write(content); err != nil {
		panic(err)
	}

	// Add the checksum and close the JSON object
	if _, err := finalFile.Write([]byte(fmt.Sprintf(",\n\"checksum\": \"%s\"\n}", checksum))); err != nil {
		panic(err)
	}

	// Remove the temporary file
	if err := os.Remove(w.tempFilename); err != nil {
		panic(err)
	}
}

type ModuleStateData struct {
	Disputes                        []*types.DisputeStateEntry                         `json:"disputes"`                            // stores dispute state entries
	Votes                           []*types.VotesStateEntry                           `json:"votes"`                               // stores vote state entries
	Voters                          []*types.VoterStateEntry                           `json:"voters"`                              // stores voter state entries
	ReportersWithDelegatorsWhoVoted []*types.ReportersWithDelegatorsWhoVotedStateEntry `json:"reporters_with_delegators_who_voted"` // stores reporters with delegators who voted
	BlockInfo                       []*types.BlockInfoStateEntry                       `json:"block_info"`                          // stores block info state entries
	DisputeFeePayer                 []*types.DisputeFeePayerStateEntry                 `json:"dispute_fee_payer"`                   // stores dispute fee payer entries
	Dust                            math.Int                                           `json:"dust"`                                // stores dust
}

func ExportModuleData(ctx sdk.Context, k keeper.Keeper) {
	writer, err := NewModuleStateWriter("dispute_module_state.json")
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	iterDisputes, err := k.Disputes.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterDisputes.Close()

	isFirst := true
	writer.StartArraySection("disputes")
	for ; iterDisputes.Valid(); iterDisputes.Next() {
		dispute_id, err := iterDisputes.Key()
		if err != nil {
			panic(err)
		}

		dispute, err := iterDisputes.Value()
		if err != nil {
			panic(err)
		}

		writer.WriteArrayItem(&types.DisputeStateEntry{DisputeId: dispute_id, Dispute: &dispute}, isFirst)
		isFirst = false
	}
	writer.EndArraySection()

	iterVotes, err := k.Votes.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterVotes.Close()

	isFirst = true
	writer.StartArraySection("votes")
	for ; iterVotes.Valid(); iterVotes.Next() {
		dispute_id, err := iterVotes.Key()
		if err != nil {
			panic(err)
		}

		vote, err := iterVotes.Value()
		if err != nil {
			panic(err)
		}
		writer.WriteArrayItem(&types.VotesStateEntry{DisputeId: dispute_id, Vote: &vote}, isFirst)
		isFirst = false
	}
	writer.EndArraySection()

	iterVoter, err := k.Voter.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterVoter.Close()

	isFirst = true
	writer.StartArraySection("voters")
	for ; iterVoter.Valid(); iterVoter.Next() {
		key, err := iterVoter.Key()
		if err != nil {
			panic(err)
		}
		dispute_id := key.K1()
		voterAddr := key.K2()

		voter, err := iterVoter.Value()
		if err != nil {
			panic(err)
		}
		writer.WriteArrayItem(&types.VoterStateEntry{DisputeId: dispute_id, VoterAddress: voterAddr, Voter: &voter}, isFirst)
		isFirst = false
	}
	writer.EndArraySection()

	iterReportersDelVoted, err := k.ReportersWithDelegatorsVotedBefore.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterReportersDelVoted.Close()

	isFirst = true
	writer.StartArraySection("reporters_with_delegators_who_voted")
	for ; iterReportersDelVoted.Valid(); iterReportersDelVoted.Next() {
		key, err := iterReportersDelVoted.Key()
		if err != nil {
			panic(err)
		}
		reporterAddr := key.K1()
		dispute_id := key.K2()

		votedAmt, err := iterReportersDelVoted.Value()
		if err != nil {
			panic(err)
		}
		writer.WriteArrayItem(&types.ReportersWithDelegatorsWhoVotedStateEntry{ReporterAddress: reporterAddr, DisputeId: dispute_id, VotedAmount: votedAmt}, isFirst)
		isFirst = false
	}
	writer.EndArraySection()

	iterBlockInfo, err := k.BlockInfo.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterBlockInfo.Close()

	isFirst = true
	writer.StartArraySection("block_info")
	for ; iterBlockInfo.Valid(); iterBlockInfo.Next() {
		hash_id, err := iterBlockInfo.Key()
		if err != nil {
			panic(err)
		}

		info, err := iterBlockInfo.Value()
		if err != nil {
			panic(err)
		}
		writer.WriteArrayItem(&types.BlockInfoStateEntry{HashId: hash_id, BlockInfo: &info}, isFirst)
		isFirst = false
	}
	writer.EndArraySection()

	iterDisputeFeePayer, err := k.DisputeFeePayer.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterDisputeFeePayer.Close()

	isFirst = true
	writer.StartArraySection("dispute_fee_payer")
	for ; iterDisputeFeePayer.Valid(); iterDisputeFeePayer.Next() {
		keys, err := iterDisputeFeePayer.Key()
		if err != nil {
			panic(err)
		}
		dispute_id := keys.K1()
		payer := keys.K2()

		payerInfo, err := iterDisputeFeePayer.Value()
		if err != nil {
			panic(err)
		}
		writer.WriteArrayItem(&types.DisputeFeePayerStateEntry{DisputeId: dispute_id, Payer: payer, PayerInfo: &payerInfo}, isFirst)
		isFirst = false
	}
	writer.EndArraySection()

	Dust, err := k.Dust.Get(ctx)
	if err != nil {
		panic(err)
	}
	writer.WriteValue("dust", Dust)

	iterVoteCountsByGroup, err := k.VoteCountsByGroup.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterVoteCountsByGroup.Close()

	isFirst = true
	writer.StartArraySection("vote_counts_by_group")
	for ; iterVoteCountsByGroup.Valid(); iterVoteCountsByGroup.Next() {
		dispute_id, err := iterVoteCountsByGroup.Key()
		if err != nil {
			panic(err)
		}

		voteCount, err := iterVoteCountsByGroup.Value()
		if err != nil {
			panic(err)
		}
		writer.WriteArrayItem(&types.VoteCountsByGroupStateEntry{DisputeId: dispute_id, Users: &voteCount.Users, Reporters: &voteCount.Reporters, Team: &voteCount.Team}, isFirst)
		isFirst = false
	}
	writer.EndArraySection()

	writer.Close()
}
