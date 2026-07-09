package aws

type BatchError struct {
	MessageId string
	Code      string
	Message   string
}

type BatchSendResult struct {
	SuccessfullIds []string
	Failed         []BatchError
}
