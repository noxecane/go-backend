package rest

type session struct {
	Workspace   uint   `json:"workspace"`
	User        uint   `json:"user"`
	Role        string `json:"role"`
	CompanyName string `json:"company_name"`
	SessionKey  string `json:"session_key"`
	FullName    string `json:"full_name"`
}
