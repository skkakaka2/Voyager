package filesystem

import (
	"fmt"

	"voyager/internal/domain"
	ftpfs "voyager/internal/filesystem/ftp"
	smbfs "voyager/internal/filesystem/smb"
	webdavfs "voyager/internal/filesystem/webdav"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (Factory) New(input domain.ConnectionInput) (Adapter, error) {
	switch input.Protocol {
	case domain.ProtocolWebDAV:
		return webdavfs.NewAdapter(input), nil
	case domain.ProtocolFTP:
		return ftpfs.NewAdapter(input), nil
	case domain.ProtocolSMB:
		return smbfs.NewAdapter(input), nil
	default:
		return nil, fmt.Errorf("protocol adapter is not implemented: %s", input.Protocol)
	}
}
