package cmd

type KV struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

type BucketData struct {
	Name     string `json:"name"`
	Error    string `json:"error,omitempty"`
	LsmSize  int64  `json:"lsmSize"`
	VlogSize int64  `json:"VlogSize"`
}
