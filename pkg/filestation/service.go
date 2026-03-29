package filestation

import (

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// FileStationService handles QNAP File Station API operations
type FileStationService struct {
	client *api.Client
}

// NewFileStationService creates a new FileStationService
func NewFileStationService(client *api.Client) *FileStationService {
	return &FileStationService{client: client}
}
