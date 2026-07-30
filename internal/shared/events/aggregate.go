package events

type Aggregate struct {
	Type string `json:"type"` //"Order"
	Id   uint   `json:"id"`   //42
}
