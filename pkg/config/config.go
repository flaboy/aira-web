package config

type FrameworkConfig struct {
	AiraTablePreifix string `cfg:"AIRA_TABLE_PREFIX" default:"ar_"`
	FrontURL         string `cfg:"FRONT_URL" default:"http://localhost:3000/"`
}

var Config *FrameworkConfig
