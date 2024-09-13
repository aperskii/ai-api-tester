package main

type RequestParam struct {
	Params interface{} `json:"params"`
	Route  string      `json:"route"`
	Type   string      `json:"type"`
}

type ResponseData struct {
	Blocks Blocks `json:"blocks"`
	Photo  struct {
		Properties map[string]interface{} `json:"properties"`
	} `json:"photo"`
}

type Blocks struct {
	Photo struct {
		Properties map[string]interface{} `json:"properties"`
	} `json:"photo,omitempty"`
	Text struct {
		Text string `json:"text"`
	} `json:"text,omitempty"`
	Qr2015Bvg struct {
		Text string `json:"text"`
	} `json:"qr2015_bvg,omitempty"`
}
