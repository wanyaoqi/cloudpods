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

func TestResolveLocalSnapshotDeleteEdgesForDisconnectedChain(t *testing.T) {
	dir := "/storage/snapshots/disk_snap"
	target := filepath.Join(dir, "s2")
	parent := filepath.Join(dir, "s1")
	child := filepath.Join(dir, "s3")
	base := filepath.Join(dir, "disk_snap_base")

	plan, err := resolveLocalSnapshotDeleteEdges(target, parent, child, target, parent, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotRebase || plan.Parent != parent || plan.Child != child {
		t.Fatalf("unexpected disconnected-chain plan: %#v", plan)
	}

	plan, err = resolveLocalSnapshotDeleteEdges(target, base, child, target, "", base, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotCommit || plan.Base != base {
		t.Fatalf("expected base commit, got %#v", plan)
	}

	legacyBase := filepath.Join(dir, legacySnapshotBaseName)
	plan, err = resolveLocalSnapshotDeleteEdges(target, legacyBase, child, target, "", base, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotCommit || plan.Base != legacyBase {
		t.Fatalf("expected legacy-base commit, got %#v", plan)
	}

	plan, err = resolveLocalSnapshotDeleteEdges(target, "/storage/imagecache/image", child, target, parent, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotPromote {
		t.Fatalf("expected segment-head promotion, got %#v", plan)
	}

	if _, err = resolveLocalSnapshotDeleteEdges(target, filepath.Join(dir, "wrong"), child, target, parent, base, false); err == nil {
		t.Fatal("expected managed snapshot parent mismatch")
	}

	plan, err = resolveLocalSnapshotDeleteEdges(target, parent, child, "/other/backing", parent, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotRemove {
		t.Fatalf("expected unreferenced removal, got %#v", plan)
	}
	if plan.Parent != parent {
		t.Fatalf("remove plan must preserve parent for base cleanup, got %#v", plan)
	}

	plan, err = resolveLocalSnapshotDeleteEdges(target, "/storage/imagecache/image", child, target, parent, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotRebase {
		t.Fatalf("expected safe rebase when another chain owns base, got %#v", plan)
	}
}
