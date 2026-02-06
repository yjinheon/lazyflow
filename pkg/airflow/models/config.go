package models

type AirflowConfigSection struct {
	Section string              `json:"name"`
	Options []AirflowConfigOpt `json:"options"`
}

type AirflowConfigOpt struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AirflowConfigResponse struct {
	Sections []AirflowConfigSection `json:"sections"`
}

type Connection struct {
	ConnId   string `json:"connection_id"`
	ConnType string `json:"conn_type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Schema   string `json:"schema"`
	Login    string `json:"login"`
}

type ConnectionCollection struct {
	Connections  []Connection `json:"connections"`
	TotalEntries int          `json:"total_entries"`
}

type Variable struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description"`
}

type VariableCollection struct {
	Variables    []Variable `json:"variables"`
	TotalEntries int        `json:"total_entries"`
}
