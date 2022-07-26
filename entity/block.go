package entity

type BlockExtend struct {
	Slug      []interface{} `json:"slug,omitempty"`
	Title     []interface{} `json:"title,omitempty"`
	StartDate []interface{} `json:"NX\\Q,omitempty"`
	Tags      []interface{}
	Type      []interface{}
}

type BlockConvert struct {
	Type []string
}
