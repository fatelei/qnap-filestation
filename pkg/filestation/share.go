package filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// ShareListResponse represents the response from listing share links
type ShareListResponse struct {
	api.BaseResponse
	Data struct {
		Shares []ShareLink `json:"shares"`
		Total  int         `json:"total"`
	} `json:"data"`
}

// ListShareLinks lists all share links
func (fs *FileStationService) ListShareLinks(ctx context.Context) ([]ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/share.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Sharing",
		"method":  "list",
		"version": "2",
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var listResp ShareListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse share list response", err)
	}

	if !listResp.IsSuccess() {
		return nil, &api.APIError{
			Code:    listResp.GetErrorCode(),
			Message: listResp.Message,
		}
	}

	return listResp.Data.Shares, nil
}

// QNAPShareCreateResponse represents the response from creating a QNAP share link
type QNAPShareCreateResponse struct {
	List []ShareLinkItem `json:"list"`
}

// ShareLinkItem represents a QNAP share link item
type ShareLinkItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// CreateShareLink creates a new share link
func (fs *FileStationService) CreateShareLink(ctx context.Context, path string, options *ShareLinkOptions) (*ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	// Extract file/folder name from path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 {
		return nil, api.WrapAPIError(api.ErrUnknown, "invalid path", nil)
	}

	fileName := parts[len(parts)-1]
	destPath := "/" + strings.Join(parts[:len(parts)-1], "/")
	if destPath == "/" {
		destPath = ""
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":       "get_share_link",
		"sid":        sid,
		"c":          "1", // Direct create
		"path":       destPath,
		"file_name":  fileName,
		"file_total": "1",
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var createResp QNAPShareCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse create share response", err)
	}

	if len(createResp.List) == 0 {
		return nil, api.NewAPIError(api.ErrUnknown, "no share link created")
	}

	item := createResp.List[0]
	return &ShareLink{
		URL: item.URL,
	}, nil
}

// DeleteShareLink deletes a share link by share name or ID
func (fs *FileStationService) DeleteShareLink(ctx context.Context, shareName string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "delete_share",
		"sid":       sid,
		"sharename": shareName,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse delete share response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// UpdateShareLinkOptions contains options for updating a share link
type UpdateShareLinkOptions struct {
	SSID       string `json:"ssid,omitempty"`
	ExpireTime int64  `json:"expire_time,omitempty"`
	Password   string `json:"password,omitempty"`
	ValidDays  int    `json:"valid_days,omitempty"`
}

// UpdateShareLinkResponse represents the response from updating a share link
type UpdateShareLinkResponse struct {
	api.BaseResponse
	Data ShareLink `json:"data"`
}

// UpdateShareLink updates an existing share link
func (fs *FileStationService) UpdateShareLink(ctx context.Context, options *UpdateShareLinkOptions) (*ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "update_share_link",
		"sid":  sid,
	}

	if options.SSID != "" {
		params["ssid"] = options.SSID
	}
	if options.ExpireTime > 0 {
		params["expire_time"] = fmt.Sprintf("%d", options.ExpireTime)
	}
	if options.Password != "" {
		params["password"] = options.Password
	}
	if options.ValidDays > 0 {
		params["valid_days"] = fmt.Sprintf("%d", options.ValidDays)
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result UpdateShareLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse update share response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result.Data, nil
}

// GetShareListResponse represents the response from getting share list
type GetShareListResponse struct {
	api.BaseResponse
	Data struct {
		Shares []ShareLink `json:"shares"`
		Total  int         `json:"total"`
	} `json:"data"`
}

// GetShareList lists all shares
func (fs *FileStationService) GetShareList(ctx context.Context) ([]ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_share_list",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetShareListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse share list response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return result.Data.Shares, nil
}

// ShareMember represents a member of a share
type ShareMember struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Access  string `json:"access"`
	IsGroup bool   `json:"is_group"`
}

// GetShareSublistResponse represents the response from getting share sublist
type GetShareSublistResponse struct {
	api.BaseResponse
	Data struct {
		Members []ShareMember `json:"members"`
		Total   int           `json:"total"`
	} `json:"data"`
}

// GetShareSublist gets members of a share
func (fs *FileStationService) GetShareSublist(ctx context.Context, shareName string) ([]ShareMember, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "get_share_sublist",
		"sid":       sid,
		"sharename": shareName,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetShareSublistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse share sublist response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return result.Data.Members, nil
}

// AddShareSublistOptions contains options for adding a member to a share
type AddShareSublistOptions struct {
	ShareName string `json:"sharename"`
	UserID    string `json:"user_id"`
	Access    string `json:"access"`
	IsGroup   bool   `json:"is_group"`
}

// AddShareSublist adds a member to a share
func (fs *FileStationService) AddShareSublist(ctx context.Context, options *AddShareSublistOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "add_share_sublist",
		"sid":       sid,
		"sharename": options.ShareName,
		"user_id":   options.UserID,
		"access":    options.Access,
	}

	if options.IsGroup {
		params["is_group"] = "1"
	} else {
		params["is_group"] = "0"
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse add share sublist response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// DeleteShareSublist removes a member from a share
func (fs *FileStationService) DeleteShareSublist(ctx context.Context, shareName, userID string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "delete_share_sublist",
		"sid":       sid,
		"sharename": shareName,
		"user_id":   userID,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse delete share sublist response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// ShareAccessControlOptions contains options for setting access control
type ShareAccessControlOptions struct {
	ShareName   string `json:"sharename"`
	AccessLevel string `json:"access_level"`
	ReadOnly    bool   `json:"read_only"`
	Writeable   bool   `json:"writeable"`
}

// ShareAccessControl sets access control for a share
func (fs *FileStationService) ShareAccessControl(ctx context.Context, options *ShareAccessControlOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "share_access_control",
		"sid":       sid,
		"sharename": options.ShareName,
		"access":    options.AccessLevel,
		"read_only": "0",
		"writeable": "0",
	}

	if options.ReadOnly {
		params["read_only"] = "1"
	}
	if options.Writeable {
		params["writeable"] = "1"
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse share access control response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// SendShareMailOptions contains options for sending share mail
type SendShareMailOptions struct {
	ShareName string   `json:"sharename"`
	To        []string `json:"to"`
	CC        []string `json:"cc,omitempty"`
	Subject   string   `json:"subject,omitempty"`
	Message   string   `json:"message,omitempty"`
}

// SendShareMail sends a share link via email
func (fs *FileStationService) SendShareMail(ctx context.Context, options *SendShareMailOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "send_share_mail",
		"sid":       sid,
		"sharename": options.ShareName,
		"to":        strings.Join(options.To, ","),
	}

	if options.Subject != "" {
		params["subject"] = options.Subject
	}
	if options.Message != "" {
		params["message"] = options.Message
	}
	if len(options.CC) > 0 {
		params["cc"] = strings.Join(options.CC, ",")
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse send share mail response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// MailContact represents a mail contact
type MailContact struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetPersonalMailListResponse represents the response from getting personal mail list
type GetPersonalMailListResponse struct {
	api.BaseResponse
	Data struct {
		Contacts []MailContact `json:"contacts"`
		Total    int           `json:"total"`
	} `json:"data"`
}

// GetPersonalMailList gets the personal mail list
func (fs *FileStationService) GetPersonalMailList(ctx context.Context) ([]MailContact, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_personal_mail_list",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetPersonalMailListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse personal mail list response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return result.Data.Contacts, nil
}

// GetSharedWithMeResponse represents the response from getting shares shared with user
type GetSharedWithMeResponse struct {
	api.BaseResponse
	Data struct {
		Shares []ShareLink `json:"shares"`
		Total  int         `json:"total"`
	} `json:"data"`
}

// GetSharedWithMe gets shares shared with the current user
func (fs *FileStationService) GetSharedWithMe(ctx context.Context) ([]ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_shared_with_me",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetSharedWithMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse shared with me response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return result.Data.Shares, nil
}

// GetShareLinkInfoResponse represents the response from getting share link info
type GetShareLinkInfoResponse struct {
	api.BaseResponse
	Data ShareLink `json:"data"`
}

// GetShareLinkInfo gets detailed information about a share link
func (fs *FileStationService) GetShareLinkInfo(ctx context.Context, shareName string) (*ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "get_share_link_info",
		"sid":       sid,
		"sharename": shareName,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetShareLinkInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse share link info response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result.Data, nil
}

// SetShareNasUserOptions contains options for sharing to NAS user
type SetShareNasUserOptions struct {
	ShareName string `json:"sharename"`
	Username  string `json:"username"`
	Access    string `json:"access"`
}

// SetShareNasUser shares a folder to a NAS user
func (fs *FileStationService) SetShareNasUser(ctx context.Context, options *SetShareNasUserOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "set_share_nas_user",
		"sid":       sid,
		"sharename": options.ShareName,
		"username":  options.Username,
		"access":    options.Access,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse set share nas user response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}
