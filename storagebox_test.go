package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/v2/internal/spectest"
)

// storageBoxListFixture mirrors the abridged example from the doc's
// GET /storagebox response, which omits the detail-only fields (disk
// usage/webdav/samba/ssh/etc) that only appear on GET/POST /storagebox/{id}.
func storageBoxListFixture() map[string]any {
	return map[string]any{
		"storagebox": map[string]any{
			"id":            123456,
			"login":         "u12345",
			"name":          "Backup Server 1",
			"product":       "BX60",
			"cancelled":     false,
			"locked":        false,
			"location":      "FSN1",
			"linked_server": 123456,
			"paid_until":    "2015-10-23",
		},
	}
}

func storageBoxFixture() map[string]any {
	return map[string]any{
		"storagebox": map[string]any{
			"id":                    123456,
			"login":                 "u12345",
			"name":                  "Backup Server 1",
			"product":               "BX60",
			"cancelled":             false,
			"locked":                false,
			"location":              "FSN1",
			"linked_server":         123456,
			"paid_until":            "2015-10-23",
			"disk_quota":            10240000,
			"disk_usage":            900,
			"disk_usage_data":       500,
			"disk_usage_snapshots":  400,
			"webdav":                true,
			"samba":                 true,
			"ssh":                   true,
			"external_reachability": true,
			"zfs":                   false,
			"server":                "u12345.your-storagebox.de",
			"host_system":           "FSN1-BX355",
		},
	}
}

func TestStorageBoxService_List(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox" {
			t.Errorf("expected '/storagebox', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{storageBoxListFixture()})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	boxes, err := client.StorageBox.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(boxes) != 1 || boxes[0].ID != 123456 {
		t.Errorf("unexpected: %+v", boxes)
	}
}

func TestStorageBoxService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox/123456" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(storageBoxFixture())
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	box, err := client.StorageBox.Get(context.Background(), 123456)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if box.HostSystem != "FSN1-BX355" {
		t.Errorf("host_system: %q", box.HostSystem)
	}
	if !box.Webdav {
		t.Error("expected webdav true")
	}
}

func TestStorageBoxService_Update(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("storagebox_name") != "renamed" {
			t.Errorf("storagebox_name: %q", r.FormValue("storagebox_name"))
		}
		if r.FormValue("samba") != "true" {
			t.Errorf("samba: %q", r.FormValue("samba"))
		}
		if r.FormValue("zfs") != "false" {
			t.Errorf("zfs: %q", r.FormValue("zfs"))
		}
		// Unset field must not be sent.
		if _, present := r.PostForm["webdav"]; present {
			t.Errorf("webdav should be omitted")
		}
		_ = json.NewEncoder(w).Encode(storageBoxFixture())
	})))
	defer server.Close()

	name := "renamed"
	samba := true
	zfs := false
	client := NewClient("u", "p", WithBaseURL(server.URL))
	box, err := client.StorageBox.Update(context.Background(), 123456, StorageBoxUpdate{
		Name:  &name,
		Samba: &samba,
		ZFS:   &zfs,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if box.ID != 123456 {
		t.Errorf("id: %d", box.ID)
	}
}

func TestStorageBoxService_ResetPassword_Generated(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/password" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, present := r.PostForm["password"]; present {
			t.Errorf("password should be omitted when empty")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"password": "h1cgLgZYJsyGl0JK"})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	pw, err := client.StorageBox.ResetPassword(context.Background(), 123456, "")
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if pw != "h1cgLgZYJsyGl0JK" {
		t.Errorf("password: %q", pw)
	}
}

func TestStorageBoxService_ResetPassword_Custom(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("password") != "TVUlzspV3YhfSJch" {
			t.Errorf("password: %q", r.FormValue("password"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"password": "TVUlzspV3YhfSJch"})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	pw, err := client.StorageBox.ResetPassword(context.Background(), 123456, "TVUlzspV3YhfSJch")
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if pw != "TVUlzspV3YhfSJch" {
		t.Errorf("password: %q", pw)
	}
}

func snapshotFixture() map[string]any {
	return map[string]any{
		"snapshot": map[string]any{
			"name":            "2015-12-21T12-40-38",
			"timestamp":       "2015-12-21T13:40:38+00:00",
			"size":            400,
			"filesystem_size": 12345,
			"automatic":       false,
			"comment":         "Test-Snapshot",
		},
	}
}

func TestStorageBoxService_ListSnapshots(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox/123456/snapshot" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{snapshotFixture()})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	snaps, err := client.StorageBox.ListSnapshots(context.Background(), 123456)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snaps) != 1 || snaps[0].Name != "2015-12-21T12-40-38" {
		t.Errorf("unexpected: %+v", snaps)
	}
	if snaps[0].FilesystemSize != 12345 {
		t.Errorf("filesystem_size: %d", snaps[0].FilesystemSize)
	}
}

func TestStorageBoxService_CreateSnapshot(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/snapshot" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"snapshot": map[string]any{
				"name":      "2015-12-21T13-13-03",
				"timestamp": "2015-12-21T13:13:03+00:00",
				"size":      400,
			},
		})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	snap, err := client.StorageBox.CreateSnapshot(context.Background(), 123456)
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if snap.Name != "2015-12-21T13-13-03" {
		t.Errorf("name: %q", snap.Name)
	}
}

func TestStorageBoxService_DeleteSnapshot(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/snapshot/2015-12-21T13-13-03" {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	if err := client.StorageBox.DeleteSnapshot(context.Background(), 123456, "2015-12-21T13-13-03"); err != nil {
		t.Fatalf("DeleteSnapshot: %v", err)
	}
}

func TestStorageBoxService_RestoreSnapshot(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/snapshot/2015-12-21T13-13-03" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("revert") != "true" {
			t.Errorf("revert: %q", r.FormValue("revert"))
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	if err := client.StorageBox.RestoreSnapshot(context.Background(), 123456, "2015-12-21T13-13-03"); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
}

func TestStorageBoxService_SetSnapshotComment(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/snapshot/snap1/comment" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("comment") != "hello" {
			t.Errorf("comment: %q", r.FormValue("comment"))
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	if err := client.StorageBox.SetSnapshotComment(context.Background(), 123456, "snap1", "hello"); err != nil {
		t.Fatalf("SetSnapshotComment: %v", err)
	}
}

func snapshotPlanFixture() map[string]any {
	return map[string]any{
		"snapshotplan": map[string]any{
			"status":        "enabled",
			"minute":        5,
			"hour":          12,
			"day_of_week":   2,
			"day_of_month":  nil,
			"month":         nil,
			"max_snapshots": 2,
		},
	}
}

func TestStorageBoxService_GetSnapshotPlan(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox/123456/snapshotplan" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{snapshotPlanFixture()})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	plan, err := client.StorageBox.GetSnapshotPlan(context.Background(), 123456)
	if err != nil {
		t.Fatalf("GetSnapshotPlan: %v", err)
	}
	if plan.Status != "enabled" {
		t.Errorf("status: %q", plan.Status)
	}
	if plan.Minute == nil || *plan.Minute != 5 {
		t.Errorf("minute: %+v", plan.Minute)
	}
	if plan.DayOfMonth != nil {
		t.Errorf("day_of_month should be nil")
	}
	if plan.MaxSnapshots != 2 {
		t.Errorf("max_snapshots: %d", plan.MaxSnapshots)
	}
}

func TestStorageBoxService_SetSnapshotPlan(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/snapshotplan" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("status") != "enabled" {
			t.Errorf("status: %q", r.FormValue("status"))
		}
		if r.FormValue("minute") != "5" {
			t.Errorf("minute: %q", r.FormValue("minute"))
		}
		if r.FormValue("hour") != "12" {
			t.Errorf("hour: %q", r.FormValue("hour"))
		}
		if r.FormValue("day_of_week") != "2" {
			t.Errorf("day_of_week: %q", r.FormValue("day_of_week"))
		}
		if r.FormValue("max_snapshots") != "2" {
			t.Errorf("max_snapshots: %q", r.FormValue("max_snapshots"))
		}
		// Nil pointers should not be sent.
		if _, present := r.PostForm["day_of_month"]; present {
			t.Errorf("day_of_month should be omitted")
		}
		if _, present := r.PostForm["month"]; present {
			t.Errorf("month should be omitted")
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{snapshotPlanFixture()})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	minute := 5
	hour := 12
	dow := 2
	plan, err := client.StorageBox.SetSnapshotPlan(context.Background(), 123456, StorageBoxSnapshotPlan{
		Status:       "enabled",
		Minute:       &minute,
		Hour:         &hour,
		DayOfWeek:    &dow,
		MaxSnapshots: 2,
	})
	if err != nil {
		t.Fatalf("SetSnapshotPlan: %v", err)
	}
	if plan.Status != "enabled" || plan.MaxSnapshots != 2 {
		t.Errorf("unexpected: %+v", plan)
	}
}

func subAccountFixture() map[string]any {
	return map[string]any{
		"subaccount": map[string]any{
			"username":              "u2342-sub1",
			"accountid":             "u2342",
			"server":                "u12345-sub1.your-storagebox.de",
			"homedirectory":         "test",
			"samba":                 true,
			"ssh":                   true,
			"external_reachability": true,
			"webdav":                false,
			"readonly":              false,
			"createtime":            "2017-05-24 13:16:45",
			"comment":               "Test-comment",
		},
	}
}

func TestStorageBoxService_ListSubAccounts(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox/123456/subaccount" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{subAccountFixture()})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	subs, err := client.StorageBox.ListSubAccounts(context.Background(), 123456)
	if err != nil {
		t.Fatalf("ListSubAccounts: %v", err)
	}
	if len(subs) != 1 || subs[0].Username != "u2342-sub1" {
		t.Errorf("unexpected: %+v", subs)
	}
	if subs[0].HomeDirectory != "test" {
		t.Errorf("homedirectory: %q", subs[0].HomeDirectory)
	}
	if subs[0].AccountID != "u2342" {
		t.Errorf("accountid: %q", subs[0].AccountID)
	}
	if subs[0].CreateTime != "2017-05-24 13:16:45" {
		t.Errorf("createtime: %q", subs[0].CreateTime)
	}
}

func TestStorageBoxService_CreateSubAccount(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/subaccount" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("homedirectory") != "test" {
			t.Errorf("homedirectory: %q", r.FormValue("homedirectory"))
		}
		if r.FormValue("samba") != "true" {
			t.Errorf("samba: %q", r.FormValue("samba"))
		}
		if r.FormValue("ssh") != "false" {
			t.Errorf("ssh: %q", r.FormValue("ssh"))
		}
		if r.FormValue("readonly") != "true" {
			t.Errorf("readonly: %q", r.FormValue("readonly"))
		}
		if r.FormValue("comment") != "hi" {
			t.Errorf("comment: %q", r.FormValue("comment"))
		}
		if r.FormValue("password") != "MySecret123!" {
			t.Errorf("password: %q", r.FormValue("password"))
		}
		// Unset field
		if _, present := r.PostForm["webdav"]; present {
			t.Errorf("webdav should be omitted")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"subaccount": map[string]any{
				"username":      "u2342-sub1",
				"password":      "MySecret123!",
				"accountid":     "u2342",
				"server":        "u12345-sub1.your-storagebox.de",
				"homedirectory": "test",
			},
		})
	})))
	defer server.Close()

	hd := "test"
	samba := true
	ssh := false
	ro := true
	comment := "hi"
	pw := "MySecret123!"
	client := NewClient("u", "p", WithBaseURL(server.URL))
	sub, err := client.StorageBox.CreateSubAccount(context.Background(), 123456, StorageBoxSubAccountInput{
		HomeDirectory: &hd,
		Samba:         &samba,
		SSH:           &ssh,
		ReadOnly:      &ro,
		Comment:       &comment,
		Password:      &pw,
	})
	if err != nil {
		t.Fatalf("CreateSubAccount: %v", err)
	}
	if sub.Username != "u2342-sub1" || sub.Password != "MySecret123!" {
		t.Errorf("unexpected: %+v", sub)
	}
}

func TestStorageBoxService_UpdateSubAccount(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/subaccount/u2342-sub1" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("homedirectory") != "test2" {
			t.Errorf("homedirectory: %q", r.FormValue("homedirectory"))
		}
		if r.FormValue("external_reachability") != "false" {
			t.Errorf("external_reachability: %q", r.FormValue("external_reachability"))
		}
		// Only the two fields above were set.
		if _, present := r.PostForm["samba"]; present {
			t.Errorf("samba should be omitted")
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	hd := "test2"
	ext := false
	client := NewClient("u", "p", WithBaseURL(server.URL))
	if err := client.StorageBox.UpdateSubAccount(context.Background(), 123456, "u2342-sub1", StorageBoxSubAccountInput{
		HomeDirectory:        &hd,
		ExternalReachability: &ext,
	}); err != nil {
		t.Fatalf("UpdateSubAccount: %v", err)
	}
}

func TestStorageBoxService_DeleteSubAccount(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/subaccount/u2342-sub1" {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	if err := client.StorageBox.DeleteSubAccount(context.Background(), 123456, "u2342-sub1"); err != nil {
		t.Fatalf("DeleteSubAccount: %v", err)
	}
}

func TestStorageBoxService_ResetSubAccountPassword_Generated(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/storagebox/123456/subaccount/u2342-sub1/password" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, present := r.PostForm["password"]; present {
			t.Errorf("password must be omitted when empty")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"password": "h1cgLgZYJsyGl0JK"})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	pw, err := client.StorageBox.ResetSubAccountPassword(context.Background(), 123456, "u2342-sub1", "")
	if err != nil {
		t.Fatalf("ResetSubAccountPassword: %v", err)
	}
	if pw != "h1cgLgZYJsyGl0JK" {
		t.Errorf("password: %q", pw)
	}
}

func TestStorageBoxService_ResetSubAccountPassword_Custom(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.FormValue("password") != "TVUlzspV3YhfSJch" {
			t.Errorf("password: %q", r.FormValue("password"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"password": "TVUlzspV3YhfSJch"})
	})))
	defer server.Close()

	client := NewClient("u", "p", WithBaseURL(server.URL))
	pw, err := client.StorageBox.ResetSubAccountPassword(context.Background(), 123456, "u2342-sub1", "TVUlzspV3YhfSJch")
	if err != nil {
		t.Fatalf("ResetSubAccountPassword: %v", err)
	}
	if pw != "TVUlzspV3YhfSJch" {
		t.Errorf("password: %q", pw)
	}
}

// TestStorageBoxService_List_Empty verifies an empty array response decodes to an empty slice, not an error.
func TestStorageBoxService_List_Empty(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox" {
			t.Errorf("expected path '/storagebox', got '%s'", r.URL.Path)
		}
		_, _ = w.Write([]byte("[]"))
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	got, err := client.StorageBox.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if got == nil {
		t.Error("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}

// TestStorageBoxService_ListSnapshots_Empty verifies an empty snapshot list
// decodes to an empty slice.
func TestStorageBoxService_ListSnapshots_Empty(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox/42/snapshot" {
			t.Errorf("expected path '/storagebox/42/snapshot', got '%s'", r.URL.Path)
		}
		_, _ = w.Write([]byte("[]"))
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	got, err := client.StorageBox.ListSnapshots(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListSnapshots returned error: %v", err)
	}
	if got == nil {
		t.Error("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}

// TestStorageBoxService_ListSubAccounts_Empty verifies an empty sub-account
// list decodes to an empty slice.
func TestStorageBoxService_ListSubAccounts_Empty(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storagebox/42/subaccount" {
			t.Errorf("expected path '/storagebox/42/subaccount', got '%s'", r.URL.Path)
		}
		_, _ = w.Write([]byte("[]"))
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	got, err := client.StorageBox.ListSubAccounts(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListSubAccounts returned error: %v", err)
	}
	if got == nil {
		t.Error("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}

// TestStorageBoxService_ErrorHandling verifies non-2xx responses surface as
// errors across the service's verbs (GET/POST/DELETE) and return shapes. It is
// not wrapped with spectest.Handler because the error bodies are generic, not
// per-operation response fixtures.
func TestStorageBoxService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		call       func(*Client, context.Context) error
	}{
		{"List error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.List(ctx)
			return err
		}},
		{"Get not found", http.StatusNotFound, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.Get(ctx, 42)
			return err
		}},
		{"Update unauthorized", http.StatusUnauthorized, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.Update(ctx, 42, StorageBoxUpdate{})
			return err
		}},
		{"ResetPassword error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.ResetPassword(ctx, 42, "new-pass")
			return err
		}},
		{"ListSnapshots error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.ListSnapshots(ctx, 42)
			return err
		}},
		{"CreateSnapshot error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.CreateSnapshot(ctx, 42)
			return err
		}},
		{"DeleteSnapshot error", http.StatusNotFound, func(c *Client, ctx context.Context) error {
			return c.StorageBox.DeleteSnapshot(ctx, 42, "snap1")
		}},
		{"GetSnapshotPlan error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.GetSnapshotPlan(ctx, 42)
			return err
		}},
		{"SetSnapshotPlan error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.SetSnapshotPlan(ctx, 42, StorageBoxSnapshotPlan{})
			return err
		}},
		{"ListSubAccounts error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.ListSubAccounts(ctx, 42)
			return err
		}},
		{"CreateSubAccount error", http.StatusInternalServerError, func(c *Client, ctx context.Context) error {
			_, err := c.StorageBox.CreateSubAccount(ctx, 42, StorageBoxSubAccountInput{})
			return err
		}},
		{"DeleteSubAccount not found", http.StatusNotFound, func(c *Client, ctx context.Context) error {
			return c.StorageBox.DeleteSubAccount(ctx, 42, "user1")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"status":  tt.statusCode,
						"code":    "ERROR",
						"message": "test error",
					},
				})
			}))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			if err := tt.call(client, context.Background()); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
