package model

const (
	MinAgentProfit     = 500.0
	MaxAgentProfit     = 2000.0
	DefaultAgentProfit = MinAgentProfit
)

type AgentProductPricing struct {
	ProductID     string  `json:"product_id"`
	Operator      string  `json:"operator"`
	Nominal       float64 `json:"nominal"`
	ProductName   string  `json:"product_name"`
	DefaultProfit float64 `json:"default_profit"`
}
