package hrobot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// StorageBoxService handles storage box API operations.
type StorageBoxService struct {
	client *Client
}

// NewStorageBoxService creates a new storage box service.
func NewStorageBoxService(client *Client) *StorageBoxService {
	return &StorageBoxService{client: client}
}

// StorageBox represents a storage box.
//
// Some fields (DiskQuota, Webdav, Samba, SSH, ZFS, ExternalReachability,
// Server, HostSystem, ...) are only populated by GET /storagebox/{id} and
// POST /storagebox/{id}; the list endpoint returns a reduced subset.
type StorageBox struct {
	ID                   int    `json:"id"`
	Login                string `json:"login"`
	Name                 string `json:"name"`
	Product              string `json:"product"`
	Cancelled            bool   `json:"cancelled"`
	Locked               bool   `json:"locked"`
	Location             string `json:"location"`
	LinkedServer         int    `json:"linked_server"`
	PaidUntil            string `json:"paid_until"`
	DiskQuota            int    `json:"disk_quota"`
	DiskUsage            int    `json:"disk_usage"`
	DiskUsageData        int    `json:"disk_usage_data"`
	DiskUsageSnapshots   int    `json:"disk_usage_snapshots"`
	Webdav               bool   `json:"webdav"`
	Samba                bool   `json:"samba"`
	SSH                  bool   `json:"ssh"`
	ExternalReachability bool   `json:"external_reachability"`
	ZFS                  bool   `json:"zfs"`
	Server               string `json:"server"`
	HostSystem           string `json:"host_system"`
}

// StorageBoxSnapshot represents a snapshot of a storage box.
type StorageBoxSnapshot struct {
	Name           string `json:"name"`
	Timestamp      string `json:"timestamp"`
	Size           int    `json:"size"`
	FilesystemSize int    `json:"filesystem_size"`
	Automatic      bool   `json:"automatic"`
	Comment        string `json:"comment"`
}

// StorageBoxSnapshotPlan represents the snapshot schedule for a storage box.
//
// minute, hour, day_of_week, day_of_month and month are nullable on the API
// (returned as null when the plan is disabled or the field is unset), so we
// model them as pointers.
type StorageBoxSnapshotPlan struct {
	Status       string `json:"status"`
	Minute       *int   `json:"minute"`
	Hour         *int   `json:"hour"`
	DayOfWeek    *int   `json:"day_of_week"`
	DayOfMonth   *int   `json:"day_of_month"`
	Month        *int   `json:"month"`
	MaxSnapshots int    `json:"max_snapshots"`
}

// StorageBoxSubAccount represents a sub-account for a storage box.
type StorageBoxSubAccount struct {
	Username             string `json:"username"`
	AccountID            string `json:"accountid"`
	Server               string `json:"server"`
	HomeDirectory        string `json:"homedirectory"`
	Samba                bool   `json:"samba"`
	SSH                  bool   `json:"ssh"`
	ExternalReachability bool   `json:"external_reachability"`
	Webdav               bool   `json:"webdav"`
	ReadOnly             bool   `json:"readonly"`
	CreateTime           string `json:"createtime"`
	Comment              string `json:"comment"`
}

// StorageBoxSubAccountCreated is the response of CreateSubAccount and
// includes the (possibly auto-generated) password.
type StorageBoxSubAccountCreated struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	AccountID     string `json:"accountid"`
	Server        string `json:"server"`
	HomeDirectory string `json:"homedirectory"`
}

// StorageBoxUpdate carries the writable fields of POST /storagebox/{id}.
// Pointer fields are only sent when non-nil so callers can update individual
// flags without having to read-modify-write the whole record.
type StorageBoxUpdate struct {
	Name                 *string
	Samba                *bool
	Webdav               *bool
	SSH                  *bool
	ExternalReachability *bool
	ZFS                  *bool
}

// StorageBoxSubAccountInput carries the writable fields of POST/PUT
// /storagebox/{id}/subaccount[/{username}]. HomeDirectory is required by the
// API on create; on update all fields are optional.
type StorageBoxSubAccountInput struct {
	HomeDirectory        *string
	Samba                *bool
	SSH                  *bool
	ExternalReachability *bool
	Webdav               *bool
	ReadOnly             *bool
	Comment              *string
	// Password is only honoured by CreateSubAccount and ResetSubAccountPassword.
	Password *string
}

type storageBoxPasswordResponse struct {
	Password string `json:"password"`
}

// List returns all storage boxes.
//
// GET /storagebox
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-storagebox
func (s *StorageBoxService) List(ctx context.Context) ([]StorageBox, error) {
	var boxes []StorageBox
	if err := s.client.Get(ctx, "/storagebox", &boxes); err != nil {
		return nil, err
	}
	return boxes, nil
}

// Get returns details for a specific storage box.
//
// GET /storagebox/{storagebox-id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-storagebox-storagebox-id
func (s *StorageBoxService) Get(ctx context.Context, storageBoxID int) (*StorageBox, error) {
	path := fmt.Sprintf("/storagebox/%d", storageBoxID)
	var box StorageBox
	if err := s.client.Get(ctx, path, &box); err != nil {
		return nil, err
	}
	return &box, nil
}

// Update modifies storage box settings. Only fields set on the input are
// sent; the API treats omitted form fields as unchanged.
//
// POST /storagebox/{storagebox-id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id
func (s *StorageBoxService) Update(ctx context.Context, storageBoxID int, in StorageBoxUpdate) (*StorageBox, error) {
	path := fmt.Sprintf("/storagebox/%d", storageBoxID)
	data := url.Values{}
	if in.Name != nil {
		data.Set("storagebox_name", *in.Name)
	}
	if in.Samba != nil {
		data.Set("samba", strconv.FormatBool(*in.Samba))
	}
	if in.Webdav != nil {
		data.Set("webdav", strconv.FormatBool(*in.Webdav))
	}
	if in.SSH != nil {
		data.Set("ssh", strconv.FormatBool(*in.SSH))
	}
	if in.ExternalReachability != nil {
		data.Set("external_reachability", strconv.FormatBool(*in.ExternalReachability))
	}
	if in.ZFS != nil {
		data.Set("zfs", strconv.FormatBool(*in.ZFS))
	}

	var box StorageBox
	if err := s.client.Post(ctx, path, data, &box); err != nil {
		return nil, err
	}
	return &box, nil
}

// ResetPassword resets the storage box account password. If newPassword is
// empty Robot generates a random one. The returned string is the password
// that is now in effect.
//
// POST /storagebox/{storagebox-id}/password
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id-password
func (s *StorageBoxService) ResetPassword(ctx context.Context, storageBoxID int, newPassword string) (string, error) {
	path := fmt.Sprintf("/storagebox/%d/password", storageBoxID)
	data := url.Values{}
	if newPassword != "" {
		data.Set("password", newPassword)
	}

	var resp storageBoxPasswordResponse
	if err := s.client.Post(ctx, path, data, &resp); err != nil {
		return "", err
	}
	return resp.Password, nil
}

// ListSnapshots returns all snapshots for a storage box.
//
// GET /storagebox/{storagebox-id}/snapshot
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-storagebox-storagebox-id-snapshot
func (s *StorageBoxService) ListSnapshots(ctx context.Context, storageBoxID int) ([]StorageBoxSnapshot, error) {
	path := fmt.Sprintf("/storagebox/%d/snapshot", storageBoxID)
	var snapshots []StorageBoxSnapshot
	if err := s.client.Get(ctx, path, &snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

// CreateSnapshot creates a new snapshot.
//
// POST /storagebox/{storagebox-id}/snapshot
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id-snapshot
func (s *StorageBoxService) CreateSnapshot(ctx context.Context, storageBoxID int) (*StorageBoxSnapshot, error) {
	path := fmt.Sprintf("/storagebox/%d/snapshot", storageBoxID)
	var snap StorageBoxSnapshot
	if err := s.client.Post(ctx, path, nil, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// DeleteSnapshot removes a snapshot.
//
// DELETE /storagebox/{storagebox-id}/snapshot/{snapshot-name}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-storagebox-storagebox-id-snapshot-snapshot-name
func (s *StorageBoxService) DeleteSnapshot(ctx context.Context, storageBoxID int, snapshotName string) error {
	path := fmt.Sprintf("/storagebox/%d/snapshot/%s", storageBoxID, url.PathEscape(snapshotName))
	return s.client.Delete(ctx, path)
}

// RestoreSnapshot reverts the storage box to the named snapshot.
//
// POST /storagebox/{storagebox-id}/snapshot/{snapshot-name}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id-snapshot-snapshot-name
func (s *StorageBoxService) RestoreSnapshot(ctx context.Context, storageBoxID int, snapshotName string) error {
	path := fmt.Sprintf("/storagebox/%d/snapshot/%s", storageBoxID, url.PathEscape(snapshotName))
	data := url.Values{}
	data.Set("revert", "true")
	return s.client.Post(ctx, path, data, nil)
}

// SetSnapshotComment sets a comment on a snapshot.
//
// POST /storagebox/{storagebox-id}/snapshot/{snapshot-name}/comment
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id-snapshot-snapshot-name-comment
func (s *StorageBoxService) SetSnapshotComment(ctx context.Context, storageBoxID int, snapshotName, comment string) error {
	path := fmt.Sprintf("/storagebox/%d/snapshot/%s/comment", storageBoxID, url.PathEscape(snapshotName))
	data := url.Values{}
	data.Set("comment", comment)
	return s.client.Post(ctx, path, data, nil)
}

// GetSnapshotPlan retrieves the snapshot plan configuration. The endpoint
// returns a single-element array; this method unwraps it.
//
// GET /storagebox/{storagebox-id}/snapshotplan
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-storagebox-storagebox-id-snapshotplan
func (s *StorageBoxService) GetSnapshotPlan(ctx context.Context, storageBoxID int) (*StorageBoxSnapshotPlan, error) {
	path := fmt.Sprintf("/storagebox/%d/snapshotplan", storageBoxID)
	var plans []StorageBoxSnapshotPlan
	if err := s.client.Get(ctx, path, &plans); err != nil {
		return nil, err
	}
	if len(plans) == 0 {
		return nil, NewAPIError(ErrUnknown, "empty snapshotplan response")
	}
	return &plans[0], nil
}

// SetSnapshotPlan configures the snapshot plan. Status is required; the
// time fields are required when the plan is enabled. Nil time fields are
// omitted from the form, which the API treats as "leave unchanged".
//
// POST /storagebox/{storagebox-id}/snapshotplan
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id-snapshotplan
func (s *StorageBoxService) SetSnapshotPlan(ctx context.Context, storageBoxID int, plan StorageBoxSnapshotPlan) (*StorageBoxSnapshotPlan, error) {
	path := fmt.Sprintf("/storagebox/%d/snapshotplan", storageBoxID)
	data := url.Values{}
	data.Set("status", plan.Status)
	if plan.Minute != nil {
		data.Set("minute", strconv.Itoa(*plan.Minute))
	}
	if plan.Hour != nil {
		data.Set("hour", strconv.Itoa(*plan.Hour))
	}
	if plan.DayOfWeek != nil {
		data.Set("day_of_week", strconv.Itoa(*plan.DayOfWeek))
	}
	if plan.DayOfMonth != nil {
		data.Set("day_of_month", strconv.Itoa(*plan.DayOfMonth))
	}
	if plan.Month != nil {
		data.Set("month", strconv.Itoa(*plan.Month))
	}
	data.Set("max_snapshots", strconv.Itoa(plan.MaxSnapshots))

	// The Robot endpoint inconsistently returns either a single-element
	// array `[{"snapshotplan":{...}}]` or the bare wrapped object
	// `{"snapshotplan":{...}}`. The auto-unwrap strips the wrapper in both
	// shapes, leaving us with either an array or a single object; decode
	// into an any first and inspect.
	var raw any
	if err := s.client.Post(ctx, path, data, &raw); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, NewParseError("failed to re-encode snapshotplan response", err)
	}
	if len(encoded) > 0 && encoded[0] == '[' {
		var plans []StorageBoxSnapshotPlan
		if err := json.Unmarshal(encoded, &plans); err != nil {
			return nil, NewParseError("failed to unmarshal snapshotplan", err)
		}
		if len(plans) == 0 {
			return nil, NewAPIError(ErrUnknown, "empty snapshotplan response")
		}
		return &plans[0], nil
	}
	var p StorageBoxSnapshotPlan
	if err := json.Unmarshal(encoded, &p); err != nil {
		return nil, NewParseError("failed to unmarshal snapshotplan", err)
	}
	return &p, nil
}

// ListSubAccounts returns all sub-accounts for a storage box.
//
// GET /storagebox/{storagebox-id}/subaccount
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-storagebox-storagebox-id-subaccount
func (s *StorageBoxService) ListSubAccounts(ctx context.Context, storageBoxID int) ([]StorageBoxSubAccount, error) {
	path := fmt.Sprintf("/storagebox/%d/subaccount", storageBoxID)
	var subs []StorageBoxSubAccount
	if err := s.client.Get(ctx, path, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

// CreateSubAccount creates a new sub-account. HomeDirectory is required by
// the API. The response contains the (possibly auto-generated) password.
//
// POST /storagebox/{storagebox-id}/subaccount
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id-subaccount
func (s *StorageBoxService) CreateSubAccount(ctx context.Context, storageBoxID int, in StorageBoxSubAccountInput) (*StorageBoxSubAccountCreated, error) {
	path := fmt.Sprintf("/storagebox/%d/subaccount", storageBoxID)
	data := subAccountForm(in)

	var sub StorageBoxSubAccountCreated
	if err := s.client.Post(ctx, path, data, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// UpdateSubAccount modifies a sub-account.
//
// PUT /storagebox/{storagebox-id}/subaccount/{sub-account-username}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#put-storagebox-storagebox-id-subaccount-sub-account-username
func (s *StorageBoxService) UpdateSubAccount(ctx context.Context, storageBoxID int, username string, in StorageBoxSubAccountInput) error {
	path := fmt.Sprintf("/storagebox/%d/subaccount/%s", storageBoxID, url.PathEscape(username))
	data := subAccountForm(in)
	return s.client.Put(ctx, path, data, nil)
}

// DeleteSubAccount removes a sub-account.
//
// DELETE /storagebox/{storagebox-id}/subaccount/{sub-account-username}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#delete-storagebox-storagebox-id-subaccount-sub-account-username
func (s *StorageBoxService) DeleteSubAccount(ctx context.Context, storageBoxID int, username string) error {
	path := fmt.Sprintf("/storagebox/%d/subaccount/%s", storageBoxID, url.PathEscape(username))
	return s.client.Delete(ctx, path)
}

// ResetSubAccountPassword resets a sub-account password. If newPassword is
// empty Robot generates a random one. Returns the password that is now in
// effect.
//
// POST /storagebox/{storagebox-id}/subaccount/{sub-account-username}/password
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-storagebox-storagebox-id-subaccount-sub-account-username-password
func (s *StorageBoxService) ResetSubAccountPassword(ctx context.Context, storageBoxID int, username, newPassword string) (string, error) {
	path := fmt.Sprintf("/storagebox/%d/subaccount/%s/password", storageBoxID, url.PathEscape(username))
	data := url.Values{}
	if newPassword != "" {
		data.Set("password", newPassword)
	}

	var resp storageBoxPasswordResponse
	if err := s.client.Post(ctx, path, data, &resp); err != nil {
		return "", err
	}
	return resp.Password, nil
}

func subAccountForm(in StorageBoxSubAccountInput) url.Values {
	data := url.Values{}
	if in.HomeDirectory != nil {
		data.Set("homedirectory", *in.HomeDirectory)
	}
	if in.Samba != nil {
		data.Set("samba", strconv.FormatBool(*in.Samba))
	}
	if in.SSH != nil {
		data.Set("ssh", strconv.FormatBool(*in.SSH))
	}
	if in.ExternalReachability != nil {
		data.Set("external_reachability", strconv.FormatBool(*in.ExternalReachability))
	}
	if in.Webdav != nil {
		data.Set("webdav", strconv.FormatBool(*in.Webdav))
	}
	if in.ReadOnly != nil {
		data.Set("readonly", strconv.FormatBool(*in.ReadOnly))
	}
	if in.Comment != nil {
		data.Set("comment", *in.Comment)
	}
	if in.Password != nil {
		data.Set("password", *in.Password)
	}
	return data
}
