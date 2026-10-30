package model

type AgentDashboardSummary struct {
	Balance              float64 `json:"balance"`
	TransactionCount     int64   `json:"transaction_count"`
	SuccessfulCount      int64   `json:"successful_transaction_count"`
	Profit               float64 `json:"profit"`
	BalanceDebited       float64 `json:"balance_debited"`
	MostFrequentOperator string  `json:"most_frequent_operator,omitempty"`
}
