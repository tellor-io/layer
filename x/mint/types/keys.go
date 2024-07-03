package types

const (
	// ModuleName is the name of the mint module.
	ModuleName = "mint"

	// StoreKey is the default store key for mint
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = StoreKey

	TimeBasedRewards = "time_based_rewards"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

func MinterKey() []byte {
	return KeyPrefix("Minter")
}
