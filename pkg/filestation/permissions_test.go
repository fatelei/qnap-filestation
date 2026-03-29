package filestation

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// TestSetACLControl_Success tests successful ACL control setting
func TestSetACLControl_Success(t *testing.T) {
	tests := []struct {
		name     string
		options  *SetACLOptions
		response string
		wantErr  bool
		verify   func(*testing.T, *http.Request)
	}{
		{
			name: "set ACL with single user",
			options: &SetACLOptions{
				ShareName: "public",
				ACLs: []ACLEntry{
					{
						User:   "admin",
						Domain: "local",
						IsUser: true,
						Right:  "full",
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("acl_0_user") != "admin" {
					t.Errorf("expected acl_0_user=admin, got %s", r.URL.Query().Get("acl_0_user"))
				}
				if r.URL.Query().Get("acl_0_isuser") != "1" {
					t.Errorf("expected acl_0_isuser=1, got %s", r.URL.Query().Get("acl_0_isuser"))
				}
				if r.URL.Query().Get("acl_0_right") != "full" {
					t.Errorf("expected acl_0_right=full, got %s", r.URL.Query().Get("acl_0_right"))
				}
			},
		},
		{
			name: "set ACL with group",
			options: &SetACLOptions{
				ShareName: "public",
				ACLs: []ACLEntry{
					{
						User:   "everyone",
						Domain: "local",
						IsUser: false,
						Right:  "r",
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("acl_0_user") != "everyone" {
					t.Errorf("expected acl_0_user=everyone, got %s", r.URL.Query().Get("acl_0_user"))
				}
				if r.URL.Query().Get("acl_0_isuser") != "0" {
					t.Errorf("expected acl_0_isuser=0, got %s", r.URL.Query().Get("acl_0_isuser"))
				}
			},
		},
		{
			name: "set ACL with multiple entries",
			options: &SetACLOptions{
				ShareName: "public",
				ACLs: []ACLEntry{
					{
						User:   "admin",
						Domain: "local",
						IsUser: true,
						Right:  "full",
					},
					{
						User:   "user1",
						Domain: "local",
						IsUser: true,
						Right:  "rw",
					},
					{
						User:   "everyone",
						Domain: "local",
						IsUser: false,
						Right:  "r",
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				// Verify all three ACL entries are present
				if r.URL.Query().Get("acl_0_user") != "admin" {
					t.Errorf("expected acl_0_user=admin, got %s", r.URL.Query().Get("acl_0_user"))
				}
				if r.URL.Query().Get("acl_1_user") != "user1" {
					t.Errorf("expected acl_1_user=user1, got %s", r.URL.Query().Get("acl_1_user"))
				}
				if r.URL.Query().Get("acl_2_user") != "everyone" {
					t.Errorf("expected acl_2_user=everyone, got %s", r.URL.Query().Get("acl_2_user"))
				}
			},
		},
		{
			name: "set ACL with root path",
			options: &SetACLOptions{
				ShareName: "public",
				Root:      "/documents",
				ACLs: []ACLEntry{
					{
						User:   "admin",
						Domain: "local",
						IsUser: true,
						Right:  "full",
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("root") != "/documents" {
					t.Errorf("expected root=/documents, got %s", r.URL.Query().Get("root"))
				}
			},
		},
		{
			name: "set ACL with recursive flag",
			options: &SetACLOptions{
				ShareName: "public",
				Recursive: true,
				ACLs: []ACLEntry{
					{
						User:   "admin",
						Domain: "local",
						IsUser: true,
						Right:  "full",
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("recursive") != "1" {
					t.Errorf("expected recursive=1, got %s", r.URL.Query().Get("recursive"))
				}
			},
		},
		{
			name: "set ACL with domain user",
			options: &SetACLOptions{
				ShareName: "public",
				ACLs: []ACLEntry{
					{
						User:   "domain.user",
						Domain: "corp.example.com",
						IsUser: true,
						Right:  "rw",
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("acl_0_domain") != "corp.example.com" {
					t.Errorf("expected acl_0_domain=corp.example.com, got %s", r.URL.Query().Get("acl_0_domain"))
				}
			},
		},
		{
			name: "set ACL with various rights",
			options: &SetACLOptions{
				ShareName: "public",
				ACLs: []ACLEntry{
					{User: "user1", Domain: "local", IsUser: true, Right: "r"},
					{User: "user2", Domain: "local", IsUser: true, Right: "w"},
					{User: "user3", Domain: "local", IsUser: true, Right: "rw"},
					{User: "user4", Domain: "local", IsUser: true, Right: "full"},
					{User: "user5", Domain: "local", IsUser: true, Right: "none"},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "set_acl_control" {
					t.Errorf("expected func=set_acl_control, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("sid") != "test-sid-12345" {
					t.Errorf("expected sid=test-sid-12345, got %s", r.URL.Query().Get("sid"))
				}

				if r.URL.Query().Get("sharename") != tt.options.ShareName {
					t.Errorf("expected sharename=%s, got %s", tt.options.ShareName, r.URL.Query().Get("sharename"))
				}

				if tt.verify != nil {
					tt.verify(t, r)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.SetACLControl(context.Background(), tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetACLControl() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSetACLControl_AuthError tests authentication errors
func TestSetACLControl_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "success": "true"}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	err := fs.SetACLControl(context.Background(), &SetACLOptions{
		ShareName: "public",
		ACLs: []ACLEntry{
			{User: "admin", Domain: "local", IsUser: true, Right: "full"},
		},
	})

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestSetACLControl_APIErrors tests API error responses
func TestSetACLControl_APIErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "success": "false"}`,
			wantErr:  true,
			errMsg:   "failed to set ACL control",
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "ACL not supported"}`,
			wantErr:  true,
			errMsg:   "failed to set ACL control",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse set ACL response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.SetACLControl(context.Background(), &SetACLOptions{
				ShareName: "public",
				ACLs: []ACLEntry{
					{User: "admin", Domain: "local", IsUser: true, Right: "full"},
				},
			})

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestGetACLControl_Success tests successful ACL control retrieval
func TestGetACLControl_Success(t *testing.T) {
	tests := []struct {
		name      string
		shareName string
		root      string
		response  string
		want      *ACLControl
		wantErr   bool
	}{
		{
			name:      "get ACL control for share",
			shareName: "public",
			root:      "",
			response: `{
				"status": 1,
				"data": {
					"enabled": true,
					"recursive": false,
					"propagation": "inherit",
					"acls": [
						{
							"user": "admin",
							"domain": "local",
							"isuser": true,
							"right": "full"
						},
						{
							"user": "everyone",
							"domain": "local",
							"isuser": false,
							"right": "r"
						}
					]
				}
			}`,
			want: &ACLControl{
				Enabled:     true,
				Recursive:   false,
				Propagation: "inherit",
				ACLs: []ACLEntry{
					{User: "admin", Domain: "local", IsUser: true, Right: "full"},
					{User: "everyone", Domain: "local", IsUser: false, Right: "r"},
				},
			},
			wantErr: false,
		},
		{
			name:      "get ACL control with root path",
			shareName: "public",
			root:      "/documents",
			response: `{
				"status": 1,
				"data": {
					"enabled": true,
					"recursive": true,
					"propagation": "override",
					"acls": [
						{
							"user": "user1",
							"domain": "local",
							"isuser": true,
							"right": "rw"
						}
					]
				}
			}`,
			want: &ACLControl{
				Enabled:     true,
				Recursive:   true,
				Propagation: "override",
				ACLs: []ACLEntry{
					{User: "user1", Domain: "local", IsUser: true, Right: "rw"},
				},
			},
			wantErr: false,
		},
		{
			name:      "get ACL control when disabled",
			shareName: "public",
			root:      "",
			response: `{
				"status": 1,
				"data": {
					"enabled": false,
					"recursive": false,
					"propagation": "",
					"acls": []
				}
			}`,
			want: &ACLControl{
				Enabled:     false,
				Recursive:   false,
				Propagation: "",
				ACLs:        []ACLEntry{},
			},
			wantErr: false,
		},
		{
			name:      "get ACL control with no root specified",
			shareName: "private",
			root:      "",
			response: `{
				"status": 1,
				"data": {
					"enabled": true,
					"recursive": false,
					"propagation": "inherit",
					"acls": [
						{
							"user": "admin",
							"domain": "local",
							"isuser": true,
							"right": "full"
						}
					]
				}
			}`,
			want: &ACLControl{
				Enabled:     true,
				Recursive:   false,
				Propagation: "inherit",
				ACLs: []ACLEntry{
					{User: "admin", Domain: "local", IsUser: true, Right: "full"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "get_acl_control" {
					t.Errorf("expected func=get_acl_control, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("sharename") != tt.shareName {
					t.Errorf("expected sharename=%s, got %s", tt.shareName, r.URL.Query().Get("sharename"))
				}

				if tt.root != "" && r.URL.Query().Get("root") != tt.root {
					t.Errorf("expected root=%s, got %s", tt.root, r.URL.Query().Get("root"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			result, err := fs.GetACLControl(context.Background(), tt.shareName, tt.root)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetACLControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.Enabled != tt.want.Enabled {
				t.Errorf("GetACLControl() Enabled = %v, want %v", result.Enabled, tt.want.Enabled)
			}

			if result.Recursive != tt.want.Recursive {
				t.Errorf("GetACLControl() Recursive = %v, want %v", result.Recursive, tt.want.Recursive)
			}

			if result.Propagation != tt.want.Propagation {
				t.Errorf("GetACLControl() Propagation = %q, want %q", result.Propagation, tt.want.Propagation)
			}

			if len(result.ACLs) != len(tt.want.ACLs) {
				t.Errorf("GetACLControl() ACLs count = %d, want %d", len(result.ACLs), len(tt.want.ACLs))
			}
		})
	}
}

// TestGetACLControl_AuthError tests authentication errors
func TestGetACLControl_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "data": {"enabled": true}}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	_, err := fs.GetACLControl(context.Background(), "public", "")

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestGetACLControl_APIErrors tests API error responses
func TestGetACLControl_APIErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "data": {"enabled": false}}`,
			wantErr:  true,
			errMsg:   "failed to get ACL control",
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "share not found"}`,
			wantErr:  true,
			errMsg:   "failed to get ACL control",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse ACL control response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.GetACLControl(context.Background(), "public", "")

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestGetACLUserGroupList_Success tests successful user/group list retrieval
func TestGetACLUserGroupList_Success(t *testing.T) {
	tests := []struct {
		name      string
		shareName string
		response  string
		want      *ACLUserGroupList
		wantErr   bool
	}{
		{
			name:      "get user and group list",
			shareName: "public",
			response: `{
				"status": 1,
				"data": {
					"users": [
						{
							"name": "admin",
							"domain": "local",
							"fullname": "Administrator",
							"isadmin": true
						},
						{
							"name": "user1",
							"domain": "local",
							"fullname": "User One",
							"isadmin": false
						}
					],
					"groups": [
						{
							"name": "everyone",
							"domain": "local",
							"desc": "All users"
						},
						{
							"name": "staff",
							"domain": "local",
							"desc": "Staff members"
						}
					]
				}
			}`,
			want: &ACLUserGroupList{
				Users: []ACLUser{
					{Name: "admin", Domain: "local", FullName: "Administrator", IsAdmin: true},
					{Name: "user1", Domain: "local", FullName: "User One", IsAdmin: false},
				},
				Groups: []ACLGroup{
					{Name: "everyone", Domain: "local", Desc: "All users"},
					{Name: "staff", Domain: "local", Desc: "Staff members"},
				},
			},
			wantErr: false,
		},
		{
			name:      "get empty user/group list",
			shareName: "empty",
			response: `{
				"status": 1,
				"data": {
					"users": [],
					"groups": []
				}
			}`,
			want: &ACLUserGroupList{
				Users:  []ACLUser{},
				Groups: []ACLGroup{},
			},
			wantErr: false,
		},
		{
			name:      "get users only",
			shareName: "public",
			response: `{
				"status": 1,
				"data": {
					"users": [
						{
							"name": "admin",
							"domain": "local",
							"fullname": "Administrator",
							"isadmin": true
						}
					],
					"groups": []
				}
			}`,
			want: &ACLUserGroupList{
				Users: []ACLUser{
					{Name: "admin", Domain: "local", FullName: "Administrator", IsAdmin: true},
				},
				Groups: []ACLGroup{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "get_acl_user_group_list_out" {
					t.Errorf("expected func=get_acl_user_group_list_out, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("sharename") != tt.shareName {
					t.Errorf("expected sharename=%s, got %s", tt.shareName, r.URL.Query().Get("sharename"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			result, err := fs.GetACLUserGroupList(context.Background(), tt.shareName)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetACLUserGroupList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result.Users) != len(tt.want.Users) {
				t.Errorf("GetACLUserGroupList() Users count = %d, want %d", len(result.Users), len(tt.want.Users))
			}

			if len(result.Groups) != len(tt.want.Groups) {
				t.Errorf("GetACLUserGroupList() Groups count = %d, want %d", len(result.Groups), len(tt.want.Groups))
			}

			if len(result.Users) > 0 {
				if result.Users[0].Name != tt.want.Users[0].Name {
					t.Errorf("GetACLUserGroupList() first user name = %q, want %q", result.Users[0].Name, tt.want.Users[0].Name)
				}
			}
		})
	}
}

// TestGetACLUserGroupList_AuthError tests authentication errors
func TestGetACLUserGroupList_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "data": {"users": [], "groups": []}}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	_, err := fs.GetACLUserGroupList(context.Background(), "public")

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestGetACLUserGroupList_APIErrors tests API error responses
func TestGetACLUserGroupList_APIErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "data": {"users": [], "groups": []}}`,
			wantErr:  true,
			errMsg:   "failed to get ACL user/group list",
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "share not found"}`,
			wantErr:  true,
			errMsg:   "failed to get ACL user/group list",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse ACL user/group list response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.GetACLUserGroupList(context.Background(), "public")

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestSetPrivilege_Success tests successful privilege setting
func TestSetPrivilege_Success(t *testing.T) {
	tests := []struct {
		name     string
		options  *SetPrivilegeOptions
		response string
		wantErr  bool
		verify   func(*testing.T, *http.Request)
	}{
		{
			name: "set privilege for user",
			options: &SetPrivilegeOptions{
				ShareName: "public",
				Privileges: []PrivilegeEntry{
					{
						User:   "admin",
						Domain: "local",
						IsUser: true,
						Right:  "rwx",
						IsFile: false,
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("priv_0_user") != "admin" {
					t.Errorf("expected priv_0_user=admin, got %s", r.URL.Query().Get("priv_0_user"))
				}
				if r.URL.Query().Get("priv_0_isuser") != "1" {
					t.Errorf("expected priv_0_isuser=1, got %s", r.URL.Query().Get("priv_0_isuser"))
				}
				if r.URL.Query().Get("priv_0_isfile") != "0" {
					t.Errorf("expected priv_0_isfile=0, got %s", r.URL.Query().Get("priv_0_isfile"))
				}
			},
		},
		{
			name: "set privilege with path",
			options: &SetPrivilegeOptions{
				ShareName: "public",
				Path:      "/documents",
				Privileges: []PrivilegeEntry{
					{
						User:   "user1",
						Domain: "local",
						IsUser: true,
						Right:  "rw",
						IsFile: false,
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("path") != "/documents" {
					t.Errorf("expected path=/documents, got %s", r.URL.Query().Get("path"))
				}
			},
		},
		{
			name: "set privilege with recursive flag",
			options: &SetPrivilegeOptions{
				ShareName: "public",
				Recursive: true,
				Privileges: []PrivilegeEntry{
					{
						User:   "admin",
						Domain: "local",
						IsUser: true,
						Right:  "rwx",
						IsFile: false,
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("recursive") != "1" {
					t.Errorf("expected recursive=1, got %s", r.URL.Query().Get("recursive"))
				}
			},
		},
		{
			name: "set privilege for file",
			options: &SetPrivilegeOptions{
				ShareName: "public",
				Privileges: []PrivilegeEntry{
					{
						User:   "user1",
						Domain: "local",
						IsUser: true,
						Right:  "r",
						IsFile: true,
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("priv_0_isfile") != "1" {
					t.Errorf("expected priv_0_isfile=1, got %s", r.URL.Query().Get("priv_0_isfile"))
				}
			},
		},
		{
			name: "set privilege for group",
			options: &SetPrivilegeOptions{
				ShareName: "public",
				Privileges: []PrivilegeEntry{
					{
						User:   "everyone",
						Domain: "local",
						IsUser: false,
						Right:  "r",
						IsFile: false,
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("priv_0_isuser") != "0" {
					t.Errorf("expected priv_0_isuser=0, got %s", r.URL.Query().Get("priv_0_isuser"))
				}
			},
		},
		{
			name: "set multiple privileges",
			options: &SetPrivilegeOptions{
				ShareName: "public",
				Privileges: []PrivilegeEntry{
					{
						User:   "admin",
						Domain: "local",
						IsUser: true,
						Right:  "rwx",
						IsFile: false,
					},
					{
						User:   "user1",
						Domain: "local",
						IsUser: true,
						Right:  "rw",
						IsFile: false,
					},
					{
						User:   "everyone",
						Domain: "local",
						IsUser: false,
						Right:  "r",
						IsFile: false,
					},
				},
			},
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "set_privilege" {
					t.Errorf("expected func=set_privilege, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("sharename") != tt.options.ShareName {
					t.Errorf("expected sharename=%s, got %s", tt.options.ShareName, r.URL.Query().Get("sharename"))
				}

				if tt.verify != nil {
					tt.verify(t, r)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.SetPrivilege(context.Background(), tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetPrivilege() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSetPrivilege_AuthError tests authentication errors
func TestSetPrivilege_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "success": "true"}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	err := fs.SetPrivilege(context.Background(), &SetPrivilegeOptions{
		ShareName: "public",
		Privileges: []PrivilegeEntry{
			{User: "admin", Domain: "local", IsUser: true, Right: "rwx", IsFile: false},
		},
	})

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestSetPrivilege_APIErrors tests API error responses
func TestSetPrivilege_APIErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "success": "false"}`,
			wantErr:  true,
			errMsg:   "failed to set privilege",
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "permission denied"}`,
			wantErr:  true,
			errMsg:   "failed to set privilege",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse set privilege response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.SetPrivilege(context.Background(), &SetPrivilegeOptions{
				ShareName: "public",
				Privileges: []PrivilegeEntry{
					{User: "admin", Domain: "local", IsUser: true, Right: "rwx", IsFile: false},
				},
			})

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestGetAccessRight_Success tests successful access right retrieval
func TestGetAccessRight_Success(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		response string
		want     *AccessRight
		wantErr  bool
	}{
		{
			name: "get access right for file",
			path: "/share/public/document.txt",
			response: `{
				"status": 1,
				"data": {
					"path": "/share/public/document.txt",
					"isfolder": false,
					"owner": "admin",
					"group": "everyone",
					"permission": "rwxr-xr-x",
					"acl_enabled": false,
					"acls": []
				}
			}`,
			want: &AccessRight{
				Path:       "/share/public/document.txt",
				IsFolder:   false,
				Owner:      "admin",
				Group:      "everyone",
				Permission: "rwxr-xr-x",
				ACLEnabled: false,
				ACLs:       []ACLEntry{},
			},
			wantErr: false,
		},
		{
			name: "get access right for folder",
			path: "/share/public/folder",
			response: `{
				"status": 1,
				"data": {
					"path": "/share/public/folder",
					"isfolder": true,
					"owner": "admin",
					"group": "everyone",
					"permission": "rwxrwxrwx",
					"acl_enabled": true,
					"acls": [
						{
							"user": "admin",
							"domain": "local",
							"isuser": true,
							"right": "full"
						},
						{
							"user": "user1",
							"domain": "local",
							"isuser": true,
							"right": "rw"
						}
					]
				}
			}`,
			want: &AccessRight{
				Path:       "/share/public/folder",
				IsFolder:   true,
				Owner:      "admin",
				Group:      "everyone",
				Permission: "rwxrwxrwx",
				ACLEnabled: true,
				ACLs: []ACLEntry{
					{User: "admin", Domain: "local", IsUser: true, Right: "full"},
					{User: "user1", Domain: "local", IsUser: true, Right: "rw"},
				},
			},
			wantErr: false,
		},
		{
			name: "get access right for root share",
			path: "/share/public",
			response: `{
				"status": 1,
				"data": {
					"path": "/share/public",
					"isfolder": true,
					"owner": "admin",
					"group": "everyone",
					"permission": "rwxr-xr-x",
					"acl_enabled": false,
					"acls": []
				}
			}`,
			want: &AccessRight{
				Path:       "/share/public",
				IsFolder:   true,
				Owner:      "admin",
				Group:      "everyone",
				Permission: "rwxr-xr-x",
				ACLEnabled: false,
				ACLs:       []ACLEntry{},
			},
			wantErr: false,
		},
		{
			name: "get access right for nested path",
			path: "/share/public/documents/reports/q1.xlsx",
			response: `{
				"status": 1,
				"data": {
					"path": "/share/public/documents/reports/q1.xlsx",
					"isfolder": false,
					"owner": "user1",
					"group": "staff",
					"permission": "rw-r-----",
					"acl_enabled": true,
					"acls": []
				}
			}`,
			want: &AccessRight{
				Path:       "/share/public/documents/reports/q1.xlsx",
				IsFolder:   false,
				Owner:      "user1",
				Group:      "staff",
				Permission: "rw-r-----",
				ACLEnabled: true,
				ACLs:       []ACLEntry{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "get_access_right" {
					t.Errorf("expected func=get_access_right, got %s", r.URL.Query().Get("func"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			result, err := fs.GetAccessRight(context.Background(), tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAccessRight() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.Path != tt.want.Path {
				t.Errorf("GetAccessRight() Path = %q, want %q", result.Path, tt.want.Path)
			}

			if result.IsFolder != tt.want.IsFolder {
				t.Errorf("GetAccessRight() IsFolder = %v, want %v", result.IsFolder, tt.want.IsFolder)
			}

			if result.Owner != tt.want.Owner {
				t.Errorf("GetAccessRight() Owner = %q, want %q", result.Owner, tt.want.Owner)
			}

			if result.Permission != tt.want.Permission {
				t.Errorf("GetAccessRight() Permission = %q, want %q", result.Permission, tt.want.Permission)
			}

			if result.ACLEnabled != tt.want.ACLEnabled {
				t.Errorf("GetAccessRight() ACLEnabled = %v, want %v", result.ACLEnabled, tt.want.ACLEnabled)
			}
		})
	}
}

// TestGetAccessRight_PathParsing tests path parsing logic
func TestGetAccessRight_PathParsing(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectedShare   string
		expectedRelPath string
	}{
		{
			name:            "simple path",
			path:            "/share/public/file.txt",
			expectedShare:   "share",
			expectedRelPath: "/public/file.txt",
		},
		{
			name:            "root share",
			path:            "/public",
			expectedShare:   "public",
			expectedRelPath: "",
		},
		{
			name:            "leading slash only",
			path:            "/file.txt",
			expectedShare:   "file.txt",
			expectedRelPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				shareName := r.URL.Query().Get("sharename")
				relPath := r.URL.Query().Get("path")

				if shareName != tt.expectedShare {
					t.Errorf("expected sharename=%s, got %s", tt.expectedShare, shareName)
				}

				// For relative path, check if it matches expected (or not present if empty)
				if tt.expectedRelPath == "" {
					if relPath != "" {
						t.Errorf("expected empty path parameter, got %s", relPath)
					}
				} else {
					if relPath != tt.expectedRelPath {
						t.Errorf("expected path=%s, got %s", tt.expectedRelPath, relPath)
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"status": 1,
					"data": {
						"path": "` + tt.path + `",
						"isfolder": false,
						"owner": "admin",
						"group": "everyone",
						"permission": "rwxr-xr-x",
						"acl_enabled": false,
						"acls": []
					}
				}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.GetAccessRight(context.Background(), tt.path)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestGetAccessRight_InvalidPath tests invalid path handling
func TestGetAccessRight_InvalidPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "invalid path",
		},
		{
			name:    "whitespace path",
			path:    "   ",
			wantErr: true,
			errMsg:  "invalid path",
		},
		{
			name:    "slash only",
			path:    "/",
			wantErr: true,
			errMsg:  "invalid path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": 1, "data": {"path": "/", "isfolder": true}}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.GetAccessRight(context.Background(), tt.path)

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestGetAccessRight_AuthError tests authentication errors
func TestGetAccessRight_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "data": {"path": "/test", "isfolder": false}}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	_, err := fs.GetAccessRight(context.Background(), "/share/public/test.txt")

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestGetAccessRight_APIErrors tests API error responses
func TestGetAccessRight_APIErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "data": {"path": "/test"}}`,
			wantErr:  true,
			errMsg:   "failed to get access right",
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "file not found"}`,
			wantErr:  true,
			errMsg:   "failed to get access right",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse access right response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.GetAccessRight(context.Background(), "/share/public/test.txt")

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestSetProjectionType_Success tests successful projection type setting
func TestSetProjectionType_Success(t *testing.T) {
	tests := []struct {
		name           string
		shareName      string
		projectionType string
		response       string
		wantErr        bool
	}{
		{
			name:           "set projection type to private",
			shareName:      "public",
			projectionType: "private",
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
		},
		{
			name:           "set projection type to public",
			shareName:      "private",
			projectionType: "public",
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
		},
		{
			name:           "set projection type to hidden",
			shareName:      "share",
			projectionType: "hidden",
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "set_projection_type" {
					t.Errorf("expected func=set_projection_type, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("sharename") != tt.shareName {
					t.Errorf("expected sharename=%s, got %s", tt.shareName, r.URL.Query().Get("sharename"))
				}

				if r.URL.Query().Get("projectionType") != tt.projectionType {
					t.Errorf("expected projectionType=%s, got %s", tt.projectionType, r.URL.Query().Get("projectionType"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.SetProjectionType(context.Background(), tt.shareName, tt.projectionType)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetProjectionType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSetProjectionType_AuthError tests authentication errors
func TestSetProjectionType_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "success": "true"}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	err := fs.SetProjectionType(context.Background(), "public", "private")

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestSetProjectionType_APIErrors tests API error responses
func TestSetProjectionType_APIErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "success": "false"}`,
			wantErr:  true,
			errMsg:   "failed to set projection type",
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "invalid projection type"}`,
			wantErr:  true,
			errMsg:   "failed to set projection type",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse set projection type response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.SetProjectionType(context.Background(), "public", "private")

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestBoolToInt tests the boolToInt helper function
func TestBoolToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected int
	}{
		{
			name:     "true converts to 1",
			input:    true,
			expected: 1,
		},
		{
			name:     "false converts to 0",
			input:    false,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := boolToInt(tt.input)
			if result != tt.expected {
				t.Errorf("boolToInt(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestACLEntrySerialization tests ACL entry serialization in various scenarios
func TestACLEntrySerialization(t *testing.T) {
	tests := []struct {
		name   string
		acls   []ACLEntry
		verify func(*testing.T, *http.Request)
	}{
		{
			name: "empty ACL list",
			acls: []ACLEntry{},
			verify: func(t *testing.T, r *http.Request) {
				// No ACL parameters should be present
				if r.URL.Query().Get("acl_0_user") != "" {
					t.Error("expected no ACL parameters for empty list")
				}
			},
		},
		{
			name: "single ACL with all fields",
			acls: []ACLEntry{
				{
					User:   "testuser",
					Domain: "testdomain",
					IsUser: true,
					Right:  "rw",
				},
			},
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("acl_0_user") != "testuser" {
					t.Errorf("expected acl_0_user=testuser, got %s", r.URL.Query().Get("acl_0_user"))
				}
				if r.URL.Query().Get("acl_0_domain") != "testdomain" {
					t.Errorf("expected acl_0_domain=testdomain, got %s", r.URL.Query().Get("acl_0_domain"))
				}
				if r.URL.Query().Get("acl_0_right") != "rw" {
					t.Errorf("expected acl_0_right=rw, got %s", r.URL.Query().Get("acl_0_right"))
				}
			},
		},
		{
			name: "ACL with domain group",
			acls: []ACLEntry{
				{
					User:   "DOMAIN+group",
					Domain: "corp.example.com",
					IsUser: false,
					Right:  "r",
				},
			},
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("acl_0_user") != "DOMAIN+group" {
					t.Errorf("expected acl_0_user=DOMAIN+group, got %s", r.URL.Query().Get("acl_0_user"))
				}
				if r.URL.Query().Get("acl_0_isuser") != "0" {
					t.Errorf("expected acl_0_isuser=0, got %s", r.URL.Query().Get("acl_0_isuser"))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				if tt.verify != nil {
					tt.verify(t, r)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": 1, "success": "true"}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_ = fs.SetACLControl(context.Background(), &SetACLOptions{
				ShareName: "test",
				ACLs:      tt.acls,
			})
		})
	}
}

// TestPrivilegeEntrySerialization tests privilege entry serialization
func TestPrivilegeEntrySerialization(t *testing.T) {
	tests := []struct {
		name       string
		privileges []PrivilegeEntry
		verify     func(*testing.T, *http.Request)
	}{
		{
			name:       "empty privilege list",
			privileges: []PrivilegeEntry{},
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("priv_0_user") != "" {
					t.Error("expected no privilege parameters for empty list")
				}
			},
		},
		{
			name: "privilege with all fields",
			privileges: []PrivilegeEntry{
				{
					User:   "testuser",
					Domain: "testdomain",
					IsUser: true,
					Right:  "rwx",
					IsFile: false,
				},
			},
			verify: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("priv_0_user") != "testuser" {
					t.Errorf("expected priv_0_user=testuser, got %s", r.URL.Query().Get("priv_0_user"))
				}
				if r.URL.Query().Get("priv_0_isfile") != "0" {
					t.Errorf("expected priv_0_isfile=0, got %s", r.URL.Query().Get("priv_0_isfile"))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				if tt.verify != nil {
					tt.verify(t, r)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": 1, "success": "true"}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_ = fs.SetPrivilege(context.Background(), &SetPrivilegeOptions{
				ShareName:  "test",
				Privileges: tt.privileges,
			})
		})
	}
}

// TestPermissionsContextCancellation tests context cancellation handling
func TestPermissionsContextCancellation(t *testing.T) {
	tests := []struct {
		name    string
		execute func(*FileStationService, context.Context) error
	}{
		{
			name: "SetACLControl cancellation",
			execute: func(fs *FileStationService, ctx context.Context) error {
				return fs.SetACLControl(ctx, &SetACLOptions{
					ShareName: "public",
					ACLs: []ACLEntry{
						{User: "admin", Domain: "local", IsUser: true, Right: "full"},
					},
				})
			},
		},
		{
			name: "GetACLControl cancellation",
			execute: func(fs *FileStationService, ctx context.Context) error {
				_, err := fs.GetACLControl(ctx, "public", "")
				return err
			},
		},
		{
			name: "GetACLUserGroupList cancellation",
			execute: func(fs *FileStationService, ctx context.Context) error {
				_, err := fs.GetACLUserGroupList(ctx, "public")
				return err
			},
		},
		{
			name: "SetPrivilege cancellation",
			execute: func(fs *FileStationService, ctx context.Context) error {
				return fs.SetPrivilege(ctx, &SetPrivilegeOptions{
					ShareName: "public",
					Privileges: []PrivilegeEntry{
						{User: "admin", Domain: "local", IsUser: true, Right: "rwx", IsFile: false},
					},
				})
			},
		},
		{
			name: "GetAccessRight cancellation",
			execute: func(fs *FileStationService, ctx context.Context) error {
				_, err := fs.GetAccessRight(ctx, "/share/public/test.txt")
				return err
			},
		},
		{
			name: "SetProjectionType cancellation",
			execute: func(fs *FileStationService, ctx context.Context) error {
				return fs.SetProjectionType(ctx, "public", "private")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": 1, "success": "true"}`))
			})
			defer mock.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			fs := NewFileStationService(mock.GetClient())
			err := tt.execute(fs, ctx)

			if err == nil {
				t.Errorf("expected context cancellation error, got nil")
			}
		})
	}
}

// TestIntegration_ACLWorkflow tests a complete ACL workflow
func TestIntegration_ACLWorkflow(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		funcName := r.URL.Query().Get("func")

		w.Header().Set("Content-Type", "application/json")

		switch funcName {
		case "set_acl_control":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": 1, "success": "true"}`))
		case "get_acl_control":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": 1,
				"data": {
					"enabled": true,
					"recursive": false,
					"propagation": "inherit",
					"acls": [
						{
							"user": "admin",
							"domain": "local",
							"isuser": true,
							"right": "full"
						}
					]
				}
			}`))
		case "get_acl_user_group_list_out":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": 1,
				"data": {
					"users": [
						{"name": "admin", "domain": "local", "fullname": "Admin", "isadmin": true}
					],
					"groups": [
						{"name": "everyone", "domain": "local", "desc": "All users"}
					]
				}
			}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "unknown function"}`))
		}
	})
	defer mock.Close()

	fs := NewFileStationService(mock.GetClient())

	// Step 1: Set ACL control
	err := fs.SetACLControl(context.Background(), &SetACLOptions{
		ShareName: "public",
		ACLs: []ACLEntry{
			{User: "admin", Domain: "local", IsUser: true, Right: "full"},
		},
	})
	if err != nil {
		t.Fatalf("SetACLControl failed: %v", err)
	}

	// Step 2: Get ACL control to verify
	aclControl, err := fs.GetACLControl(context.Background(), "public", "")
	if err != nil {
		t.Fatalf("GetACLControl failed: %v", err)
	}
	if !aclControl.Enabled {
		t.Error("expected ACL to be enabled")
	}

	// Step 3: Get user/group list
	userGroupList, err := fs.GetACLUserGroupList(context.Background(), "public")
	if err != nil {
		t.Fatalf("GetACLUserGroupList failed: %v", err)
	}
	if len(userGroupList.Users) == 0 {
		t.Error("expected at least one user")
	}
	if len(userGroupList.Groups) == 0 {
		t.Error("expected at least one group")
	}
}

// TestIntegration_PrivilegeWorkflow tests a complete privilege workflow
func TestIntegration_PrivilegeWorkflow(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		funcName := r.URL.Query().Get("func")

		w.Header().Set("Content-Type", "application/json")

		switch funcName {
		case "set_privilege":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": 1, "success": "true"}`))
		case "get_access_right":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": 1,
				"data": {
					"path": "/share/public/file.txt",
					"isfolder": false,
					"owner": "admin",
					"group": "everyone",
					"permission": "rwxr-xr-x",
					"acl_enabled": true,
					"acls": [
						{
							"user": "admin",
							"domain": "local",
							"isuser": true,
							"right": "rwx"
						}
					]
				}
			}`))
		case "set_projection_type":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": 1, "success": "true"}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "unknown function"}`))
		}
	})
	defer mock.Close()

	fs := NewFileStationService(mock.GetClient())

	// Step 1: Set privilege
	err := fs.SetPrivilege(context.Background(), &SetPrivilegeOptions{
		ShareName: "public",
		Path:      "/documents",
		Privileges: []PrivilegeEntry{
			{User: "admin", Domain: "local", IsUser: true, Right: "rwx", IsFile: false},
		},
	})
	if err != nil {
		t.Fatalf("SetPrivilege failed: %v", err)
	}

	// Step 2: Get access right to verify
	accessRight, err := fs.GetAccessRight(context.Background(), "/share/public/file.txt")
	if err != nil {
		t.Fatalf("GetAccessRight failed: %v", err)
	}
	if accessRight.Owner != "admin" {
		t.Errorf("expected owner=admin, got %s", accessRight.Owner)
	}

	// Step 3: Set projection type
	err = fs.SetProjectionType(context.Background(), "public", "private")
	if err != nil {
		t.Fatalf("SetProjectionType failed: %v", err)
	}
}

// TestPermissionsNetworkErrors tests network error handling
func TestPermissionsNetworkErrors(t *testing.T) {
	tests := []struct {
		name    string
		execute func(*FileStationService, context.Context) error
	}{
		{
			name: "SetACLControl network error",
			execute: func(fs *FileStationService, ctx context.Context) error {
				return fs.SetACLControl(ctx, &SetACLOptions{
					ShareName: "public",
					ACLs:      []ACLEntry{{User: "admin", Domain: "local", IsUser: true, Right: "full"}},
				})
			},
		},
		{
			name: "GetACLControl network error",
			execute: func(fs *FileStationService, ctx context.Context) error {
				_, err := fs.GetACLControl(ctx, "public", "")
				return err
			},
		},
		{
			name: "GetACLUserGroupList network error",
			execute: func(fs *FileStationService, ctx context.Context) error {
				_, err := fs.GetACLUserGroupList(ctx, "public")
				return err
			},
		},
		{
			name: "SetPrivilege network error",
			execute: func(fs *FileStationService, ctx context.Context) error {
				return fs.SetPrivilege(ctx, &SetPrivilegeOptions{
					ShareName:  "public",
					Privileges: []PrivilegeEntry{{User: "admin", Domain: "local", IsUser: true, Right: "rwx", IsFile: false}},
				})
			},
		},
		{
			name: "GetAccessRight network error",
			execute: func(fs *FileStationService, ctx context.Context) error {
				_, err := fs.GetAccessRight(ctx, "/share/public/test.txt")
				return err
			},
		},
		{
			name: "SetProjectionType network error",
			execute: func(fs *FileStationService, ctx context.Context) error {
				return fs.SetProjectionType(ctx, "public", "private")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock but close server immediately to simulate network error
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {})
			mock.server.Close()
			mock.server = nil
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := tt.execute(fs, context.Background())

			if err == nil {
				t.Errorf("expected network error, got nil")
			}
		})
	}
}

// TestPermissionsSpecialCharacters tests handling of special characters in parameters
func TestPermissionsSpecialCharacters(t *testing.T) {
	tests := []struct {
		name      string
		shareName string
		path      string
		user      string
		domain    string
	}{
		{
			name:      "share with spaces",
			shareName: "my share",
			path:      "",
			user:      "admin",
			domain:    "local",
		},
		{
			name:      "path with special characters",
			shareName: "public",
			path:      "/documents/folder with spaces/file (1).txt",
			user:      "admin",
			domain:    "local",
		},
		{
			name:      "user with special characters",
			shareName: "public",
			path:      "",
			user:      "user@example.com",
			domain:    "local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": 1, "success": "true"}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())

			// Test SetACLControl with special characters
			err := fs.SetACLControl(context.Background(), &SetACLOptions{
				ShareName: tt.shareName,
				ACLs: []ACLEntry{
					{User: tt.user, Domain: tt.domain, IsUser: true, Right: "full"},
				},
			})

			if err != nil {
				t.Errorf("unexpected error with special characters: %v", err)
			}
		})
	}
}
