package tools

type JobWorker struct {
	WorkerID string `json:"id"`
	JobType  string `json:"type"`
	Data     string `json:"data"`
}

type Similarity struct {
	UserID     int
	Similarity float64
}
