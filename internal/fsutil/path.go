package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve under root. relative may be "" or "a/b". Rejects escapes.
func Resolve(root, relative string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootAbs = filepath.Clean(rootAbs)

	rel := strings.TrimSpace(relative)
	// Reject absolute paths before we strip separators (Unix + Windows).
	if rel != "" && (filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") || (len(rel) > 1 && rel[1] == ':')) {
		return "", fmt.Errorf("absolute path not allowed")
	}
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		return rootAbs, nil
	}
	// Disallow parent segment tricks before join.
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("path escape not allowed")
	}

	joined := filepath.Clean(filepath.Join(rootAbs, rel))
	if joined != rootAbs && !strings.HasPrefix(joined, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("path outside projects root")
	}
	return joined, nil
}

// Rel returns path relative to root, using slash separators.
func Rel(root, abs string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootAbs = filepath.Clean(rootAbs)
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path outside projects root")
	}
	if rel == "." {
		return "", nil
	}
	return filepath.ToSlash(rel), nil
}

type DirEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"` // relative to projects root
	IsDir bool   `json:"isDir"`
}

func ListDir(root, relative string) ([]DirEntry, string, error) {
	abs, err := Resolve(root, relative)
	if err != nil {
		return nil, "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, "", err
	}
	if !st.IsDir() {
		return nil, "", fmt.Errorf("not a directory")
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, "", err
	}

	curRel, err := Rel(root, abs)
	if err != nil {
		return nil, "", err
	}

	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		// hide dotfiles
		if strings.HasPrefix(name, ".") {
			continue
		}
		var childRel string
		if curRel == "" {
			childRel = name
		} else {
			childRel = curRel + "/" + name
		}
		out = append(out, DirEntry{
			Name:  name,
			Path:  childRel,
			IsDir: e.IsDir(),
		})
	}
	return out, curRel, nil
}

// SafeName rejects path separators and parent refs in a single path segment.
func SafeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty name")
	}
	if name == "." || name == ".." {
		return "", fmt.Errorf("invalid name")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("name must not contain path separators")
	}
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid name")
	}
	return name, nil
}

// Mkdir creates a directory under root/parentRel/name.
func Mkdir(root, parentRel, name string) (string, error) {
	name, err := SafeName(name)
	if err != nil {
		return "", err
	}
	parentAbs, err := Resolve(root, parentRel)
	if err != nil {
		return "", err
	}
	target := filepath.Join(parentAbs, name)
	// re-check still under root
	if _, err := Rel(root, target); err != nil {
		return "", err
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		return "", err
	}
	return Rel(root, target)
}


// NameFromGitURL derives a folder name from a git remote URL.
func NameFromGitURL(url string) string {
	base := strings.TrimSpace(url)
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	// git@host:org/repo form may still have org/repo after last :
	if i := strings.LastIndex(base, ":"); i >= 0 && !strings.Contains(base, "/") {
		base = base[i+1:]
		if j := strings.LastIndex(base, "/"); j >= 0 {
			base = base[j+1:]
		}
	}
	return strings.TrimSuffix(base, ".git")
}
