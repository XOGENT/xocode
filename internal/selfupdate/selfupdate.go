// Package selfupdate implements `xocode upgrade`: it resolves the latest
// GitHub release, downloads the matching archive, verifies its checksum, and
// atomically replaces the running binary. It mirrors scripts/install.sh so the
// asset naming stays in one mental model.
package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xogent/xocode/internal/version"
)

const (
	repo      = "xogent/xocode"
	binary    = "xocode"
	userAgent = "xocode-selfupdate"
)

// LatestVersion returns the newest release tag (e.g. "v1.2.0").
func LatestVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", errors.New("no tag_name in latest release")
	}
	return rel.TagName, nil
}

// Run performs the upgrade. When checkOnly is true it only reports status.
func Run(ctx context.Context, checkOnly bool, out io.Writer) error {
	current := version.Version
	latest, err := LatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("check latest: %w", err)
	}
	fmt.Fprintf(out, "current: %s\nlatest:  %s\n", current, latest)

	if current == latest {
		fmt.Fprintln(out, "xocode is up to date.")
		return nil
	}
	if checkOnly {
		fmt.Fprintf(out, "A newer version is available. Run `xocode upgrade` to install it.\n")
		return nil
	}

	platform := runtime.GOOS + "_" + runtime.GOARCH
	archive := fmt.Sprintf("%s_%s.tar.gz", binary, platform)
	base := fmt.Sprintf("https://github.com/%s/releases/download/%s", repo, latest)

	tmp, err := os.MkdirTemp("", "xocode-upgrade-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, archive)
	if err := download(ctx, base+"/"+archive, archivePath); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	sums := filepath.Join(tmp, "checksums.txt")
	if err := download(ctx, base+"/checksums.txt", sums); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	if err := verifyChecksum(archivePath, sums, archive); err != nil {
		return err
	}

	newBin := filepath.Join(tmp, binary)
	if err := extractBinary(archivePath, binary, newBin); err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	if err := replaceRunning(newBin); err != nil {
		return err
	}
	fmt.Fprintf(out, "Upgraded xocode to %s.\n", latest)
	return nil
}

func download(ctx context.Context, url, dst string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func verifyChecksum(archivePath, sumsPath, archiveName string) error {
	sums, err := os.ReadFile(sumsPath)
	if err != nil {
		return err
	}
	var want string
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == archiveName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum for %s", archiveName)
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch (want %s, got %s)", want, got)
	}
	return nil
}

func extractBinary(archivePath, name, dst string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s not found in archive", name)
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) != name {
			continue
		}
		w, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, tr); err != nil { //nolint:gosec // trusted release archive, size-bounded by download
			w.Close()
			return err
		}
		return w.Close()
	}
}

// replaceRunning atomically swaps the currently running executable with newBin.
func replaceRunning(newBin string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)

	staged := filepath.Join(dir, "."+binary+".new")
	if err := copyFile(newBin, staged, 0o755); err != nil {
		return fmt.Errorf("stage new binary (need write access to %s?): %w", dir, err)
	}

	// Rename the running binary aside first (handles Linux "text file busy"),
	// then move the new one into place.
	old := filepath.Join(dir, "."+binary+".old")
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		_ = os.Remove(staged)
		return err
	}
	if err := os.Rename(staged, exe); err != nil {
		_ = os.Rename(old, exe) // roll back
		return err
	}
	_ = os.Remove(old)
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
