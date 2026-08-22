package main

import (
	"encoding/json/v2"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/pkg/lsp"
	"github.com/ProCode-Software/klar/pkg/lsp/rpc"
)

const integerIsInt32 = false

// https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/3.18/metaModel/metaModel.json
const MetaModelURL = "https://raw.githubusercontent.com/microsoft/language-server-protocol/gh-pages/_specifications/lsp/" + lsp.ProtocolVersion + "/metaModel/metaModel.json"

var (
	exclude = [...]string{}
	outDir  string
)

func main() {
	outDir = os.Args[1]
	mm := fetchMetaModel()
	types := makeSymbolMap(mm)
	for _, exc := range exclude {
		delete(types, exc)
	}
	categorizeTypes(mm, types)

	sortedSymbols := slices.Sorted(maps.Keys(types))
	w := writer{mm: mm, symbols: types, sortedSymbols: sortedSymbols}
	// Always write structs first
	w.writeStructs()
	for _, name := range sortedSymbols {
		entry := w.symbols[name]
		if entry.category != "" {
			w.currCategory = entry.category
		}
		switch decl := entry.decl.(type) {
		case *TypeAlias:
			w.writeTypeAlias(decl, w.getFile(entry.category))
		case *Enumeration:
			w.writeEnum(decl, w.getFile(entry.category))
		}
	}
	
	w.writeFiles()
}

// https://github.com/microsoft/language-server-protocol/blob/8b9fab8f0912b694c795d05c1d5e9d357bee0193/_specifications/lsp/3.18/metaModel/metaModel.ts
type MetaModel struct {
	// metaData is ignored
	Requests      []*Request      `json:"requests"`
	Notifications []*Notification `json:"notifications"`
	Structures    []*Structure    `json:"structures"`
	Enumerations  []*Enumeration  `json:"enumerations"`
	TypeAliases   []*TypeAlias    `json:"typeAliases"`
}

func fetchMetaModel() (mm *MetaModel) {
	res, err := http.Get(MetaModelURL)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	if err := json.UnmarshalRead(res.Body, &mm); err != nil {
		panic(err)
	}
	return
}

type symbol struct {
	decl     any
	category string
}

type symbolMap map[string]symbol

func makeSymbolMap(mm *MetaModel) symbolMap {
	// Requests and notifications aren't declared as types
	types := make(symbolMap)
	for _, t := range mm.Structures {
		types[t.Name] = symbol{decl: t}
	}
	for _, t := range mm.Enumerations {
		types[t.Name] = symbol{decl: t}
	}
	for _, t := range mm.TypeAliases {
		types[t.Name] = symbol{decl: t}
	}
	return types
}

func getCategory(method string) string {
	var scope string
	scopes := strings.Split(method, "/")
	switch l := len(scopes); l {
	case 0:
		scope = method
	case 1:
		scope = scopes[0]
	default:
		scopes = scopes[:l-1]
		if slices.Contains(scopes, "semanticTokens") {
			scope = "semanticTokens"
		} else {
			scope = scopes[len(scopes)-1]
		}
		if scopes[0] == "$" {
			return "dollar"
		}
	}
	switch scope {
	case "initialize", "shutdown", "initialized", "exit":
		return "lifecycle"
	default:
		// textDocument -> text_document for file names
		snakeCase := make([]rune, 0, len(scope)+2)
		for _, r := range scope {
			if unicode.IsUpper(r) {
				snakeCase = append(snakeCase, '_')
				r = unicode.ToLower(r)
			}
			snakeCase = append(snakeCase, r)
		}
		return string(snakeCase)
	}
}

func (sm symbolMap) setCategory(s, cat string) {
	sm[s] = symbol{decl: sm[s].decl, category: cat}
}

func categorizeTypes(mm *MetaModel, sm symbolMap) {
	type queueEntry struct {
		method string
		typ    *Type
	}
	var queue []queueEntry
	enqueueListUnion := func(method string, union *rpc.Union2[Type, []Type]) {
		switch {
		case union.IsNil():
		case union.Curr() == 0:
			queue = append(queue, queueEntry{method, new(union.A())})
		default:
			for _, param := range union.B() {
				queue = append(queue, queueEntry{method, &param})
			}
		}
	}
	for _, req := range mm.Requests {
		enqueueListUnion(req.Method, req.Params)
		queue = append(
			queue,
			queueEntry{req.Method, req.ErrorData},
			queueEntry{req.Method, req.RegistrationOptions},
			queueEntry{req.Method, req.Result},
		)
	}
	for _, notif := range mm.Notifications {
		enqueueListUnion(notif.Method, notif.Params)
		queue = append(queue, queueEntry{notif.Method, notif.RegistrationOptions})
	}
	for _, entry := range queue {
		typ := entry.typ
		if typ == nil || typ.Name == "" || sm[typ.Name].category != "" {
			continue
		}
		sm.setCategory(typ.Name, getCategory(entry.method))
	}
}
