package updater

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// isMSI returns true if the URL points to an MSI installer.
func isMSI(url string) bool {
	return strings.HasSuffix(strings.ToLower(url), ".msi")
}

// ApplyUpdate downloads the new version, replaces the current exe, and restarts.
// This only works on Windows. Returns an error if something goes wrong before
// the restart — if it succeeds, the process exits and never returns.
func ApplyUpdate(downloadURL string, quitFn func()) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("self-update only supported on Windows")
	}

	if downloadURL == "" {
		return fmt.Errorf("no download URL")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	slog.Info("self-update: downloading", "url", downloadURL)

	// Download to temp file
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "scbridge-update-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("save download: %w", err)
	}
	tmpFile.Close()

	// If it's a zip, extract the exe from it
	var newExePath string
	if strings.HasSuffix(downloadURL, ".zip") {
		extracted, err := extractExeFromZip(tmpPath, tmpDir)
		os.Remove(tmpPath) // clean up the zip
		if err != nil {
			return fmt.Errorf("extract zip: %w", err)
		}
		newExePath = extracted
	} else {
		newExePath = tmpPath
	}

	slog.Info("self-update: downloaded", "path", newExePath)

	var psScript string
	if isMSI(downloadURL) {
		// MSI install: msiexec handles UAC elevation via -Verb RunAs.
		// Wait for it to finish, clean up, then relaunch from the same path
		// (msiexec replaces the installed exe in place).
		psScript = fmt.Sprintf(
			`Start-Process msiexec -ArgumentList '/i','%s','/passive','/norestart' -Wait -Verb RunAs; `+
				`Remove-Item -Force '%s'; `+
				`ie4uinit.exe -show; `+
				`Start-Process '%s'`,
			newExePath, newExePath, currentExe,
		)
	} else {
		// Portable exe: wait for this process to exit (by PID) before copying,
		// avoiding the file-lock race that a fixed sleep can lose.
		psScript = fmt.Sprintf(
			`$p = Get-Process -Id %d -ErrorAction SilentlyContinue; `+
				`if ($p) { $p.WaitForExit(30000) }; `+
				`Copy-Item -Force '%s' '%s'; `+
				`Remove-Item -Force '%s'; `+
				`ie4uinit.exe -show; `+
				`Start-Process '%s'`,
			os.Getpid(), newExePath, currentExe, newExePath, currentExe,
		)
	}

	cmd := exec.Command("powershell.exe", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.Dir = tmpDir
	if err := cmd.Start(); err != nil {
		os.Remove(newExePath)
		return fmt.Errorf("launch update script: %w", err)
	}

	slog.Info("self-update: restarting")

	// Quit the app — the batch script takes over
	quitFn()
	return nil
}

// extractExeFromZip finds and extracts the .exe from a zip archive.
func extractExeFromZip(zipPath, destDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".exe") {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			outPath := filepath.Join(destDir, "scbridge-update-new.exe")
			out, err := os.Create(outPath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				os.Remove(outPath)
				return "", err
			}
			out.Close()
			return outPath, nil
		}
	}

	return "", fmt.Errorf("no .exe found in zip")
}
