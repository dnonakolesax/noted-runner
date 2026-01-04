package preproc

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"log/slog"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
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

	mark := func(line int, k Kind) bool {
		if line == 0 {
			return false
		}
		if b.lineKinds[line] == nil {
			b.lineKinds[line] = make(map[Kind]bool)
		}
		_, ok := b.lineKinds[line][k]
		b.lineKinds[line][k] = true
		return !ok
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
				ok := mark(line, KindVarDecl)
				if ok {
					b.vnames = append(b.vnames, lit)
				}
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
				ok := mark(line, KindVarDecl)
				b.vnames = append(b.vnames, c.name)
				if ok {
					totalLine += lines[c.line-1]
				}
			}
			tp, err := b.parseLineType(totalLine)

			if err != nil {
				return err
			}

			if len(tp) != len(pendingCandidates) {
				return fmt.Errorf("%s", "variables decl and val don't match at line "+totalLine)
			}

			for idx, val := range pendingCandidates {
				name := val.name
				b.types.vars[name] = tp[idx]
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

func (b *Block) parseLineType(totalLine string) ([]string, error) {
	splitted := strings.Split(totalLine, ":=")
	if len(splitted) != 2 {
		return []string{}, fmt.Errorf("not valid statement")
	}
	val := strings.Split(splitted[1], ",")
	res := make([]string, 0)

	for _, value := range val {
		expr, err := parser.ParseExpr(value)

		if err != nil {
			return []string{}, err
		}

		switch v := expr.(type) {
		case *ast.BasicLit:
			res = append(res, strings.ToLower(v.Kind.String()))
		default:
			typ := ""
			typLen := 0
			for i := 0; i < len(value); i++ {
				if value[i] == ' ' {
					typLen++
					continue
				}
				if value[i] != '&' {
					break
				}
				typ += "*"
			}
			if value[len(value)-1] == '}' {
				for i := len(typ) + typLen; i < len(value); i++ {
					if value[i] == '{' {
						break
					}
					typ += string(value[i])
				}
			} else if value[len(value)-1] == ')' {
				fname := ""
				for i := len(typ) + typLen; i < len(value); i++ {
					if value[i] == '(' {
						break
					}
					fname += string(value[i])
				}
				fnameSplitted := strings.SplitN(fname, ".", 2)
				if len(fnameSplitted) == 2 {
					importPath := fnameSplitted[0]
					funcName := fnameSplitted[1]
					loadedType, err := loadFuncResultType(importPath, funcName)
					if err != nil {
						return []string{}, err
					}
					typ += loadedType
				} else {
					if b.types.funcs[fname] != "" {
						fmt.Println(b.types.funcs[fname])
						types := strings.SplitAfterN(b.types.funcs[fname], ")", 2)
						if len(types) != 2 {
							return []string{}, fmt.Errorf("can't resolve func result type for %s", fname)
						}
						fmt.Println(types)
						if strings.HasPrefix(types[1], "(") && strings.HasSuffix(types[1], ")") {
							types[1] = strings.TrimSuffix(strings.TrimPrefix(types[1], "("), ")")
						}

						typesArr := strings.Split(types[1], ",")

						if len(typesArr) == 0 {
							return []string{}, fmt.Errorf("can't resolve func result type for %s", fname)
						} else if len(typesArr) == 1 {
							typ += strings.TrimSpace(typesArr[0])
						} else {
							typ = ""
							for _, t := range typesArr {
								res = append(res, strings.TrimSpace(t))
							}
						}
					}
					//return []string{}, fmt.Errorf("can't resolve func result type for %s", fname)
				}
			}
			if typ != "" {
				res = append(res, typ)
			}
		}
	}
	return res, nil
}

func loadFuncResultType(importPath, funcName string) (string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return "", err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return "", fmt.Errorf("packages contain errors")
	}
	if len(pkgs) == 0 {
		return "", fmt.Errorf("package not found")
	}

	scope := pkgs[0].Types.Scope()
	obj := scope.Lookup(funcName)
	if obj == nil {
		return "", fmt.Errorf("func %s not found in %s", funcName, importPath)
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return "", fmt.Errorf("%s is not a function", funcName)
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return "", fmt.Errorf("not a function signature")
	}

	res := sig.Results()
	if res.Len() == 0 {
		return "", fmt.Errorf("function %s has no results", funcName)
	}

	// Берём тип первого результата
	return res.At(0).Type().String(), nil
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
			if slices.Contains(filteredImports, impSpec) {
				filteredSpecs = append(filteredSpecs, spec)
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
	//fmt.Println(code)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		slog.Error("clear imports", "error", err.Error())
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

	//slog.Info(str)
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
		fPlaced := make(map[string]struct{})
		for _, funcName := range b.reusedFuncs {
			ok := true
			for _, fname := range b.fnames {
				if fname == funcName {
					ok = false
				}
			}
			_, placed := fPlaced[funcName]
			if ok && !placed {
				mains += fmt.Sprintf("\t%s := funcsMap[\"%s\"].(%s)\n", funcName, funcName, b.types.funcs[funcName])
				fPlaced[funcName] = struct{}{}
			}
		}
	}

	if len(b.reusedVars) != 0 {
		vPlaced := make(map[string]struct{})
		for _, varName := range b.reusedVars {
			ok := true
			for _, vname := range b.vnames {
				if vname == varName {
					ok = false
				}
			}
			_, placed := vPlaced[varName]
			if ok && !placed {
				mains += fmt.Sprintf("\t%s := varsMap[\"%s\"].(%s)\n", varName, varName, b.types.vars[varName])
				vPlaced[varName] = struct{}{}
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
		fused := make(map[string]struct{})
		for _, fname := range b.fnames {
			if _, ok := fused[fname]; ok {
				continue
			}
			mains += fmt.Sprintf("\tfuncsMap[\"%s\"] = %s \n", fname, fname)
			fused[fname] = struct{}{}
		}
	}
	if len(b.vnames) != 0 {
		vused := make(map[string]struct{})
		for _, vname := range b.vnames {
			if _, ok := vused[vname]; ok {
				continue
			}
			mains += fmt.Sprintf("\tvarsMap[\"%s\"] = %s \n", vname, vname)
			vused[vname] = struct{}{}
		}
	}

	//slog.Info(funcDefs + mains + "}\n")
	return b.ClearImports(funcDefs + mains + "}\n")
}
