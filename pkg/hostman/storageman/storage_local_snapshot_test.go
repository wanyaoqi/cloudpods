package storageman

import (
	"path/filepath"
	"testing"
)

func TestSnapshotBasePath(t *testing.T) {
	dir := "/storage/snapshots/disk-id_snapshots"
	disk := "/storage/disks/disk-id"
	base := filepath.Join(dir, "disk-id_snap_base")
	legacyBase := filepath.Join(dir, legacySnapshotBaseName)

	if got := snapshotBasePath(dir, disk, base); got != base {
		t.Fatalf("expected base %q, got %q", base, got)
	}
	if got := snapshotBasePath(dir, disk, legacyBase); got != legacyBase {
		t.Fatalf("expected legacy base %q, got %q", legacyBase, got)
	}
	if got := snapshotBasePath(dir, disk, filepath.Join(dir, "other_snap_base")); got != "" {
		t.Fatalf("must not accept another disk's base: %q", got)
	}
}

func TestPrefixSnapshotIds(t *testing.T) {
	ids := prefixSnapshotIds([]string{"s1", "snap_base", "disk_snap_base"})
	want := []string{"snap_s1", "snap_base", "disk_snap_base"}
	if len(ids) != len(want) {
		t.Fatalf("expected %v, got %v", want, ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, ids)
		}
	}
}

func TestResolveLocalSnapshotDeleteEdges(t *testing.T) {
	dir := "/storage/snapshots/disk_snap"
	target := filepath.Join(dir, "s2")
	parent := filepath.Join(dir, "s1")
	child := filepath.Join(dir, "s3")
	base := filepath.Join(dir, "disk_snap_base")

	plan := resolveLocalSnapshotDeleteEdges(target, parent, child, base, false)
	if plan.Action != LocalSnapshotRebase || plan.Parent != parent || plan.Child != child {
		t.Fatalf("unexpected disconnected-chain plan: %#v", plan)
	}

	plan = resolveLocalSnapshotDeleteEdges(target, base, child, base, true)
	if plan.Action != LocalSnapshotCommit || plan.Base != base {
		t.Fatalf("expected base commit, got %#v", plan)
	}

	legacyBase := filepath.Join(dir, legacySnapshotBaseName)
	plan = resolveLocalSnapshotDeleteEdges(target, legacyBase, child, legacyBase, true)
	if plan.Action != LocalSnapshotCommit || plan.Base != legacyBase {
		t.Fatalf("expected legacy-base commit, got %#v", plan)
	}

	otherBase := filepath.Join(dir, "other-disk_snap_base")
	plan = resolveLocalSnapshotDeleteEdges(target, otherBase, child, base, true)
	if plan.Action == LocalSnapshotCommit {
		t.Fatalf("must not commit into another disk's base: %#v", plan)
	}

	plan = resolveLocalSnapshotDeleteEdges(target, "/storage/imagecache/image", child, base, false)
	if plan.Action != LocalSnapshotPromote {
		t.Fatalf("expected segment-head promotion, got %#v", plan)
	}

	plan = resolveLocalSnapshotDeleteEdges(target, "/storage/imagecache/image", child, base, true)
	if plan.Action != LocalSnapshotRebase {
		t.Fatalf("expected safe rebase when another chain owns base, got %#v", plan)
	}
}
