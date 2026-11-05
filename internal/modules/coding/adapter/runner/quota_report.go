package runner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// formatBytes formats bytes in human-readable B, KiB, MiB, or GiB.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}

// cleanupAbandonedBuildStages removes only nested stage directories from retained
// build outputs. The caller holds buildOutputMu so active runs cannot be removed.
func cleanupAbandonedBuildStages(dir string) {
	rootInfo, err := os.Lstat(dir)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".build-output-") {
			continue
		}
		output := filepath.Join(root, entry.Name())
		outputReal, err := filepath.EvalSymlinks(output)
		if err != nil || filepath.Dir(outputReal) != root {
			continue
		}
		if _, active := activeBuildOutputs[outputReal]; active {
			continue
		}
		stage := filepath.Join(outputReal, ".build-stage")
		stageInfo, err := os.Lstat(stage)
		if err != nil || !stageInfo.IsDir() || stageInfo.Mode()&os.ModeSymlink != 0 {
			continue
		}
		stageReal, err := filepath.EvalSymlinks(stage)
		if err != nil || filepath.Dir(stageReal) != outputReal {
			continue
		}
		_ = os.RemoveAll(stageReal)
	}
}

// pruneOldArtifactOutputs retains only the most recent keepCount output folders.
func pruneOldArtifactOutputs(dir string, keepCount int) {
	if keepCount < 0 {
		keepCount = 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type folderInfo struct {
		path    string
		modTime int64
	}
	var artifacts []folderInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".build-output-") || strings.HasPrefix(name, ".artifact-output-") {
			full := filepath.Clean(filepath.Join(dir, name))
			if _, active := activeBuildOutputs[buildOutputKey(full)]; active {
				continue
			}
			if info, infoErr := entry.Info(); infoErr == nil {
				artifacts = append(artifacts, folderInfo{path: full, modTime: info.ModTime().UnixNano()})
			}
		}
	}
	if len(artifacts) <= keepCount {
		return
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].modTime < artifacts[j].modTime
	})
	toRemove := len(artifacts) - keepCount
	for i := 0; i < toRemove; i++ {
		_ = os.RemoveAll(artifacts[i].path)
	}
}

// workspaceQuotaReport generates an actionable diagnostic breakdown when the workspace
// exceeds quota, detailing source code, preview runs, test artifacts, and largest directories.
func workspaceQuotaReport(dir string, limit int64) string {
	var (
		totalBytes, sourceBytes int64
		previewRunsCount        int
		previewRunsBytes        int64
		playwrightCount         int
		playwrightBytes         int64
		buildOutputCount        int
		buildOutputBytes        int64
		depsBytes               int64
	)
	dirSizes := make(map[string]int64)

	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil || rel == "." {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		topDir := parts[0]

		if entry.IsDir() {
			if rel == "data/preview-runs" {
				if sub, err := os.ReadDir(path); err == nil {
					previewRunsCount = len(sub)
				}
			}
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		sz := info.Size()
		totalBytes += sz
		dirSizes[topDir] += sz

		switch {
		case strings.HasPrefix(rel, "data/preview-runs/"):
			previewRunsBytes += sz
		case strings.HasPrefix(topDir, ".playwright") || topDir == "playwright-report" || topDir == "test-results":
			playwrightBytes += sz
			playwrightCount++
		case strings.HasPrefix(topDir, ".build-output-") || strings.HasPrefix(topDir, ".artifact-output-"):
			buildOutputBytes += sz
			buildOutputCount++
		case topDir == "node_modules" || topDir == ".nuxt" || topDir == ".output" || topDir == "dist" || topDir == ".next":
			depsBytes += sz
		default:
			sourceBytes += sz
		}
		return nil
	})

	type dirEntry struct {
		name string
		size int64
	}
	var sortedDirs []dirEntry
	for name, sz := range dirSizes {
		sortedDirs = append(sortedDirs, dirEntry{name: name, size: sz})
	}
	sort.Slice(sortedDirs, func(i, j int) bool { return sortedDirs[i].size > sortedDirs[j].size })

	var out strings.Builder
	out.WriteString(fmt.Sprintf("The workspace is over its size limit (%s used / %s limit).\n",
		formatBytes(totalBytes), formatBytes(limit)))
	out.WriteString("Storage Breakdown:\n")
	out.WriteString(fmt.Sprintf("- Source code: %s\n", formatBytes(sourceBytes)))
	if previewRunsCount > 0 || previewRunsBytes > 0 {
		out.WriteString(fmt.Sprintf("- Preview runs (data/preview-runs): %d runs (%s)\n",
			previewRunsCount, formatBytes(previewRunsBytes)))
	}
	if playwrightCount > 0 || playwrightBytes > 0 {
		out.WriteString(fmt.Sprintf("- Playwright test artifacts: %d entries (%s)\n",
			playwrightCount, formatBytes(playwrightBytes)))
	}
	if buildOutputCount > 0 || buildOutputBytes > 0 {
		out.WriteString(fmt.Sprintf("- Build & artifact outputs (.build-output-*): %d files (%s)\n",
			buildOutputCount, formatBytes(buildOutputBytes)))
	}
	if depsBytes > 0 {
		out.WriteString(fmt.Sprintf("- Dependencies & build targets (node_modules, .output, dist): %s\n",
			formatBytes(depsBytes)))
	}
	if len(sortedDirs) > 0 {
		out.WriteString("Largest directories:\n")
		maxTop := min(len(sortedDirs), 5)
		for i := 0; i < maxTop; i++ {
			out.WriteString(fmt.Sprintf("  %d. %s: %s\n", i+1, sortedDirs[i].name, formatBytes(sortedDirs[i].size)))
		}
	}
	out.WriteString("Recommendation: Clean old preview-runs or build artifacts. Do not delete databases, caches, or project source.")
	return out.String()
}
