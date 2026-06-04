package domain

type Protocol string

const (
	ProtocolSMB    Protocol = "smb"
	ProtocolWebDAV Protocol = "webdav"
	ProtocolFTP    Protocol = "ftp"
)

type ConnectionInput struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name"`
	Protocol     Protocol `json:"protocol"`
	Host         string   `json:"host,omitempty"`
	Port         int      `json:"port,omitempty"`
	BaseURL      string   `json:"baseUrl,omitempty"`
	Share        string   `json:"share,omitempty"`
	RootPath     string   `json:"rootPath,omitempty"`
	Username     string   `json:"username,omitempty"`
	Domain       string   `json:"domain,omitempty"`
	PassiveMode  bool     `json:"passiveMode,omitempty"`
	Password     string   `json:"password,omitempty"`
	SavePassword bool     `json:"savePassword,omitempty"`
}

type ConnectionProfile struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Protocol      Protocol `json:"protocol"`
	Host          string   `json:"host,omitempty"`
	Port          int      `json:"port,omitempty"`
	BaseURL       string   `json:"baseUrl,omitempty"`
	Share         string   `json:"share,omitempty"`
	RootPath      string   `json:"rootPath,omitempty"`
	Username      string   `json:"username,omitempty"`
	Domain        string   `json:"domain,omitempty"`
	PassiveMode   bool     `json:"passiveMode,omitempty"`
	CredentialKey string   `json:"credentialKey,omitempty"`
	PasswordSaved bool     `json:"passwordSaved"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type EntryType string

const (
	EntryFile      EntryType = "file"
	EntryDirectory EntryType = "directory"
)

type RemoteEntry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       EntryType `json:"type"`
	Size       int64     `json:"size"`
	ModifiedAt string    `json:"modifiedAt"`
}

type TransferDirection string

const (
	TransferUpload   TransferDirection = "upload"
	TransferDownload TransferDirection = "download"
)

type TransferStatus string

const (
	TransferQueued    TransferStatus = "queued"
	TransferRunning   TransferStatus = "running"
	TransferSucceeded TransferStatus = "succeeded"
	TransferFailed    TransferStatus = "failed"
	TransferCanceled  TransferStatus = "canceled"
)

type TransferTask struct {
	ID           string            `json:"id"`
	ConnectionID string            `json:"connectionId"`
	Direction    TransferDirection `json:"direction"`
	Source       string            `json:"source"`
	Destination  string            `json:"destination"`
	Status       TransferStatus    `json:"status"`
	BytesDone    int64             `json:"bytesDone"`
	BytesTotal   int64             `json:"bytesTotal"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
	CreatedAt    string            `json:"createdAt"`
	StartedAt    *string           `json:"startedAt,omitempty"`
	FinishedAt   *string           `json:"finishedAt,omitempty"`
}
