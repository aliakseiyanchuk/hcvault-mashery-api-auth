package mashery

type APIV2QueryRequest struct {
	Method string `json:"method"`
	Query  string `json:"query"`
}

type APICertificatePinnigRequest struct {
	CommonName   string `json:"cn,omitempty"`
	SerialNumber string `json:"sn,omitempty"`
	Fingerprint  string `json:"fp,omitempty"`
}

type APICreateRoleRequest struct {
	AreaID   string `json:"area_id,omitempty"`
	AreaNID  int    `json:"area_nid,omitempty"`
	ApiKey   string `json:"api_key,omitempty"`
	Secret   string `json:"secret,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	QPS      int    `json:"qps,omitempty"`
}

type APIRoleDataExportRequest struct {
	PEM             string `json:"pem,omitempty"`
	ExplicitTerm    string `json:"explicit_term,omitempty"`
	ExplicitNumUses int    `json:"explicit_num_uses,omitempty"`
	V2Only          bool   `json:"v2_only,omitempty"`
	V3Only          bool   `json:"v3_only,omitempty"`
	ExplicitQPS     int    `json:"explicit_qps,omitempty"`
	ForceProxyMode  bool   `json:"force_proxy_mode,omitempty"`
}

type APIConfigRequest struct {
	OAEPLabel              string `json:"oaep_label,omitempty"`
	ProxyServer            string `json:"proxy_server,omitempty"`
	ProxyServerAuth        string `json:"proxy_server_auth,omitempty"`
	ProxyServerCredentials string `json:"proxy_server_creds,omitempty"`
	CLIWriteEnabled        *bool  `json:"enable_cli_v3_write,omitempty"`
	NetworkLatency         string `json:"net_latency,omitempty"`
	TLSPinning             string `json:"tls_pinning,omitempty"`
}

type APIRoleDataImportRequest struct {
	PEM string `json:"pem,omitempty"`
}
