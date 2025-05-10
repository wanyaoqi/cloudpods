package uefi

import (
	"io"
	"os"
	"strings"
)

// Contains checks if a string contains any of the substrings
func Contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// MatchBootEntries matches disk and CDROM boot entries
func MatchBootEntries(entries []BootEntry, diskPaths, cdromPaths []string) ([]BootEntry, []BootEntry) {
	var diskEntries, cdromEntries []BootEntry
	
	for _, entry := range entries {
		if entry.DevType == "HD" {
			// If no disk paths specified, add all disk entries
			if len(diskPaths) == 0 {
				diskEntries = append(diskEntries, entry)
				continue
			}
			
			// Check if entry matches any of the disk paths
			for _, diskPath := range diskPaths {
				if Contains(entry.Name, diskPath) || Contains(entry.Path, diskPath) {
					diskEntries = append(diskEntries, entry)
					break
				}
			}
		} else if entry.DevType == "CDROM" {
			// If no CDROM paths specified, add all CDROM entries
			if len(cdromPaths) == 0 {
				cdromEntries = append(cdromEntries, entry)
				continue
			}
			
			// Check if entry matches any of the CDROM paths
			for _, cdromPath := range cdromPaths {
				if Contains(entry.Name, cdromPath) || Contains(entry.Path, cdromPath) {
					cdromEntries = append(cdromEntries, entry)
					break
				}
			}
		}
	}
	
	return diskEntries, cdromEntries
}

// BuildBootOrder builds a boot order based on priorities
func BuildBootOrder(diskEntries, cdromEntries []BootEntry, diskPriority, cdromPriority int32) []string {
	// Sort entries by priority
	type priorityEntry struct {
		entry    BootEntry
		priority int32
	}
	
	var priorityEntries []priorityEntry
	
	// Add disk boot entries
	if diskPriority > 0 {
		for _, entry := range diskEntries {
			priorityEntries = append(priorityEntries, priorityEntry{
				entry:    entry,
				priority: diskPriority,
			})
		}
	}
	
	// Add CDROM boot entries
	if cdromPriority > 0 {
		for _, entry := range cdromEntries {
			priorityEntries = append(priorityEntries, priorityEntry{
				entry:    entry,
				priority: cdromPriority,
			})
		}
	}
	
	// Sort by priority (higher priority first)
	sort.Slice(priorityEntries, func(i, j int) bool {
		return priorityEntries[i].priority > priorityEntries[j].priority
	})
	
	// Build boot order
	var bootOrder []string
	for _, pe := range priorityEntries {
		// Extract the numeric part of the ID (e.g., "0002" from "Boot0002")
		idNumber := strings.TrimPrefix(pe.entry.ID, "Boot")
		bootOrder = append(bootOrder, idNumber)
	}
	
	return bootOrder
} 