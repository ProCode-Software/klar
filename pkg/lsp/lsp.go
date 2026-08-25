package lsp

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

const ProtocolVersion = "3.18"

// URIs. See:
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#uri
type (
	DocumentURI string
	URI         string
)

func (uri *DocumentURI) UnmarshalText(text []byte) (err error) {
	*uri, err = ParseDocumentURI(string(text))
	return
}

// Code adapted from https://github.com/golang/tools/blob/master/gopls/internal/protocol/uri.go
func ParseDocumentURI(s string) (DocumentURI, error) {
	if s == "" {
		return "", nil
	}
	// Valid schemes:
	// 	- file://...
	//  - untitled:...
	scheme, after, ok := strings.Cut(s, ":")
	after = strings.TrimPrefix(after, "//")
	switch {
	case !ok:
		return "", errors.New("DocumentURI requires a scheme, got " + s)
	case scheme != "file" && scheme != "untitled":
		return "", errors.New(
			"DocumentURI scheme must be 'file' or 'untitled', got " + scheme,
		)

		// Gopls has this, but I don't know if it's still necessary.
		/* case !strings.HasPrefix(s, "file:///"):
		// VS Code sends URLs with only two slashes, which are invalid.
		s = "file:///" + s[len("file://"):] */
	}
	// Even though the input is a URI, it may not be in canonical form. VS Code
	// in particular over-escapes :, @, etc. Unescape and re-encode to canonicalize.
	path, err := url.PathUnescape(after)
	if err != nil {
		return "", err
	}
	// File URIs from Windows may have lowercase drive letters.
	// Since drive letters are guaranteed to be case insensitive,
	// we change them to uppercase to remain consistent.
	// For example, file:///c:/x/y/z becomes file:///C:/x/y/z.
	if isWindowsDriveURIPath(path) {
		path = path[:1] + strings.ToUpper(string(path[1])) + path[2:]
	}
	return DocumentURI((&url.URL{Scheme: scheme, Path: path}).String()), nil
}

// isWindowsDriveURIPath returns true if the file URI is of the format used by
// Windows URIs. The url.Parse package does not specially handle Windows paths,
// so we check if the URI path has a drive prefix (e.g. "/C:").
func isWindowsDriveURIPath(uri string) bool {
	return len(uri) >= 4 &&
		uri[0] == '/' && unicode.IsLetter(rune(uri[1])) && uri[2] == ':'
}
