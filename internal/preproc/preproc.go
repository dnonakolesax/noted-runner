package preproc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
)

type KernelTypes struct {
	vars  map[string]string
	funcs map[string]string
	//types map[string]string
}

func NewKernelTypes() *KernelTypes {
	mv := make(map[string]string)
	mf := make(map[string]string)
	return &KernelTypes{
		vars:  mv,
		funcs: mf,
	}
}

var knownTypes map[string]any = map[string]any{
	"bool": struct{}{},

	"string": struct{}{},
	"byte":   struct{}{},
	"rune":   struct{}{},

	"int64":   struct{}{},
	"int32":   struct{}{},
	"int":     struct{}{},
	"float32": struct{}{},
	"float64": struct{}{},
}

type Block struct {
	content     string
	lineKinds   map[int]map[Kind]bool
	fnames      []string
	vnames      []string
	id          string
	types       *KernelTypes
	reusedFuncs []string
	reusedVars  []string
}

func NewBlock(id string, content string, types *KernelTypes) *Block {
	fnames := make([]string, 0)
	vnames := make([]string, 0)
	lineKinds := make(map[int]map[Kind]bool)

	return &Block{content: content, lineKinds: lineKinds, fnames: fnames, vnames: vnames, id: id, types: types}
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
	lines := strings.Split(b.content, "\n")

	b.reusedFuncs = make([]string, 0)
	b.reusedVars = make([]string, 0)
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
				b.types.funcs[lit] = "func"
				break
			}

			if funcSignatureOpen {
				if _, ok := knownTypes[lit]; ok {
					b.types.funcs[b.fnames[len(b.fnames)-1]] = b.types.funcs[b.fnames[len(b.fnames)-1]] + lit
				}
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

			if _, ok := b.types.vars[lit]; ok {
				b.reusedVars = append(b.reusedVars, lit)
			}

			if _, ok := b.types.funcs[lit]; ok {
				b.reusedFuncs = append(b.reusedFuncs, lit)
			}

		case token.VAR:
			inVarDecl = true
			pendingCandidates = nil

		case token.DEFINE:
			totalLine := ""
			for _, c := range pendingCandidates {
				mark(c.line, KindVarDecl)
				b.vnames = append(b.vnames, c.name)
				totalLine += lines[c.line-1]
			}
			tp, err := parseLineType(totalLine)

			if err != nil {
				return err
			}

			// TODO: implement multi-var support
			b.types.vars[pendingCandidates[0].name] = tp

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
		case token.LPAREN:
			if funcSignatureOpen {
				b.types.funcs[b.fnames[len(b.fnames)-1]] = b.types.funcs[b.fnames[len(b.fnames)-1]] + "("
			}
		case token.RPAREN:
			if funcSignatureOpen {
				b.types.funcs[b.fnames[len(b.fnames)-1]] = b.types.funcs[b.fnames[len(b.fnames)-1]] + ")"
			}
		case token.COMMA:
			if funcSignatureOpen {
				b.types.funcs[b.fnames[len(b.fnames)-1]] = b.types.funcs[b.fnames[len(b.fnames)-1]] + ","
			}

		default:
			// остальные токены пропустить
		}
	}

	for i, text := range lines {
		lineNum := i + 1
		if strings.TrimSpace(text) == "" {
			continue
		}
		if len(b.lineKinds[lineNum]) == 0 {
			mark(lineNum, KindOther)
		}
	}
	// fmt.Printf("\n-----funcs-----\n")
	// for k, v := range b.types.funcs {
	// 	fmt.Printf("%s %s \n", k, v)
	// }
	// fmt.Printf("-----/funcs-----\n")
	// fmt.Printf("\n-----vars-----\n")
	// for k, v := range b.types.vars {
	// 	fmt.Printf("%s %s \n", k, v)
	// }
	// fmt.Printf("-----/vars-----\n")
	return nil
}

func parseLineType(totalLine string) (string, error) {
	splitted := strings.Split(totalLine, ":=")
	if len(splitted) != 2 {
		return "", fmt.Errorf("not valid statement")
	}
	value := splitted[1]

	expr, err := parser.ParseExpr(value)

	if err != nil {
		return "", err
	}

	switch v := expr.(type) {
	case *ast.BasicLit:
		return strings.ToLower(v.Kind.String()), nil
	}
	return "", fmt.Errorf("unknown type")
}

func (b *Block) FormExportFunc(attempt string) string {
	funcDefs := baseCopypaste
	fMapName := "_"
	vMapName := "_"
	if len(b.fnames) != 0 || len(b.reusedFuncs) != 0 {
		fMapName = "funcMap"
	}
	if len(b.vnames) != 0 || len(b.reusedVars) != 0 {
		vMapName = "varMap"
	}
	bFname := strings.ReplaceAll(b.id, "-", "_")
	mains := fmt.Sprintf("func Export_block_%s_%s(%s *map[string]any, %s *map[string]any){\n", bFname, attempt, fMapName, vMapName)
	if len(b.fnames) != 0 || len(b.reusedFuncs) != 0 {
		mains += "\tfuncsMap := *funcMap \n"
	}
	if len(b.vnames) != 0 || len(b.reusedVars) != 0 {
		mains += "\tvarsMap := *varMap \n"
	}
	lines := strings.Split(b.content, "\n")

	if len(b.reusedFuncs) != 0 {
		for _, funcName := range b.reusedFuncs {
			mains += fmt.Sprintf("\t%s := funcsMap[\"%s\"].(%s)\n", funcName, funcName, b.types.funcs[funcName])
		}
	}

	if len(b.reusedVars) != 0 {
		for _, varName := range b.reusedVars {
			mains += fmt.Sprintf("\t%s := varsMap[\"%s\"].(%s)\n", varName, varName, b.types.vars[varName])
		}
	}

	for i, text := range lines {
		lineNum := i + 1
		for k := range b.lineKinds[lineNum] {
			switch k {
			case KindFuncName, KindFuncBody:
				funcDefs += text + "\n"
			case KindVarDecl, KindOther:
				mains += "\t" + text + "\n"
			}
		}
	}
	if len(b.fnames) != 0 {
		for _, fname := range b.fnames {
			mains += fmt.Sprintf("\tfuncsMap[\"%s\"] = %s \n", fname, fname)
		}
	}
	if len(b.vnames) != 0 {
		for _, vname := range b.vnames {
			mains += fmt.Sprintf("\tvarsMap[\"%s\"] = %s \n", vname, vname)
		}
	}

	return funcDefs + mains + "}\n"
}
