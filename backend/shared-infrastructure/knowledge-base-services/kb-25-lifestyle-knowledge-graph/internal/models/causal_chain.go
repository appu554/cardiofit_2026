package models

type CausalChain struct {
	Source        string           `json:"source"`
	SourceType    string           `json:"source_type"`
	Target        string           `json:"target"`
	Components    []ChainComponent `json:"components"`
	NetEffect     EffectDescriptor `json:"net_effect"`
	PathLength    int              `json:"path_length"`
	EvidenceGrade string           `json:"evidence_grade"`
}

type ChainComponent struct {
	FromNode string           `json:"from_node"`
	FromType string           `json:"from_type"`
	EdgeType string           `json:"edge_type"`
	ToNode   string           `json:"to_node"`
	ToType   string           `json:"to_type"`
	Effect   EffectDescriptor `json:"effect"`
}
