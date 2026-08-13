package glaslock

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/version"
)

// The glas.lock format is line-based. Each directive starts with a keyword
// on a line, followed by values that are parsed specifically by the
// directive's handler. 3 directives are considered top-level: `lockfile`,
// `klar`, and `package`. Other valid directives are handled under the
// `package` directive.
//
// We mainly went for a custom format over Klon for performance (Klon needs
// to tokenize and parse an AST before decoding) and to avoid reflection.
//
// See a sample of the lockfile format in lockfile_sample.txt.

func ParseFile(file string) (*Lockfile, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func Parse(r io.Reader) (*Lockfile, error) {
	var (
		s = bufio.NewScanner(r)
		l = &Lockfile{PackageMap: make(map[PkgHash]*Package)}
		i int
	)
	for s.Scan() {
	readDirective:
		line := trimLine(s.Text())
		if line == "" {
			continue
		}
		words := strings.Fields(line) // Will not be empty
		switch i {
		case 0:
			// Lockfile directive must be first
			if words[0] != "lockfile" {
				return nil, fmt.Errorf(
					"expected first directive to be lockfile, got %s", words[0],
				)
			}
			if len(words) < 2 {
				return nil, errors.New("expected lockfile version")
			}
			v, err := strconv.Atoi(strings.TrimPrefix(words[1], "v"))
			if err != nil {
				return nil, fmt.Errorf("couldn't parse lockfile version: %w", err)
			}
			l.Version = v
			if l.Version != LockfileVersion {
				return nil, ErrUnsupportedFormat
			}
		case 1:
			// Klar directive must be second
			if words[0] != "klar" {
				return nil, fmt.Errorf(
					"expected second directive to be klar, got %s", words[0],
				)
			}
			if len(words) < 2 {
				return nil, errors.New("expected Klar version")
			}
			v, err := version.Parse(strings.TrimSpace(line[len(words[0]):]))
			if err != nil {
				return nil, fmt.Errorf("couldn't parse Klar version: %w", err)
			}
			l.Klar = v
		default:
			// Package directive: any other directive is parsed by [parsePackageDirective]
			if words[0] != "package" {
				return nil, fmt.Errorf("expected package directive, got %s", words[0])
			}
			endedInPkg, err := parsePackageDirective(l, s, words)
			if err != nil {
				return nil, err
			}
			if endedInPkg {
				// parsePackageDirective stopped at the next 'package' directive. If we
				// don't use a goto, s.Scan() will be called and the 'package'
				// directive line will be skipped
				goto readDirective
			}
		}
		i++
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return l, nil
}

// line should exclude the "package" directive
func parsePackageDirective(
	l *Lockfile, s *bufio.Scanner, header []string,
) (endedInPkg bool, err error) {
	p := &Package{}
	// Package header (<name> from <source> [git hash])
	ph, err := parsePackageHeader(header[1:])
	if err != nil {
		return false, err
	}
	p.PackageHeader = *ph
	defer p.Init()

	// Package info
loop:
	for s.Scan() {
		line := trimLine(s.Text())
		if line == "" {
			continue
		}
		// There are always at least 2 words in a line
		var dir, rest string
		for dir = range strings.FieldsSeq(line) {
			rest = strings.TrimSpace(line[len(dir):])
			break
		}
		if rest == "" { // There were less than 2 fields
			return false, fmt.Errorf("expected parameter for directive %s", dir)
		}
		// Some directives can only be used with specific sources. Check to
		// ensure the current source is compatible with the directive.
		if srcs, ok := expectedSources[dir]; ok && !slices.Contains(srcs, ph.From) {
			return false, fmt.Errorf(
				"%s directive can only be used with the following sources: %v",
				dir, srcs,
			)
		}
		switch dir {
		case "for":
			switch {
			case rest == "dev":
				p.DevOnly = true
			case strings.HasPrefix(rest, "workspace"): // for workspace
				ws := strings.TrimSpace(rest[len("workspace"):])
				if ws == "" {
					// Missing workspace after `for workspace`
					return false, fmt.Errorf(
						"expected workspace name after 'for workspace' directive",
					)
				}
				p.For = append(p.For, ws)
			default:
				return false, fmt.Errorf("unknown parameter for directive for: %s", rest)
			}
		case "dependency":
			ph, err := parsePackageHeader(strings.Fields(rest))
			if err != nil {
				return false, err
			}
			p.Deps = append(p.Deps, ph)

		case "integrity":
			switch ph.From {
			case NPM:
				getInfo[*NPMInfo](p).Integrity = rest
			case Git:
				getInfo[*GitInfo](p).Integrity = rest
			}
		case "registry":
			getInfo[*NPMInfo](p).Registry = rest
		case "subpath":
			switch ph.From {
			case Git:
				getInfo[*GitInfo](p).Subpath = rest
			case Workspace:
				getInfo[*WorkspaceInfo](p).Dir = rest
			}
		case "path":
			getInfo[*LocalInfo](p).Path = rest
		case "url":
			getInfo[*GitInfo](p).URL = rest
		case "branch":
			info := getInfo[*GitInfo](p)
			info.RefType, info.Ref = glaspack.BranchRef, rest
		case "tag":
			info := getInfo[*GitInfo](p)
			info.RefType, info.Ref = glaspack.TagRef, rest
		case "package", "lockfile", "klar":
			endedInPkg = dir == "package"
			break loop // Top-level
		default:
			return false, fmt.Errorf("unknown package directive: %s", line)
		}
	}
	if err = s.Err(); err != nil {
		return false, err
	}
	l.Packages = append(l.Packages, p)
	l.PackageMap[p.PackageHeader.Hash] = p
	return endedInPkg, nil
}

func getInfo[T PackageInfo](p *Package) T {
	if p.Info == nil {
		switch v := new(T); any(*v).(type) {
		case *GitInfo:
			p.Info = &GitInfo{}
		case *LocalInfo:
			p.Info = &LocalInfo{}
		case *NPMInfo:
			p.Info = &NPMInfo{}
		case *WorkspaceInfo:
			p.Info = &WorkspaceInfo{}
		default:
			panic(fmt.Sprintf("unhandled PackageInfo: %T", v))
		}
	}
	return p.Info.(T)
}

// Contains only directives specific to sources
var expectedSources = map[string][]PackageSource{
	"registry":  {NPM},
	"integrity": {NPM, Git},
	"subpath":   {Git, Workspace},
	"url":       {Git},
	"branch":    {Git},
	"tag":       {Git},
	"path":      {Local},
}

func parsePackageHeader(parts []string) (p *PackageHeader, err error) {
	// name + version* + 'from' + source
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid package header: %v", parts)
	}
	p = &PackageHeader{Name: parts[0]}

	// Version can contain spaces (for the build type)
	fromI := slices.Index(parts[1:], "from")
	if fromI < 0 {
		return nil, fmt.Errorf("package header must contain 'from'")
	}
	fromI += 1 // parts was sliced when passed to slices.Index
	ver := strings.Join(parts[1:fromI], " ")
	if p.Version, err = version.Parse(ver); err != nil {
		return nil, fmt.Errorf("invalid version: %w", err)
	}

	if len(parts) < fromI+2 {
		return nil, fmt.Errorf("package header must contain a source after 'from'")
	}
	switch src := parts[fromI+1]; src {
	case "git":
		// There must be a commit hash after the git source
		if len(parts) < fromI+3 {
			return nil, fmt.Errorf(
				"package header must contain a commit hash after 'from git'",
			)
		}
		p.From = Git
		p.GitCommit = parts[fromI+2]
	case "npm":
		p.From = NPM
	case "workspace":
		p.From = Workspace
	case "local":
		p.From = Local
	default:
		return nil, fmt.Errorf("invalid source: %s", src)
	}
	p.GenerateHash()
	return p, nil
}

func (ph *PackageHeader) GenerateHash() PkgHash {
	h := fnv.New64a()
	h.Write([]byte(ph.Name))
	h.Write([]byte(ph.Version.Normalize().String()))
	h.Write([]byte(ph.From.String()))
	if ph.From == Git {
		h.Write([]byte(ph.GitCommit))
	}
	ph.Hash = PkgHash(h.Sum64())
	return ph.Hash
}

// trimLine returns an empty string if the line is empty or contains only a comment.
func trimLine(s string) string {
	// Remove any line comment from the end of the line
	if hash := strings.IndexByte(s, '#'); hash >= 0 {
		s = s[:hash]
	}
	return strings.TrimSpace(s)
}

// String returns a representation of the header in the same format used
// after the 'package' directive in the lockfile.
func (h *PackageHeader) String() string {
	var commit string
	if h.GitCommit != "" {
		commit = " " + h.GitCommit
	}
	return fmt.Sprintf("%s %s from %s%s", h.Name, h.Version, h.From, commit)
}

func (s PackageSource) String() string {
	srcs := [...]string{
		Git:       "git",
		NPM:       "npm",
		Workspace: "workspace",
		Local:     "local",
	}
	if s < 0 || int(s) >= len(srcs) {
		panic(fmt.Sprintf("invalid PackageSource: %d", s))
	}
	return srcs[s]
}

// WriteTo writes a lockfile in 'glas.lock' format to the given writer.
func (l *Lockfile) WriteTo(w io.Writer) (int64, error) {
	const indentSize = 4
	indent := char.Repeat(' ', indentSize)
	buf := &bytes.Buffer{}

	fmt.Fprintf(w, "lockfile v%d\nklar %s\n", l.Version, l.Klar)
	for _, pkg := range l.Packages {
		fmt.Fprintf(buf, "\npackage %s\n", pkg.PackageHeader.String())
		if pkg.DevOnly {
			buf.Write(indent)
			buf.WriteString("for dev\n")
		}
		for _, ws := range pkg.For {
			fmt.Fprintf(buf, "%*s %s\n", indentSize, "for", ws)
		}
		writeInfo(pkg.Info, buf, indent)
		for _, dep := range pkg.Deps {
			fmt.Fprintf(buf, "%sdependency %s\n", indent, dep.String())
		}
	}
	return buf.WriteTo(w)
}

const LockfileDisclaimer = `# Generated by Glas. Do not edit.`

func (l *Lockfile) WriteWithDisclaimerTo(w io.Writer) (int64, error) {
	if n, err := fmt.Fprintln(w, LockfileDisclaimer); err != nil {
		return int64(n), err
	}
	return l.WriteTo(w)
}

func writeInfo(info PackageInfo, buf *bytes.Buffer, indent []byte) {
	switch info := info.(type) {
	case *GitInfo:
		fmt.Fprintf(buf, "%surl %s\n", indent, info.URL)
		switch info.RefType {
		case glaspack.BranchRef:
			fmt.Fprintf(buf, "%sbranch %s\n", indent, info.Ref)
		case glaspack.TagRef:
			fmt.Fprintf(buf, "%stag %s\n", indent, info.Ref)
		case glaspack.CommitRef: // Nothing to do. Commit hash is already in header
		default:
			panic(fmt.Sprintf(
				"unhandled git RefType while writing lockfile: %d",
				info.RefType,
			))
		}
		if info.Subpath != "" {
			fmt.Fprintf(buf, "%ssubpath %s\n", indent, info.Subpath)
		}
		fmt.Fprintf(buf, "%sintegrity %s\n", indent, info.Integrity)
	case *NPMInfo:
		if strings.TrimSuffix(info.Registry, "/") != DefaultNPMRegistry {
			fmt.Fprintf(buf, "%sregistry %s\n", indent, info.Registry)
		}
		fmt.Fprintf(buf, "%sintegrity %s\n", indent, info.Integrity)
	case *WorkspaceInfo:
		fmt.Fprintf(buf, "%ssubpath %s\n", indent, info.Dir)
	case *LocalInfo:
		fmt.Fprintf(buf, "%spath %s\n", indent, info.Path)
	default:
		panic(fmt.Sprintf("unhandled PackageInfo type while writing lockfile: %T", info))
	}
}
