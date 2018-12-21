package feedback

// Reason is a basic structure of feedback reason
type Reason struct {
	ID      int64  `json:"id"`
	Content string `json:"content"`
	Index   int    `json:"idx"`
}