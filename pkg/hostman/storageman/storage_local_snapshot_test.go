package storageman

import (
	"path/filepath"
	"testing"
)

func TestPlanLocalSnapshotDelete(t *testing.T) {
	dir := "/storage/snapshots/disk_snap"
	disk := "/storage/disks/disk"
	cache := "/storage/imagecache/image"
	base := filepath.Join(dir, localSnapshotBaseName)
	snap := func(id string) string { return filepath.Join(dir, id) }

	tests := []struct {
		name   string
		chain  []string
		id     string
		action LocalSnapshotDeleteAction
		parent string
		child  string
	}{
		{"promote oldest", []string{cache, snap("s1"), snap("s2"), disk}, "s1", LocalSnapshotPromote, cache, snap("s2")},
		{"rebase middle without base", []string{cache, snap("s1"), snap("s2"), snap("s3"), disk}, "s2", LocalSnapshotRebase, snap("s1"), snap("s3")},
		{"commit adjacent to base", []string{cache, base, snap("s2"), snap("s3"), disk}, "s2", LocalSnapshotCommit, base, snap("s3")},
		{"rebase middle with base", []string{cache, base, snap("s2"), snap("s3"), disk}, "s3", LocalSnapshotRebase, snap("s2"), disk},
		{"remove out of chain", []string{cache, base, snap("s2"), disk}, "old", LocalSnapshotRemove, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := PlanLocalSnapshotDelete(tt.chain, dir, tt.id)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Action != tt.action || plan.Parent != tt.parent || plan.Child != tt.child {
				t.Fatalf("unexpected plan: %#v", plan)
			}
		})
	}
}
