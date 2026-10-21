package access

import "testing"

func TestPathAllowed(t *testing.T) {
	p := Policy{AllowedPaths: []string{"/data/free/"}}
	cases := map[string]bool{
		"/data/free":           true,
		"/data/free/a.mp4":     true,
		"/data/freebies/a.mp4": false,
		"/data/premium/a.mp4":  false,
	}
	for path, want := range cases {
		if got := p.PathAllowed(path); got != want {
			t.Errorf("%s: got %v want %v", path, got, want)
		}
	}
	if !(Policy{}).PathAllowed("/anything") {
		t.Error("empty policy should allow everything")
	}
}

func TestRoles(t *testing.T) {
	if !RoleAdmin.AtLeast(RoleEditor) || RoleViewer.AtLeast(RoleEditor) {
		t.Error("role ranks wrong")
	}
	admin := &Principal{Role: RoleAdmin, Policy: Policy{AllowedPaths: []string{"/x"}}}
	if admin.RestrictsContent() {
		t.Error("admins are never restricted")
	}
	viewer := &Principal{UserID: 2, Role: RoleViewer, Policy: Policy{BlockedTagIDs: []int{1}}}
	if !viewer.RestrictsContent() {
		t.Error("viewer with policy should be restricted")
	}
	var none *Principal
	if none.RestrictsContent() || !none.CanDownload() {
		t.Error("no principal means full access")
	}
}
