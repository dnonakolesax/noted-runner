package model

type KernelMessage struct {
	KernelID string `json:"kernel_id"`
	BlockID  string `json:"block_id"`
	Result   string `json:"result"`
	Fail     bool   `json:"fail"`
}
