package lsp

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

const ProtocolVersion = "3.18"

const (
	LanguageKlar LanguageKind = "klar"
	LanguageKlon LanguageKind = "klon"
)

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
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	// Valid schemes supported by KlarLS:
	// 	- file://...
	//  - untitled:...
	switch u.Scheme {
	case "":
		return "", errors.New("DocumentURI requires a scheme, got " + s)
	case "file", "untitled":
	default:
		return "", errors.New(
			"DocumentURI scheme must be 'file' or 'untitled', got " + u.Scheme,
		)

		// Gopls has this, but I don't know if it's still necessary.
		/* case !strings.HasPrefix(s, "file:///"):
		// VS Code sends URLs with only two slashes, which are invalid.
		s = "file:///" + s[len("file://"):] */
	}
	// File URIs from Windows may have lowercase drive letters.
	// Since drive letters are guaranteed to be case insensitive,
	// we change them to uppercase to remain consistent.
	// For example, file:///c:/x/y/z becomes file:///C:/x/y/z.
	if isWindowsDriveURIPath(u.Path) && u.Scheme == "file" {
		u.Path = u.Path[:1] + strings.ToUpper(string(u.Path[1])) + u.Path[2:]
	}
	return DocumentURI(u.String()), nil
}

// isWindowsDriveURIPath returns true if the file URI is of the format used by
// Windows URIs. The url.Parse package does not specially handle Windows paths,
// so we check if the URI path has a drive prefix (e.g. "/C:").
func isWindowsDriveURIPath(uri string) bool {
	return len(uri) >= 4 &&
		uri[0] == '/' && unicode.IsLetter(rune(uri[1])) && uri[2] == ':'
}
