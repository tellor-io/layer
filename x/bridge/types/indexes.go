package types

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
)

// ValidatorCheckpointParamsIndexes defines indexes for ValidatorCheckpointParams
type ValidatorCheckpointParamsIndexes struct {
	ByCheckpoint *indexes.Unique[[]byte, uint64, ValidatorCheckpointParams]
}

// IndexesList implements collections.Indexer
func (v ValidatorCheckpointParamsIndexes) IndexesList() []collections.Index[uint64, ValidatorCheckpointParams] {
	return []collections.Index[uint64, ValidatorCheckpointParams]{v.ByCheckpoint}
}

// NewValidatorCheckpointParamsIndexes creates indexes for ValidatorCheckpointParams
func NewValidatorCheckpointParamsIndexes(sb *collections.SchemaBuilder) ValidatorCheckpointParamsIndexes {
	return ValidatorCheckpointParamsIndexes{
		ByCheckpoint: indexes.NewUnique(
			sb, ValidatorCheckpointByCheckpointIndexPrefix, "validator_checkpoint_by_checkpoint",
			collections.BytesKey, collections.Uint64Key,
			func(_ uint64, v ValidatorCheckpointParams) ([]byte, error) {
				return v.Checkpoint, nil
			},
		),
	}
}
