package aws

type BatchError struct {
	Id      string
	Code    string
	Message string
}

type BatchResult struct {
	SuccessfulIds []string
	Failed        []BatchError
}

type QueueStats struct {
	ApproximateMessages           int64 // Messages available for retrieval
	ApproximateMessagesNotVisible int64 // Messages being processed
	ApproximateMessagesDelayed    int64 // Messages waiting for delay to expire
}
