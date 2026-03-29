package filestation

import "time"

// File represents a file in QNAP File Station
type File struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Type       string    `json:"type"`
	MimeType   string    `json:"mimeType"`
	Owner      string    `json:"owner"`
	Group      string    `json:"group"`
	Created    time.Time `json:"created"`
	Modified   time.Time `json:"modified"`
	Permissions string   `json:"permissions"`
	IsFile     bool      `json:"isfile"`
	IsFolder   bool      `json:"isfolder"`
}

// Folder represents a folder in QNAP File Station
type Folder struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Owner       string    `json:"owner"`
	Group       string    `json:"group"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	Permissions string    `json:"permissions"`
	ChildCount  int       `json:"child_count,omitempty"`
}

// ListOptions contains options for listing files/folders
type ListOptions struct {
	Offset    int    `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
	FileType  string `json:"filetype,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
}

// CopyMoveOptions contains options for copy/move operations
type CopyMoveOptions struct {
	Overwrite bool `json:"overwrite,omitempty"`
}

// ShareLink represents a share link
type ShareLink struct {
	ID            string    `json:"id"`
	URL           string    `json:"url"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Expires       time.Time `json:"expires,omitempty"`
	Password      bool      `json:"password_protected"`
	Writeable     bool      `json:"writeable"`
	DownloadCount int       `json:"download_count,omitempty"`
	Created       time.Time `json:"created"`
	Validity      int       `json:"validity,omitempty"`
}

// ShareLinkOptions contains options for creating share links
type ShareLinkOptions struct {
	Expires   time.Time `json:"expires,omitempty"`
	Password  string    `json:"password,omitempty"`
	Writeable bool      `json:"writeable,omitempty"`
	Validity  int       `json:"validity,omitempty"`
}

// SearchOptions contains options for searching
type SearchOptions struct {
	Pattern        string    `json:"pattern,omitempty"`
	FileType       string    `json:"filetype,omitempty"`
	Extension      []string  `json:"extension,omitempty"`
	SizeMin        int64     `json:"size_min,omitempty"`
	SizeMax        int64     `json:"size_max,omitempty"`
	ModifiedBefore time.Time `json:"mtime_before,omitempty"`
	ModifiedAfter  time.Time `json:"mtime_after,omitempty"`
	Recursive      bool      `json:"recursive,omitempty"`
}

// UploadProgress reports upload progress
type UploadProgress struct {
	Total      int64
	Transferred int64
	Percentage float64
}

// DownloadProgress reports download progress
type DownloadProgress struct {
	Total      int64
	Transferred int64
	Percentage float64
}
