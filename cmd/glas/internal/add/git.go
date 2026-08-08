package add

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/pm/git"
	"github.com/ProCode-Software/klar/internal/util/manifest"
	"github.com/ProCode-Software/klar/internal/version"
)

var gitPath string

type gitPackage struct {
	url         string
	rawSpec     string
	specKind    glaspack.GitRefKind
	rev         string // Resolved commit (full hash)
	manifest    *glaspack.Manifest
	cloneDir    string // Directory that has '.git'. ~/.local/share/klar/packages/<url>.
	checkoutDir string // [gitPackage.cloneDir]/[gitPackage.rev]
	pkgDir      string // Directory in / == cloneDir where the actual package is
}

func (p *gitPackage) Source() PackageSource   { return GitSource }
func (p *gitPackage) ResolvedVersion() string { return p.manifest.Version.String() }
func (p *gitPackage) Name() string {
	if p.manifest != nil {
		return p.manifest.Name
	}
	return p.url
}

func (p *gitPackage) Info(ic *installContext) *pkgInfo {
	if gitPath == "" {
		gitPath = findGit()
	}
	// 1. Validate that any provided specifier/branch/commit exists
	// For commits, we have to clone first then rev-parse
	// Otherwise, use ls-remote before cloning
	if p.specKind == glaspack.BranchRef || p.specKind == glaspack.TagRef {
		p.validateSpec(ic)
	}
	// 2. Clone
	p.clone(ic)

	// 3. If the user provided a commit hash, now is the time to validate it.
	if p.specKind == glaspack.CommitRef {
		p.validateShortCommit(ic)
	}
	// 4. At this point, we have a full hash, now let's fetch and checkout.
	// The contents for this commit/version will be in a new directory.
	p.checkout(ic)
	// 5. Resolve the project/package. If url+subpath refers to a monorepo
	// root, ask the user which package to install.
	p.getPackage(ic)

	info := infoFromManifest(p.manifest)
	info.url = p.url
	return info
}

func (p *gitPackage) validateSpec(ic *installContext) {
	if !strings.Contains(p.url, "://") {
		p.url = "https://" + p.url
	}
	var b strings.Builder
	cmd := exec.Command(gitPath, "ls-remote", p.url)
	runCommand(cmd, &b, nil)

	var providedVer *version.Specifier
	if p.specKind == glaspack.TagRef {
		cleaned := strings.ReplaceAll(p.rawSpec, "-", " ")
		v, err := version.ParseSpecifier(cleaned)
		if err != nil {
			cli.Failuref(
				"Invalid version %s for dependency %s",
				klarerrs.Quote(cleaned), klarerrs.Quote(p.url),
			)
		}
		providedVer = &v
	}

	var latestTag *version.Version
	var latestTagRev string
	for line := range strings.Lines(b.String()) {
		if line == "" {
			continue
		}
		rev, path, found := strings.Cut(line, "\t")
		if !found || rev == "" || path == "" {
			panic(fmt.Sprintf(
				`unexpected output from 'git ls-remote': expected ${rev}\t${path} %q`,
				line,
			))
		}
		path = strings.TrimSpace(path)
		switch {
		case p.rawSpec == "":
			// Use the latest *tag*
			ver, isTag := strings.CutPrefix(path, "refs/tags/")
			if !isTag {
				continue
			}
			parsedVer, err := version.Parse(ver)
			if err != nil {
				continue
			}
			if latestTag == nil || version.Compare(*latestTag, parsedVer) < 0 {
				latestTagRev, latestTag = rev, &parsedVer
			}
		case p.specKind == glaspack.BranchRef:
			if path == "refs/heads/"+p.rawSpec {
				p.rev = rev
				return
			}
		case p.specKind == glaspack.TagRef:
			lineTag, ok := strings.CutPrefix(path, "refs/tags/")
			if !ok {
				continue
			}
			// TODO: Supply latest version to Specifier
			lineVer, err := version.Parse(lineTag)
			if err == nil && providedVer.Matches(lineVer) {
				p.rev = rev
				return
			}
		}
	}

	switch {
	case p.rawSpec == "" && latestTagRev != "":
		// No specific version provided, use latest tag
		p.rev = latestTagRev
	case p.rawSpec == "":
		cli.Failure("No valid version tags found in repository", p.url)
	case p.specKind == glaspack.TagRef:
		allTags := listGitEntriesWithPrefix(b.String(), "refs/tags/")
		// Show only valid version tags
		validTags := make([]string, 0, len(allTags))
		for _, tag := range allTags {
			if _, err := version.Parse(tag); err == nil {
				validTags = append(validTags, tag)
			}
		}
		slices.Sort(validTags)

		var allTagsMsg string
		if len(validTags) > 0 {
			allTagsMsg = "The valid tags are: " +
				ansi.JoinColor(validTags, ansi.CodeCyan, ", ")
		} else {
			allTagsMsg = "The repository has no valid tags"
		}
		cli.FailureDetailf(
			"Couldn't find a tag for version %s in the repository at %s\n\n",
			allTagsMsg,
			klarerrs.Quote(p.rawSpec), klarerrs.Quote(p.url),
		)
	case p.specKind == glaspack.BranchRef:
		allBranches := listGitEntriesWithPrefix(b.String(), "refs/heads/")
		var allBranchesMsg string
		if len(allBranches) > 0 {
			allBranchesMsg = "The available branches are: " +
				ansi.JoinColor(allBranches, ansi.CodeCyan, ", ")
		} else {
			allBranchesMsg = "The repository has no branches"
		}
		cli.FailureDetailf(
			"Couldn't find a branch named %s in the repository at %s\n\n",
			allBranchesMsg,
			klarerrs.Quote(p.rawSpec), klarerrs.Quote(p.url),
		)
	}
}

func listGitEntriesWithPrefix(s string, prefix string) (items []string) {
	for line := range strings.Lines(s) {
		_, path, _ := strings.Cut(line, "\t")
		if after, ok := strings.CutPrefix(path, prefix); ok {
			items = append(items, strings.TrimSuffix(after, "\n"))
		}
	}
	return items
}

func (p *gitPackage) clone(ic *installContext) {
	if err := module.LoadSystemDirs(); err != nil {
		cli.Failure("Failed to load system directories:", err)
	}
	if p.cloneDir == "" {
		p.cloneDir = git.InstallDirFor(ic.targetPkgs[0].DataDir(), p.url)
	}
	if _, err := os.Stat(p.cloneDir); err == nil {
		return // Already cloned in the past. All we need to do is checkout.
	} else if err := os.MkdirAll(p.cloneDir, 0o755); err != nil {
		cli.Failuref("Failed to create target directory %q:", err.Error(), p.cloneDir)
	}
	// Partial clone
	cmd := exec.Command(
		gitPath, "clone", "--bare", "--filter", "blob:none",
		p.url, filepath.Join(p.cloneDir, ".git"),
	)
	runCommand(cmd, nil, nil)
}

func (p *gitPackage) validateShortCommit(ic *installContext) {
	cmd := exec.Command(gitPath, "rev-parse", "--verify", "-q", p.rawSpec+"^{commit}")
	cmd.Dir = p.cloneDir
	rev, err := cmd.Output()
	switch err := err.(type) {
	case nil:
	case *exec.ExitError:
		if err.ExitCode() == 1 {
			cli.Failuref(
				"Can't find commit %s in repository %s",
				klarerrs.Quote(p.rawSpec), klarerrs.Quote(p.url),
			)
		}
		cli.FailureDetailf(
			"Command %q failed with exit code %d:\n\n", "%s",
			strings.Join(cmd.Args, " "), err.ExitCode(), bytes.TrimSpace(err.Stderr),
		)
	default:
		cli.FailureDetailf(
			"Failed to run command %q", err.Error(), strings.Join(cmd.Args, " "),
		)
	}
	p.rev = string(bytes.TrimSpace(rev))
}

func (p *gitPackage) checkout(ic *installContext) {
	checkoutDir := filepath.Join(p.cloneDir, p.rev)
	p.checkoutDir = checkoutDir
	if _, err := os.Stat(checkoutDir); err == nil {
		return // Version already installed
	}
	cmd := exec.Command(gitPath, "fetch", "--depth=1", "origin", p.rev)
	cmd.Dir = p.cloneDir
	runCommand(cmd, nil, nil)

	cmd = exec.Command(gitPath, "worktree", "add", checkoutDir, p.rev)
	cmd.Dir = p.cloneDir
	runCommand(cmd, nil, nil)
}

func (p *gitPackage) getPackage(ic *installContext) {
	dir := p.checkoutDir
	if subDir := ic.Flag("subdir").String(); subDir != "" {
		dir = filepath.Join(dir, subDir)
	}
	if _, err := os.Stat(dir); err != nil && errors.Is(err, fs.ErrNotExist) {
		cli.Failuref(
			"Can't find subdirectory %s in repository %s",
			klarerrs.Quote(ic.Flag("subdir").String()), klarerrs.Quote(p.url),
		)
	}
	pkgInfo := manifest.GetPackageInfo(dir)
	if manifest.IsMonorepoRoot(pkgInfo) {
		// TODO: Prompt the user to select a specific package. In the
		// future, we can allow them to select multiple.
	}
	p.pkgDir = pkgInfo.Dir
	p.manifest = pkgInfo.Manifest
}

func (p *gitPackage) Install(ic *installContext) {
	// We did all of the downloading just to prompt the user before
	// installing, so now we just add the package to the manifest.
}

func findGit() string {
	gitPath, err := exec.LookPath("git")
	switch {
	case err == nil:
	case errors.Is(err, exec.ErrNotFound):
		cli.Failure(
			"Couldn't find Git on the device",
			"Check if 'git' is in $PATH, or install at https://git-scm.com.",
		)
	default:
		cli.FailureDetailf("Failed to find Git on the device: ", err.Error())
	}
	return gitPath
}

func runCommand(cmd *exec.Cmd, stdout, stderr io.Writer) {
	var stderrB strings.Builder
	cmd.Stdout, cmd.Stderr = stdout, cmp.Or[io.Writer](stderr, &stderrB)
	switch err := cmd.Run().(type) {
	case nil:
	case *exec.ExitError:
		cli.FailureDetailf(
			"Command %q failed with exit code %d:\n\n", "%s",
			strings.Join(cmd.Args, " "), err.ExitCode(), stderrB.String(),
		)
	default:
		cli.FailureDetailf(
			"Failed to run command %q", err.Error(), strings.Join(cmd.Args, " "),
		)
	}
}
