package filestation

import (
	"fmt"
	"time"
)

// File represents a file in QNAP File Station
type File struct {
	FileName  string `json:"filename"`       // File name
	IsFolder  int    `json:"isfolder"`       // 0=file, 1=folder
	FileSize  string `json:"filesize"`       // File size in bytes (string)
	Owner     string `json:"owner"`          // Owner name
	Group     string `json:"group"`          // Group name
	Privilege string `json:"privilege"`      // File permissions
	MT        string `json:"mt"`             // Modified time
	EpochMT   int64  `json:"epochmt"`        // Epoch modified time
	Exist     int    `json:"exist"`          // File exists
	FileType  int    `json:"filetype"`       // File type
	Path      string `json:"path,omitempty"` // File path
}

// Name returns the file name (for compatibility)
func (f *File) Name() string {
	return f.FileName
}

// Size returns the file size as int64
func (f *File) Size() int64 {
	if f.FileSize == "" {
		return 0
	}
	var size int64
	_, _ = fmt.Sscanf(f.FileSize, "%d", &size)
	return size
}

// IsDirectory returns true if this is a folder
func (f *File) IsDirectory() bool {
	return f.IsFolder == 1
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
	Mode      int  `json:"mode,omitempty"` // 0=overwrite, 1=skip, 2=auto rename
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
	Total       int64
	Transferred int64
	Percentage  float64
}

// DownloadProgress reports download progress
type DownloadProgress struct {
	Total       int64
	Transferred int64
	Percentage  float64
}

// CompressOptions contains options for compression operations
type CompressOptions struct {
	SourceFiles  []string // List of source files to compress
	SourcePath   string   // Source directory path
	CompressName string   // Name of the compressed archive
	Level        int      // Compression level (optional)
}

// ExtractOptions contains options for extraction operations
type ExtractOptions struct {
	ExtractFile string // Archive file to extract
	DestPath    string // Destination path for extraction
	CodePage    string // Character encoding (e.g., "utf8", "cp936")
	Overwrite   bool   // Overwrite existing files
}

// CompressStatus represents the status of a compression operation
type CompressStatus struct {
	PID        string  `json:"pid"`             // Process ID
	Status     string  `json:"status"`          // Status: running, finished, failed
	Progress   float64 `json:"progress"`        // Progress percentage (0-100)
	SourcePath string  `json:"source_path"`     // Source file path
	DestPath   string  `json:"dest_path"`       // Destination archive path
	FileSize   int64   `json:"file_size"`       // Total file size
	Processed  int64   `json:"processed"`       // Processed bytes
	Error      string  `json:"error,omitempty"` // Error message if failed
}

// ExtractStatus represents the status of an extraction operation
type ExtractStatus struct {
	PID         string  `json:"pid"`             // Process ID
	Status      string  `json:"status"`          // Status: running, finished, failed
	Progress    float64 `json:"progress"`        // Progress percentage (0-100)
	ExtractFile string  `json:"extract_file"`    // Archive being extracted
	DestPath    string  `json:"dest_path"`       // Destination path
	FileCount   int     `json:"file_count"`      // Total file count
	Processed   int     `json:"processed"`       // Processed files
	Error       string  `json:"error,omitempty"` // Error message if failed
}

// ExtractFile represents a file in an archive
type ExtractFile struct {
	FileName string `json:"filename"` // File name
	FileSize int64  `json:"filesize"` // File size in bytes
	IsFolder int    `json:"isfolder"` // 0=file, 1=folder
}

// ACL (Access Control List) Types

// ACLEntry represents a single ACL entry
type ACLEntry struct {
	User   string `json:"user"`   // User or group name
	Domain string `json:"domain"` // Domain (e.g., "local", "domain")
	IsUser bool   `json:"isuser"` // True for user, false for group
	Right  string `json:"right"`  // Access rights (e.g., "r", "w", "rw", "full", "none")
}

// SetACLOptions contains options for setting ACL control
type SetACLOptions struct {
	ShareName string     `json:"sharename"` // Share name
	Root      string     `json:"root"`      // Root path within share
	Recursive bool       `json:"recursive"` // Apply recursively
	ACLs      []ACLEntry `json:"acls"`      // ACL entries
}

// ACLControl represents ACL control settings
type ACLControl struct {
	Enabled     bool       `json:"enabled"`     // ACL enabled
	Recursive   bool       `json:"recursive"`   // Recursive ACL
	ACLs        []ACLEntry `json:"acls"`        // ACL entries
	Propagation string     `json:"propagation"` // Propagation mode
}

// ACLControlResponse represents the response from get_acl_control
type ACLControlResponse struct {
	Status int        `json:"status"`
	Data   ACLControl `json:"data"`
}

// ACLUser represents a user in ACL user/group list
type ACLUser struct {
	Name     string `json:"name"`     // User name
	Domain   string `json:"domain"`   // Domain
	FullName string `json:"fullname"` // Full display name
	IsAdmin  bool   `json:"isadmin"`  // Is admin user
}

// ACLGroup represents a group in ACL user/group list
type ACLGroup struct {
	Name   string `json:"name"`   // Group name
	Domain string `json:"domain"` // Domain
	Desc   string `json:"desc"`   // Group description
}

// ACLUserGroupList represents list of users and groups
type ACLUserGroupList struct {
	Users  []ACLUser  `json:"users"`  // List of users
	Groups []ACLGroup `json:"groups"` // List of groups
}

// ACLUserGroupListResponse represents the response from get_acl_user_group_list_out
type ACLUserGroupListResponse struct {
	Status int              `json:"status"`
	Data   ACLUserGroupList `json:"data"`
}

// PrivilegeEntry represents a privilege entry
type PrivilegeEntry struct {
	User   string `json:"user"`   // User or group name
	Domain string `json:"domain"` // Domain (e.g., "local", "domain")
	IsUser bool   `json:"isuser"` // True for user, false for group
	Right  string `json:"right"`  // Access rights
	IsFile bool   `json:"isfile"` // True for file, false for folder
}

// SetPrivilegeOptions contains options for setting privileges
type SetPrivilegeOptions struct {
	ShareName  string           `json:"sharename"`  // Share name
	Path       string           `json:"path"`       // Path within share
	Recursive  bool             `json:"recursive"`  // Apply recursively
	Privileges []PrivilegeEntry `json:"privileges"` // Privilege entries
}

// AccessRight represents access rights for a file/folder
type AccessRight struct {
	Path       string     `json:"path"`        // File/folder path
	IsFolder   bool       `json:"isfolder"`    // True for folder, false for file
	Owner      string     `json:"owner"`       // Owner name
	Group      string     `json:"group"`       // Group name
	Permission string     `json:"permission"`  // Permission string (e.g., "rwxr-xr-x")
	ACLEnabled bool       `json:"acl_enabled"` // ACL enabled
	ACLs       []ACLEntry `json:"acls"`        // ACL entries if enabled
}

// AccessRightResponse represents the response from get_access_right
type AccessRightResponse struct {
	Status int         `json:"status"`
	Data   AccessRight `json:"data"`
}
