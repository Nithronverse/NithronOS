package pools

import "time"

type TxStep struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Cmd         string     `json:"cmd"`
	Destructive bool       `json:"destructive"`
	Status      string     `json:"status"` // pending|running|ok|error
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
	Err         string     `json:"err,omitempty"`
}

type Tx struct {
	ID         string     `json:"id"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Steps      []TxStep   `json:"steps"`
	OK         bool       `json:"ok"`
	Error      string     `json:"error,omitempty"`
}
