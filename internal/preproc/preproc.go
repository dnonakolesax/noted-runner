package preproc

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"log/slog"
	"strings"
)

type KernelTypes struct {
	vars  map[string]string
	funcs map[string]string
	types map[string]string
}

func NewKernelTypes() *KernelTypes {
	mv := make(map[string]string)
	mf := make(map[string]string)
	mt := make(map[string]string)
	return &KernelTypes{
		vars:  mv,
		funcs: mf,
		types: mt,
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
	content       string
	lineKinds     map[int]map[Kind]bool
	fnames        []string
	vnames        []string
	snames        []string
	id            string
	efaceRequired bool
	types         *KernelTypes
	reusedFuncs   []string
	reusedVars    []string
	reusedStructs []string
	structDefines map[string]string
}

func NewBlock(id string, content string, types *KernelTypes) *Block {
	fnames := make([]string, 0)
	vnames := make([]string, 0)
	snames := make([]string, 0)
	lineKinds := make(map[int]map[Kind]bool)

	return &Block{content: content, lineKinds: lineKinds, fnames: fnames, vnames: vnames, snames: snames, id: id, types: types}
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
		insideType         bool
		funcLevel          int  = -1
		typeLevel          int  = -1
		lastWasFuncKeyword bool // предыдущий токен был "func"
		lastWasTypeKeyword bool
		cTypeName          string

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

		case token.TYPE:
			lastWasTypeKeyword = true

		case token.IDENT:
			if lastWasFuncKeyword {
				mark(line, KindFuncName)
				b.fnames = append(b.fnames, lit)
				lastWasFuncKeyword = false
				b.types.funcs[lit] = "func"
				break
			}

			if lastWasTypeKeyword {
				b.snames = append(b.snames, lit)
				cTypeName = lit
				lastWasTypeKeyword = false
				insideType = true
				b.types.types[cTypeName] = "type"
				break
			}

			if funcSignatureOpen {
				if _, ok := knownTypes[lit]; ok {
					b.types.funcs[b.fnames[len(b.fnames)-1]] = b.types.funcs[b.fnames[len(b.fnames)-1]] + lit
				}
			}

			if insideType {
				b.types.types[cTypeName] = b.types.types[cTypeName] + lit
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
			if insideType {
				typeLevel = braceDepth
			}

		case token.RBRACE:
			if braceDepth > 0 {
				braceDepth--
			}
			if insideFunc && braceDepth < funcLevel {
				insideFunc = false
				funcLevel = -1
			}
			if insideType && braceDepth < typeLevel {
				insideType = false
				typeLevel = -1
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

func collectUsedNames(f *ast.File) map[string]bool {
	used := make(map[string]bool)

	// Рекурсивный обход AST
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// Обрабатываем выражения вида package.Identifier
			if ident, ok := node.X.(*ast.Ident); ok {
				used[ident.Name] = true
			}
		case *ast.Ident:
			// Игнорируем имена, которые являются пакетами в импортах
			// (они обрабатываются в SelectorExpr)
			// Но учитываем прямые использования без селектора
			if !isPackageName(f, node.Name) {
				used[node.Name] = true
			}
		}
		return true
	})

	return used
}

func isPackageName(f *ast.File, name string) bool {
	for _, imp := range f.Imports {
		if imp.Name != nil {
			if imp.Name.Name == name {
				return true
			}
		} else {
			// Извлекаем имя пакета из пути
			path := strings.Trim(imp.Path.Value, `"`)
			parts := strings.Split(path, "/")
			pkgName := parts[len(parts)-1]
			if pkgName == name {
				return true
			}
		}
	}
	return false
}

func filterImports(f *ast.File, usedNames map[string]bool) {
	var filteredImports []*ast.ImportSpec

	for _, imp := range f.Imports {
		// Не удаляем импорты с . и _
		if imp.Name != nil {
			if imp.Name.Name == "." || imp.Name.Name == "_" {
				filteredImports = append(filteredImports, imp)
				continue
			}
		}

		// Проверяем используется ли импорт
		if isImportUsed(imp, usedNames, f) {
			filteredImports = append(filteredImports, imp)
		}
	}

	// Обновляем список импортов в AST
	f.Imports = filteredImports

	// Обновляем декларации импортов
	var filteredDecls []ast.Decl
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			filteredDecls = append(filteredDecls, decl)
			continue
		}

		// Фильтруем спецификации в блоке импортов
		var filteredSpecs []ast.Spec
		for _, spec := range genDecl.Specs {
			impSpec := spec.(*ast.ImportSpec)
			// Проверяем есть ли этот импорт в отфильтрованном списке
			for _, filteredImp := range filteredImports {
				if impSpec == filteredImp {
					filteredSpecs = append(filteredSpecs, spec)
					break
				}
			}
		}

		// Если остались импорты - добавляем декларацию
		if len(filteredSpecs) > 0 {
			genDecl.Specs = filteredSpecs
			filteredDecls = append(filteredDecls, genDecl)
		}
	}

	f.Decls = filteredDecls
}

func isImportUsed(imp *ast.ImportSpec, usedNames map[string]bool, f *ast.File) bool {
	// Получаем имя, под которым импорт доступен в коде
	var importName string
	if imp.Name != nil {
		importName = imp.Name.Name
	} else {
		// Извлекаем имя пакета из пути
		path := strings.Trim(imp.Path.Value, `"`)
		parts := strings.Split(path, "/")
		importName = parts[len(parts)-1]
	}

	// Проверяем используется ли это имя
	return usedNames[importName]
}

func (b *Block) ClearImports(code string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		slog.Error("clear imports 378", "error", err.Error())
		return ""
	}

	// Собираем используемые имена
	usedNames := collectUsedNames(f)

	// Фильтруем импорты
	filterImports(f, usedNames)

	// Форматируем и возвращаем результат
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		slog.Error("clear imports 391", "error", err.Error())
		return ""
	}

	str := buf.String()

	slog.Info(str)
	return str
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
			ok := true
			for _, fname := range b.fnames {
				if fname == funcName {
					ok = false
				}
			}
			if ok {
				mains += fmt.Sprintf("\t%s := funcsMap[\"%s\"].(%s)\n", funcName, funcName, b.types.funcs[funcName])
			}
		}
	}

	if len(b.reusedVars) != 0 {
		for _, varName := range b.reusedVars {
			ok := true
			for _, vname := range b.vnames {
				if vname == varName {
					ok = false
				}
			}
			if ok {
				mains += fmt.Sprintf("\t%s := varsMap[\"%s\"].(%s)\n", varName, varName, b.types.vars[varName])
			}
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

	slog.Info(funcDefs + mains + "}\n")
	return b.ClearImports(funcDefs + mains + "}\n")
}
