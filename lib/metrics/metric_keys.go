// nolint:lll
package metrics

// Metrics Keys Guidelines
// 1. Be wary of length
// 2. Prefix by module
// 3. Suffix keys with a unit of measurement
// 4. Delimit with '_'
// 5. Information such as callback type should be added as tags, not in key names.
// Example: clob_place_order_count, clob_msg_place_order_latency_ms, clob_operations_queue_length
// clob_expired_stateful_orders_count, clob_processed_orders_ms_total

// Clob Metrics Keys
const (
	// Measure Since
	DaemonGetPreviousBlockInfoLatency     = "daemon_get_previous_block_info_latency"
	DaemonGetAllMarketPricesLatency       = "daemon_get_all_market_prices_latency"
	DaemonGetMarketPricesPaginatedLatency = "daemon_get_market_prices_paginated_latency"
	DaemonGetAllPerpetualsLatency         = "daemon_get_all_perpetuals_latency"
	DaemonGetPerpetualsPaginatedLatency   = "daemon_get_perpetuals_paginated_latency"
	MevLatency                            = "mev_latency"
)
