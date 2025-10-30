package metrics

// Keep the metric fields alphabetized within each category.
const (
	// Common.
	AppVersion       = "app_version"
	AppInfo          = "app_info"
	BlockHeight      = "block_height"
	Count            = "count"
	Detail           = "detail"
	Deterministic    = "deterministic"
	Distribution     = "distribution"
	Error            = "error"
	GitCommit        = "git_commit"
	HttpGet5xx       = "http_get_5xx"
	HttpGetHangup    = "http_get_hangup"
	HttpGetRequest   = "http_get_request"
	HttpGetResponse  = "http_get_response"
	HttpGetTimeout   = "http_get_timeout"
	Invalid          = "invalid"
	Latency          = "latency"
	Matched          = "matched"
	MessageType      = "message_type"
	Msg              = "msg"
	Negative         = "negative"
	NonDeterministic = "non_deterministic"
	Positive         = "positive"
	Reason           = "reason"
	Received         = "received"
	Rejected         = "rejected"
	SampleRate       = "sample_rate"
	Success          = "success"
	Valid            = "valid"
	ValidateBasic    = "validate_basic"
	CheckTx          = "check_tx"
	ReCheckTx        = "recheck_tx"
	DeliverTx        = "deliver_tx"
	ProcessProposal  = "process_proposal"

	// Common (Daemons).
	MainTaskLoop = "main_task_loop"

	// ABCI: Prepare / Process
	ConsensusRound     = "consensus_round"
	DisallowMsg        = "disallow_msg"
	Decode             = "decode"
	FundingTx          = "funding_tx"
	GetTxsInOrder      = "get_txs_in_order"
	Handler            = "handler"
	NumOtherTxs        = "num_other_txs"
	OperationsTx       = "operations_tx"
	OriginalNumTxs     = "original_num_txs"
	OtherTxs           = "other_txs"
	RemoveDisallowMsgs = "remove_disallow_msgs"
	PrepareProposalTxs = "prepare_proposal_txs"
	PrepareCheckState  = "prepare_check_state"
	PricesTx           = "prices_tx"
	TotalNumBytes      = "total_num_bytes"
	TotalNumTxs        = "total_num_txs"
	Validate           = "validate"

	RateLimit = "rate_limit"

	// Daemon
	DaemonServer    = "daemon_server"
	ValidResponse   = "valid_response"
	MissingResponse = "missing_response"

	// Epochs.
	EpochInfoName = "epoch_name"
	EpochNumber   = "epoch_number"
	IsEpochOne    = "is_epoch_one"

	// Block Time.
	BlockTimeMs = "block_time_ms"

	// Prices.
	CreateOracleMarket                           = "create_oracle_market"
	CurrentMarketPrices                          = "current_market_prices"
	GetValidMarketPriceUpdates                   = "get_valid_market_price_updates"
	IndexPriceDoesNotExist                       = "index_price_does_not_exist"
	IndexPriceIsZero                             = "index_price_is_zero"
	IndexPriceNotAccurate                        = "index_price_not_accurate"
	IndexPriceNotAvailForAccuracyCheck           = "index_price_not_available_for_accuracy_check"
	LastPriceUpdateForMarketBlock                = "last_price_update_for_market_block"
	MissingPriceUpdates                          = "missing_price_updates"
	NumMarketPricesToUpdate                      = "num_market_prices_to_update"
	PriceChangeRate                              = "price_change_rate"
	ProposedPriceChangesPriceUpdateDecision      = "proposed_price_changes_price_update_decision"
	ProposedPriceCrossesOraclePrice              = "proposed_price_crosses_oracle_price"
	ProposedPriceDoesNotMeetMinPriceChange       = "proposed_price_does_not_meet_min_price_change"
	RecentSmoothedPriceDoesNotMeetMinPriceChange = "recent_smoothed_price_doesnt_meet_min_price_change"
	RecentSmoothedPriceCrossesOraclePrice        = "recent_smoothed_price_crosses_old_price"
	StatefulPriceUpdateValidation                = "stateful_price_update_validation"
	UpdateMarketParam                            = "update_market_param"
	UpdateMarketPrices                           = "update_market_prices"
	UpdateSmoothedPrices                         = "update_smoothed_prices"

	// Pricefeed Daemon.
	Exchange                                = "exchange"
	ExchangeQueryHandlerApiRequest          = "exchange_query_handler_api_request"
	ExchangeSpecificError                   = "exchange_specific_error"
	GetAllPrices_MarketIdToPrice            = "get_all_prices_market_id_to_price"
	PriceEncoderUpdatePrice                 = "price_encoder_update_price"
	PricefeedDaemon                         = "pricefeed_daemon"
	ConfiguredMarketCount                   = "configured_market_count"
	ConfiguredMarketCountPerExchange        = "configured_market_count_per_exchange"
	ConfiguredExchangeCountPerMarket        = "configured_exchange_count_per_market"
	MarketUpdaterGetAllMarketParams         = "market_updater_get_all_market_params"
	MarketUpdaterApplyMarketUpdates         = "market_updater_apply_market_updates"
	MarketUpdaterUpdateMarkets              = "market_updater_update_markets"
	PriceEncoderPriceConversion             = "price_encoder_price_conversion"
	PriceFetcherQueryExchange               = "price_fetcher_query_exchange"
	PriceFetcherQueryForMarket              = "price_fetcher_query_for_market_sampled"
	PriceFetcherSubtaskLoop                 = "price_fetcher_subtask_loop"
	PriceFetcherSubtaskLoopAndSetCtxTimeout = "price_fetcher_subtask_loop_and_set_ctx_timeout"
	PriceUpdateCount                        = "price_update_count"
	PriceUpdaterSendPrices                  = "price_updater_send_prices"
	PriceUpdaterTaskLoop                    = "price_updater_task_loop"
	PriceUpdaterTransformPrices             = "price_updater_transform_prices"
	PriceUpdaterZeroPrices                  = "price_updater_zero_prices"

	// Pricefeed Server.
	GetValidPrices                = "get_valid_prices"
	ValidPrices                   = "valid_prices"
	NoMarketPrice                 = "no_market_price"
	NoValidMedianPrice            = "no_valid_median_price"
	PricefeedServer               = "pricefeed_server"
	PricefeedServerUpdatePrices   = "pricefeed_server_update_prices"
	PricefeedServerValidatePrices = "pricefeed_server_validate_prices"
	PriceIsInvalid                = "price_is_invalid"

	// Shared Pricefeed Server and Daemon.
	UpdatePrice = "update_price"

	// msgsender
	MessageSendSuccess    = "message_send_success"
	MessageSendError      = "message_send_error"
	SendOffchainData      = "send_offchain_data"
	SendOnchainData       = "send_onchain_data"
	OnchainMessageLength  = "onchain_message_length"
	OffchainMessageLength = "offchain_message_length"

	// Indexer events.
	TotalNumIndexerBlockEvents = "total_num_block_events"
	TotalNumIndexerTxnEvents   = "total_num_txn_events"
)

const (
	LatencyMetricSampleRate    = 0.01
	AvailableMarketsSampleRate = .1
)
