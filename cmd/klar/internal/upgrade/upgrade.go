// Package upgrade handles the Klar upgrade process via the `klar upgrade` command.
//
// Functionality is closely related to the [Klar install.sh script] and should be
// kept in sync with it.
//
// [Klar install.sh script]: https://github.com/ProCode-Software/klar/tree/main/install.sh
package upgrade

import (
	"archive/zip"
	"cmp"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"time"

	"github.com/ProCode-Software/klar/cmd/klar/internal/clean"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util"
	klarver "github.com/ProCode-Software/klar/internal/version"
)

// The URL to the GitHub API endpoint used for querying Klar releases.
const ReleasesURL = "https://api.github.com/repos/ProCode-Software/klar/releases"

const (
	unixInstallCmd    = "curl -fsSL https://raw.githubusercontent.com/ProCode-Software/klar/main/install.sh | bash"
	windowsInstallCmd = "irm https://raw.githubusercontent.com/ProCode-Software/klar/main/install.ps1 | iex"
)

// The regular expression for Klar release assets published to GitHub.
var BinaryAssetRegex = regexp.MustCompile(
	// Arch not provided for browser (Wasm) binary
	`^klar-(?:(?P<commit>[0-9a-f]+)|(?P<version>[0-9\._a-z]+))-(?P<os>[a-z]+)(?:-(?P<arch>[-\w_]+))?`,
)

var (
	// For development purposes
	dryRun              = os.Getenv("KLAR_DRY_RUN") == "1"
	displayManualUpdate = true
)

func Run(c *command.Runner) {
	var ok bool
	defer func() {
		// Hidden if there are no prebuilds for the current platform
		if ok || !displayManualUpdate {
			return
		}
		// See project README
		installCmd := unixInstallCmd
		if runtime.GOOS == "windows" {
			installCmd = windowsInstallCmd
		}
		// If there's an error, tell the user how to update manually
		fmt.Printf("\nPlease update Klar manually by running:\n\n    %s\n", installCmd)
	}()
	startTime := time.Now()

	rel := getLatestRelease()
	// 0. Check if this is newer than the working Klar version
	isNewer, latestVer := isNewer(rel)
	if !isNewer {
		ansi.TagPrintfln(
			"<** g!>Congrats!</> You're already on the latest version of Klar (<c>v%s</c>)\n\n"+
				// TODO: Uncomment once we have numbered versions, and also add link to docs
				// "Release notes: <m>https://github.com/ProCode-Software/klar/releases/tag/%[1]s</m>"
				"Release notes: <m>https://github.com/ProCode-Software/klar/releases</m>",
			cli.KlarVersion,
		)
		cli.Exit(0)
	}

	ansi.TagFprintf(
		os.Stderr,
		// TODO: Once we have numbered releases, switch "build" for "v"
		"<** m!>Upgrading Klar to build <c>%s</c></> - You're currently on <y>v%s</y>\r",
		latestVer, cli.KlarVersionAndCommit,
	)

	// 1. Create temp dir to download zips to
	var tempDir string
	if !dryRun {
		var err error
		if tempDir, err = os.MkdirTemp("", "klar-upgrade"); err != nil {
			cli.Failure("Couldn't create temporary directory for upgrade", err)
		}
		defer os.RemoveAll(tempDir) // The error doesn't really matter
	}

	// 2. Download binaries for the appropriate platform to temp dir
	binariesZip := downloadBinaries(rel, tempDir)
	// 3. Download updated standard library
	stdlibZip := downloadStdlib(rel, tempDir)

	// 4. Determine location of Klar binary and standard library
	// LoadSystemDirs result will be used by [clean.ClearCache]
	if err := module.LoadSystemDirs(); err != nil {
		cli.Failure("Failed to get standard library directory:", err)
	}
	stdDir := module.SystemDirs.Std
	exec, err := os.Executable()
	if err != nil {
		cli.Failure("Failed to get path of 'klar' binary:", err)
	}
	binDir := filepath.Dir(exec)

	// 5. Extract Klar/Glas binaries and standard library
	if !dryRun {
		// Windows doesn't let you delete or overwrite the running Klar
		// executable, so instead we rename the current one to klar.old
		// and extract the new version in its place. When the new klar
		// exec is run again, it will delete klar.old.
		//
		// On other platforms, don't delete the old executable before overwriting.
		// TODO: Does the current approach leave the user without a working
		// executable if writing fails?
		extractZip(binariesZip, binDir, runtime.GOOS == "windows")
		// Delete old standard library
		if err := os.RemoveAll(stdDir); err != nil {
			cli.Failure("Failed to delete the previous version of the standard library:", err)
		}
		// Stdlib is in 'stdlib.zip/std'
		extractZip(stdlibZip, filepath.Dir(stdDir), false)

		// 6. Clear old cache, as the cache format in the new version may have changed
		clean.ClearCache(true, true)
	}

	fmt.Print(ansi.ClearLine)
	ansi.TagPrintfln(
		upgradeMessage, latestVer, util.FormatDuration(time.Since(startTime)),
		rel.TagName,
	)
	ok = true
}

const upgradeMessage = `<** g!>Welcome to <c>Klar build %s</c>!</> Upgraded in <b>%s</b>

<y>Release notes:</> <m>https://github.com/ProCode-Software/klar/releases/tag/%s</m>
<r>Report bugs:</> <m>https://github.com/ProCode-Software/klar/issues</m>`

func getLatestRelease() *githubRelease {
	res, err := http.Get(ReleasesURL)
	if err != nil {
		cli.Failure("Couldn't get Klar releases from GitHub", err)
	}
	defer res.Body.Close()
	var releases []*githubRelease
	if err := json.UnmarshalRead(res.Body, &releases); err != nil {
		cli.Failure("Couldn't decode Klar release list", err)
	}
	if len(releases) == 0 {
		cli.Failure("The GitHub API responded with no releases (this shouldn't happen)")
	}
	for _, release := range releases {
		if len(release.Assets) > 0 {
			return release
		}
	}
	cli.Failure("No Klar releases with assets found from GitHub API")
	return nil
}

func isNewer(rel *githubRelease) (newer bool, versionOrCommit string) {
	var assetName string
	var groups map[string]string
	for _, a := range rel.Assets {
		if groups = regexGroups(BinaryAssetRegex, a.Name); groups != nil {
			assetName = a.Name
			break
		}
	}
	if groups == nil {
		cli.Failuref(
			"No Klar binaries found in GitHub release named %q (tag %s)",
			rel.Name, rel.TagName,
		)
	}
	// Groups defined in [BinaryAssetRegex]
	if version := groups["version"]; version != "" {
		ver, err := klarver.Parse(version)
		if err != nil {
			cli.Failuref(
				"Release binary named %q has invalid Klar version %q",
				assetName, version,
			)
		}
		return klarver.Compare(ver, cli.ParsedKlarVersion) >= 1, version
	}
	if commit := groups["commit"]; commit != "" {
		// For the Klar prebuild stage (before numbered releases), this is all we
		// will rely on.
		// TODO: After the first numbered release of Klar, remove the commit group
		// from the regex, and only use versions.
		return commit != cli.KlarCommit, commit
	}
	cli.Failuref("Asset named %q has no version or commit", assetName)
	return false, ""
}

// downloadBinaries downloads the zip archive containing the Klar and Glas
// executables for the given release and appropriate platform/architecture
// (based on the current). The zip archive is downloaded to dir/binaries.zip.
func downloadBinaries(rel *githubRelease, dir string) (zipPath string) {
	// Convert current GOOS to the format used in release assets
	// To be kept in sync with:
	// - https://github.com/ProCode-Software/klar/tree/main/install.sh (oses, arches)
	// - https://github.com/ProCode-Software/klar/tree/main/scripts/build.sh (OS_NAMES, ARCH_NAMES)
	goosToName := map[string]string{
		"darwin": "macos", "linux": "linux", "windows": "windows",
	}
	goarchToName := map[string]string{"amd64": "x86_64", "arm64": "arm64"}

	osName, archName := goosToName[runtime.GOOS], goarchToName[runtime.GOARCH]
	if osName == "" || archName == "" {
		displayManualUpdate = false
		cli.Failuref(
			"Sorry, we don't provide prebuilds for the %s/%s Go platform. Please %s.",
			runtime.GOOS, runtime.GOARCH,
			ansi.Hyperlink("build Klar from source", buildFromSourceDocs),
		)
	}

	i := slices.IndexFunc(rel.Assets, func(a *githubAsset) bool {
		groups := regexGroups(BinaryAssetRegex, a.Name)
		return groups != nil && groups["os"] == osName && groups["arch"] == archName
	})
	if i < 0 {
		displayManualUpdate = false
		// TODO: This shows the tag (prebuild-*) rather than the commit number. Once
		// we have numbered releases, this won't be an issue. But change "build" to "v"
		cli.Failuref("Klar build %s is out, but not for your platform yet", rel.TagName)
	}
	asset := rel.Assets[i]
	res, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		cli.Failuref("Failed to download asset: %v", err)
	}
	defer res.Body.Close()

	if dryRun {
		return ""
	}
	// Temporary file
	zipPath = filepath.Join(dir, "binaries.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		cli.Failure("Failed to create file to download binaries:", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, res.Body); err != nil {
		cli.Failure("Failed to download binaries:", err)
	}
	return zipPath
}

func downloadStdlib(rel *githubRelease, dir string) (stdlibPath string) {
	i := slices.IndexFunc(rel.Assets, func(a *githubAsset) bool {
		return a.Name == "stdlib.zip"
	})
	if i == -1 {
		cli.Failure("The new release doesn't provide a new standard library")
	}
	asset := rel.Assets[i]
	res, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		cli.Failuref("Failed to download asset: %v", err)
	}
	defer res.Body.Close()

	// Temporary file
	stdlibPath = filepath.Join(dir, "stdlib.zip")
	file, err := os.Create(stdlibPath)
	if err != nil {
		cli.Failure("Failed to create file to download standard library:", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, res.Body); err != nil {
		cli.Failure("Failed to download standard library:", err)
	}
	return stdlibPath
}

func extractZip(zipPath, dir string, shouldRenameOld bool) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		cli.Failuref("Failed to open '%s': %v", filepath.Base(zipPath), err)
	}
	defer r.Close()

	for _, f := range r.File {
		isDir := f.FileInfo().IsDir()
		target := filepath.Join(dir, f.Name) // file.Name may be a nested path

		// Rename old EXE to klar/glas.old. See note in [Run]
		if shouldRenameOld && !isDir {
			if _, err := os.Stat(target); err == nil {
				if err := os.Rename(target, target+".old"); err != nil {
					cli.Failuref(
						"Failed to rename old %s executable: %v",
						filepath.Base(target), err,
					)
				}
			}
		}

		if isDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				cli.FailureDetailf(
					"Failed to create new directory at %q ", "(extracted from %s): %v",
					target, filepath.Base(zipPath), err,
				)
			}
			continue
		}
		// Create the parent directory if needed
		// May not equal `dir` as file.Name may be a nested path
		parentDir := filepath.Dir(target)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			cli.FailureDetailf(
				"Failed to create new directory at %q ", "(extracted from %s): %v",
				parentDir, filepath.Base(zipPath), err,
			)
		}
		writeFileFromZip(f, zipPath, target, shouldRenameOld)
	}
}

func writeFileFromZip(f *zip.File, zipPath, target string, oldRenamed bool) {
	var reinstallMsg string
	if oldRenamed {
		reinstallMsg = `

Your Klar installation now is corrupt. Please reinstall by running:

    irm https://raw.githubusercontent.com/ProCode-Software/klar/main/install.ps1 | iex

We apologise for the inconvenience.`
	}
	zipReader, err := f.Open()
	if err != nil {
		displayManualUpdate = false
		cli.FailureDetailf(
			"Failed to open %s/%s for reading: %v", reinstallMsg,
			filepath.Base(zipPath), f.Name,
		)
	}
	defer zipReader.Close()
	outFile, err := os.Create(target)
	if err != nil {
		displayManualUpdate = false
		cli.FailureDetailf("Failed to create %s: %v", reinstallMsg, target, err)
	}

	_, writeErr := io.Copy(outFile, zipReader)
	defer func() {
		closeErr := outFile.Close()
		if writeErr != nil || closeErr != nil {
			displayManualUpdate = false
			cli.FailureDetailf(
				"Failed to write %s: %v", reinstallMsg, target, cmp.Or(writeErr, closeErr),
			)
		}
	}()
}

// A nil map means no matches.
func regexGroups(re *regexp.Regexp, s string) map[string]string {
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return nil
	}
	res := make(map[string]string, len(re.SubexpNames()))
	for i, name := range re.SubexpNames()[1:] {
		match := matches[1:][i]
		res[name] = match
	}
	return res
}

// See https://api.github.com/repos/ProCode-Software/klar/releases
type githubRelease struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	Draft       bool           `json:"draft"`
	PreRelease  bool           `json:"prerelease"`
	PublishedAt time.Time      `json:"published_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Assets      []*githubAsset `json:"assets"`
}

type githubAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	Name               string `json:"name"`
	Label              string `json:"label"`
	Digest             string `json:"digest"`
	ContentType        string `json:"content_type"`
	Size               int64  `json:"size"`
}

const LongDescription = `We frequently release updates to Klar to add new features, fix bugs, increase stability, and enhance security. 'klar upgrade' makes it easy to update your Klar installation to the latest version. Upgrading Klar doesn't delete your projects or installed packages, but cache from the previous version of Klar will be cleared.

Note for development versions of Klar: 'klar upgrade' replaces the working Klar version with a prebuild from GitHub. If you don't want this, rebuild the Klar repo from source by syncing your clone and building from source. See instructions in the ` +
	"\x1b]8;;" + buildFromSourceDocs + "\x1b\\contributing guide\x1b]8;;\x1b\\."

const buildFromSourceDocs = "https://github.com/ProCode-Software/klar/blob/main/CONTRIBUTING.md#development-guide"
