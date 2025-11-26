package preproc

import (
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
)

type Block struct {
	content   string
	lineKinds map[int]map[Kind]bool
	fnames    []string
	vnames    []string
	id        string
}

func NewBlock(id string, content string) *Block {
	fnames := make([]string, 0)
	vnames := make([]string, 0)
	lineKinds := make(map[int]map[Kind]bool)
	return &Block{content: content, lineKinds: lineKinds, fnames: fnames, vnames: vnames, id: id}
}

type Kind string

const (
	KindFuncName = "func-name"
	KindFuncBody = "func-body"
	KindVarDecl  = "var-decl"
	KindOther    = "other"
)

type identCandidate struct {
	name string
	line int
}

func (b *Block) Parse() error {
	fset := token.NewFileSet()
	file := fset.AddFile("snippet.go", -1, len(b.content))

	var s scanner.Scanner
	s.Init(file, []byte(b.content), nil, scanner.ScanComments)

	mark := func(line int, k Kind) {
		if line == 0 {
			return
		}
		if b.lineKinds[line] == nil {
			b.lineKinds[line] = make(map[Kind]bool)
		}
		b.lineKinds[line][k] = true
	}

	var (
		braceDepth         int  // общая глубина фигурных скобок
		funcSignatureOpen  bool // после func ... до {
		insideFunc         bool // сейчас внутри тела функции
		funcLevel          int  = -1
		lastWasFuncKeyword bool // предыдущий токен был "func"

		inVarDecl         bool             // внутри var-объявления
		pendingCandidates []identCandidate // кандидаты для :=
	)

	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		position := fset.Position(pos)
		line := position.Line

		if insideFunc {
			mark(line, KindFuncBody)
		}

		switch tok {
		case token.FUNC:
			lastWasFuncKeyword = true
			funcSignatureOpen = true

		case token.IDENT:
			if lastWasFuncKeyword {
				mark(line, KindFuncName)
				b.fnames = append(b.fnames, lit)
				lastWasFuncKeyword = false
				break
			}

			if inVarDecl {
				mark(line, KindVarDecl)
				b.vnames = append(b.vnames, lit)
			} else {
				pendingCandidates = append(pendingCandidates, identCandidate{
					name: lit,
					line: line,
				})
			}

		case token.VAR:
			inVarDecl = true
			pendingCandidates = nil

		case token.DEFINE:
			for _, c := range pendingCandidates {
				mark(c.line, KindVarDecl)
				b.vnames = append(b.vnames, c.name)
			}
			pendingCandidates = nil

		case token.SEMICOLON:
			inVarDecl = false
			pendingCandidates = nil
			lastWasFuncKeyword = false

		case token.LBRACE:
			braceDepth++
			if funcSignatureOpen {
				insideFunc = true
				funcSignatureOpen = false
				funcLevel = braceDepth
			}

		case token.RBRACE:
			if braceDepth > 0 {
				braceDepth--
			}
			if insideFunc && braceDepth < funcLevel {
				insideFunc = false
				funcLevel = -1
			}

		default:
			// остальные токены пропустить
		}
	}

	lines := strings.Split(b.content, "\n")
	for i, text := range lines {
		lineNum := i + 1
		if strings.TrimSpace(text) == "" {
			continue
		}
		if len(b.lineKinds[lineNum]) == 0 {
			mark(lineNum, KindOther)
		}
	}
	return nil
}

func (b *Block) FormExportFunc() string {
	funcDefs := baseCopypaste
	fMapName := "_"
	vMapName := "_"
	if len(b.fnames) != 0 {
		fMapName = "varMap"
	}
	if len (b.vnames) != 0 {
		vMapName = "funcMap"
	}
	mains := fmt.Sprintf("func Export_block%s(%s *map[string]any, %s *map[string]any){\n", b.id, fMapName, vMapName)
	if len(b.fnames) != 0 {
		mains += "\tfuncsMap := *funcMap \n"
	}
	if len(b.vnames) != 0 {
		mains += "\tvarsMap := *varMap \n"
	}
	lines := strings.Split(b.content, "\n")

	for i, text := range lines {
		lineNum := i + 1
		kinds := []string{}
		for k := range b.lineKinds[lineNum] {
			switch k {
			case KindFuncName, KindFuncBody:
				funcDefs += lines[lineNum]
			case KindVarDecl, KindOther:
				mains += lines[lineNum]
			}
		}
		fmt.Printf("%2d: %-40q -> %v\n", lineNum, text, kinds)
	}
	if len(b.fnames) != 0 {
		for _, fname := range(b.fnames) {
			mains += fmt.Sprintf("\tfuncsMap[\"%s\"] = %s \n", fname, fname)
		}
	}
	if len(b.vnames) != 0 {
		for _, vname := range(b.vnames) {
			mains += fmt.Sprintf("\tfuncsMap[\"%s\"] = %s \n", vname, vname)
		}
	}

	return funcDefs + mains + "}\n"
}
