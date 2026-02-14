package hrobot

import (
	"context"
	"fmt"
	"net/url"
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
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Size      int    `json:"size"`
	Comment   string `json:"comment"`
}

// StorageBoxSnapshotPlan represents the snapshot schedule for a storage box.
type StorageBoxSnapshotPlan struct {
	Status       string `json:"status"`
	Minute       int    `json:"minute"`
	Hour         int    `json:"hour"`
	DayOfWeek    int    `json:"day_of_week"`
	DayOfMonth   int    `json:"day_of_month"`
	Month        int    `json:"month"`
	MaxSnapshots int    `json:"max_snapshots"`
}

// StorageBoxSubAccount represents a sub-account for a storage box.
type StorageBoxSubAccount struct {
	Username             string `json:"username"`
	AccountID            string `json:"accountid"`
	Server               string `json:"server"`
	HostSystem           string `json:"hotsystem"`
	ExternalReachability bool   `json:"external_reachability"`
	SSH                  bool   `json:"ssh"`
	Webdav               bool   `json:"webdav"`
	Samba                bool   `json:"samba"`
	CreateTime           string `json:"createtime"`
	HomeDir              string `json:"home_directory"`
	ReadOnly             bool   `json:"readonly"`
	Comment              string `json:"comment"`
}

// List returns all storage boxes.
func (s *StorageBoxService) List(ctx context.Context) ([]StorageBox, error) {
	return nil, errNotImplemented
}

// Get returns details for a specific storage box.
func (s *StorageBoxService) Get(ctx context.Context, storageBoxID int) (*StorageBox, error) {
	_ = fmt.Sprintf("/storagebox/%d", storageBoxID)
	return nil, errNotImplemented
}

// Update modifies storage box settings.
func (s *StorageBoxService) Update(ctx context.Context, storageBoxID int, name string) (*StorageBox, error) {
	_ = fmt.Sprintf("/storagebox/%d", storageBoxID)
	return nil, errNotImplemented
}

// ResetPassword updates the storage box account password.
func (s *StorageBoxService) ResetPassword(ctx context.Context, storageBoxID int) (*StorageBox, error) {
	_ = fmt.Sprintf("/storagebox/%d/password", storageBoxID)
	return nil, errNotImplemented
}

// ListSnapshots returns all snapshots for a storage box.
func (s *StorageBoxService) ListSnapshots(ctx context.Context, storageBoxID int) ([]StorageBoxSnapshot, error) {
	_ = fmt.Sprintf("/storagebox/%d/snapshot", storageBoxID)
	return nil, errNotImplemented
}

// CreateSnapshot creates a new snapshot.
func (s *StorageBoxService) CreateSnapshot(ctx context.Context, storageBoxID int) error {
	_ = fmt.Sprintf("/storagebox/%d/snapshot", storageBoxID)
	return errNotImplemented
}

// DeleteSnapshot removes a snapshot.
func (s *StorageBoxService) DeleteSnapshot(ctx context.Context, storageBoxID int, snapshotName string) error {
	_ = fmt.Sprintf("/storagebox/%d/snapshot/%s", storageBoxID, url.PathEscape(snapshotName))
	return errNotImplemented
}

// RestoreSnapshot restores a snapshot.
func (s *StorageBoxService) RestoreSnapshot(ctx context.Context, storageBoxID int, snapshotName string) error {
	_ = fmt.Sprintf("/storagebox/%d/snapshot/%s", storageBoxID, url.PathEscape(snapshotName))
	return errNotImplemented
}

// SetSnapshotComment sets a comment on a snapshot.
func (s *StorageBoxService) SetSnapshotComment(ctx context.Context, storageBoxID int, snapshotName string, comment string) error {
	_ = fmt.Sprintf("/storagebox/%d/snapshot/%s/comment", storageBoxID, url.PathEscape(snapshotName))
	return errNotImplemented
}

// GetSnapshotPlan retrieves the snapshot plan configuration.
func (s *StorageBoxService) GetSnapshotPlan(ctx context.Context, storageBoxID int) (*StorageBoxSnapshotPlan, error) {
	_ = fmt.Sprintf("/storagebox/%d/snapshotplan", storageBoxID)
	return nil, errNotImplemented
}

// SetSnapshotPlan configures the snapshot plan.
func (s *StorageBoxService) SetSnapshotPlan(ctx context.Context, storageBoxID int, plan StorageBoxSnapshotPlan) (*StorageBoxSnapshotPlan, error) {
	_ = fmt.Sprintf("/storagebox/%d/snapshotplan", storageBoxID)
	return nil, errNotImplemented
}

// ListSubAccounts returns all sub-accounts for a storage box.
func (s *StorageBoxService) ListSubAccounts(ctx context.Context, storageBoxID int) ([]StorageBoxSubAccount, error) {
	_ = fmt.Sprintf("/storagebox/%d/subaccount", storageBoxID)
	return nil, errNotImplemented
}

// CreateSubAccount creates a new sub-account.
func (s *StorageBoxService) CreateSubAccount(ctx context.Context, storageBoxID int) (*StorageBoxSubAccount, error) {
	_ = fmt.Sprintf("/storagebox/%d/subaccount", storageBoxID)
	return nil, errNotImplemented
}

// UpdateSubAccount modifies a sub-account.
func (s *StorageBoxService) UpdateSubAccount(ctx context.Context, storageBoxID int, username string) (*StorageBoxSubAccount, error) {
	_ = fmt.Sprintf("/storagebox/%d/subaccount/%s", storageBoxID, url.PathEscape(username))
	return nil, errNotImplemented
}

// DeleteSubAccount removes a sub-account.
func (s *StorageBoxService) DeleteSubAccount(ctx context.Context, storageBoxID int, username string) error {
	_ = fmt.Sprintf("/storagebox/%d/subaccount/%s", storageBoxID, url.PathEscape(username))
	return errNotImplemented
}

// ResetSubAccountPassword updates a sub-account password.
func (s *StorageBoxService) ResetSubAccountPassword(ctx context.Context, storageBoxID int, username string) error {
	_ = fmt.Sprintf("/storagebox/%d/subaccount/%s/password", storageBoxID, url.PathEscape(username))
	return errNotImplemented
}
