package monarch

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

type shapeField struct {
	name   string
	args   []string
	sel    shapeSelection
	hasSel bool
}

type shapeSelection struct {
	fields  []shapeField
	spreads []string
}

type shapeOperation struct {
	kind string
	name string
	sel  shapeSelection
}

type shapeDocument struct {
	path  string
	ops   []shapeOperation
	frags map[string]shapeSelection
}

type shapeParser struct {
	src string
	pos int
}

func parseShapeDocument(path, src string) (shapeDocument, error) {
	p := &shapeParser{src: src}
	doc := shapeDocument{path: path, frags: map[string]shapeSelection{}}
	for {
		p.skipIgnored()
		if p.eof() {
			break
		}
		keyword := p.parseName()
		switch keyword {
		case "query", "mutation":
			name := p.parseName()
			p.skipIgnored()
			if p.peek() == '(' {
				p.skipBalanced()
			}
			sel, err := p.parseSelection()
			if err != nil {
				return doc, fmt.Errorf("%s: %v", path, err)
			}
			doc.ops = append(doc.ops, shapeOperation{kind: keyword, name: name, sel: sel})
		case "fragment":
			name := p.parseName()
			if p.parseName() != "on" {
				return doc, fmt.Errorf("%s: malformed fragment %s", path, name)
			}
			p.parseName()
			sel, err := p.parseSelection()
			if err != nil {
				return doc, fmt.Errorf("%s: %v", path, err)
			}
			doc.frags[name] = sel
		default:
			return doc, fmt.Errorf("%s: unexpected %q", path, keyword)
		}
	}
	if len(doc.ops) == 0 {
		return doc, fmt.Errorf("%s: no operations", path)
	}
	return doc, nil
}

func (p *shapeParser) eof() bool {
	return p.pos >= len(p.src)
}

func (p *shapeParser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *shapeParser) skipIgnored() {
	for !p.eof() {
		c := p.src[p.pos]
		if c == '#' {
			for !p.eof() && p.src[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',' {
			p.pos++
			continue
		}
		break
	}
}

func (p *shapeParser) parseName() string {
	p.skipIgnored()
	start := p.pos
	for !p.eof() {
		c := p.src[p.pos]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= '0' && c <= '9' && start < p.pos {
			p.pos++
			continue
		}
		break
	}
	return p.src[start:p.pos]
}

func (p *shapeParser) skipBalanced() {
	open := p.src[p.pos]
	p.pos++
	depth := 1
	for !p.eof() && depth > 0 {
		c := p.src[p.pos]
		if c == '"' {
			p.skipString()
			continue
		}
		if c == open || (open == '(' && c == '(') || (open == '[' && c == '[') || (open == '{' && c == '{') {
			depth++
		}
		if (open == '(' && c == ')') || (open == '[' && c == ']') || (open == '{' && c == '}') {
			depth--
		}
		p.pos++
	}
}

func (p *shapeParser) skipString() {
	p.pos++
	for !p.eof() {
		c := p.src[p.pos]
		p.pos++
		if c == '\\' {
			p.pos++
			continue
		}
		if c == '"' {
			break
		}
	}
}

func (p *shapeParser) skipValue() {
	p.skipIgnored()
	switch p.peek() {
	case '$':
		p.pos++
		p.parseName()
	case '"':
		p.skipString()
	case '[', '{':
		p.skipBalanced()
	default:
		for !p.eof() {
			c := p.src[p.pos]
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '+' {
				p.pos++
				continue
			}
			break
		}
	}
}

func (p *shapeParser) skipDirectives() {
	for {
		p.skipIgnored()
		if p.peek() != '@' {
			break
		}
		p.pos++
		p.parseName()
		p.skipIgnored()
		if p.peek() == '(' {
			p.skipBalanced()
		}
	}
}

func (p *shapeParser) parseSelection() (shapeSelection, error) {
	var sel shapeSelection
	p.skipIgnored()
	if p.peek() != '{' {
		return sel, fmt.Errorf("expected selection set")
	}
	p.pos++
	for {
		p.skipIgnored()
		if p.eof() {
			return sel, fmt.Errorf("unterminated selection set")
		}
		if p.peek() == '}' {
			p.pos++
			return sel, nil
		}
		if p.src[p.pos] == '.' {
			if !strings.HasPrefix(p.src[p.pos:], "...") {
				return sel, fmt.Errorf("unexpected '.'")
			}
			p.pos += 3
			p.skipIgnored()
			if strings.HasPrefix(p.src[p.pos:], "on ") || strings.HasPrefix(p.src[p.pos:], "on\t") {
				return sel, fmt.Errorf("inline fragments unsupported")
			}
			sel.spreads = append(sel.spreads, p.parseName())
			p.skipDirectives()
			continue
		}
		name := p.parseName()
		var f shapeField
		f.name = name
		p.skipDirectives()
		p.skipIgnored()
		if p.peek() == '(' {
			p.pos++
			for {
				p.skipIgnored()
				if p.peek() == ')' {
					p.pos++
					break
				}
				arg := p.parseName()
				p.skipIgnored()
				if p.peek() != ':' {
					return sel, fmt.Errorf("expected ':' after argument %s", arg)
				}
				p.pos++
				p.skipValue()
				f.args = append(f.args, arg)
			}
			p.skipDirectives()
		}
		p.skipIgnored()
		if p.peek() == '{' {
			nested, err := p.parseSelection()
			if err != nil {
				return sel, err
			}
			f.sel = nested
			f.hasSel = true
		}
		sel.fields = append(sel.fields, f)
	}
}

type shapeServerField struct {
	args     map[string]bool
	required map[string]bool
	returns  string
}

type shapeSchema interface {
	rootFields(root string) (map[string]shapeServerField, error)
	typeFields(name string) (map[string]shapeServerField, bool, error)
}

func checkShapeDocument(doc shapeDocument, schema shapeSchema) []string {
	var issues []string
	for _, op := range doc.ops {
		root := "Query"
		if op.kind == "mutation" {
			root = "Mutation"
		}
		fields, err := schema.rootFields(root)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %s %s: %v", doc.path, op.kind, op.name, err))
			continue
		}
		checkShapeSelection(doc, op.name, op.name, op.sel, fields, schema, &issues)
	}
	return issues
}

func checkShapeSelection(doc shapeDocument, op, path string, sel shapeSelection, fields map[string]shapeServerField, schema shapeSchema, issues *[]string) {
	for _, name := range sel.spreads {
		frag, ok := doc.frags[name]
		if !ok {
			*issues = append(*issues, fmt.Sprintf("%s: %s: unknown fragment %s at %s", doc.path, op, name, path))
			continue
		}
		checkShapeSelection(doc, op, path, frag, fields, schema, issues)
	}
	for _, f := range sel.fields {
		if f.name == "__typename" {
			continue
		}
		fp := path + "." + f.name
		sf, ok := fields[f.name]
		if !ok {
			*issues = append(*issues, fmt.Sprintf("%s: %s: field %s missing from live schema", doc.path, op, fp))
			continue
		}
		passed := map[string]bool{}
		for _, a := range f.args {
			passed[a] = true
			if !sf.args[a] {
				*issues = append(*issues, fmt.Sprintf("%s: %s: argument %s not accepted by live schema", doc.path, op, fp+":"+a))
			}
		}
		for req := range sf.required {
			if !passed[req] {
				*issues = append(*issues, fmt.Sprintf("%s: %s: required argument %s missing: upstream shape changed", doc.path, op, fp+":"+req))
			}
		}
		if !f.hasSel {
			continue
		}
		if sf.returns == "" {
			*issues = append(*issues, fmt.Sprintf("%s: %s: field %s selects subfields of a scalar: upstream shape changed", doc.path, op, fp))
			continue
		}
		sub, composite, err := schema.typeFields(sf.returns)
		if err != nil {
			*issues = append(*issues, fmt.Sprintf("%s: %s: %v", doc.path, op, err))
			continue
		}
		if !composite {
			continue
		}
		checkShapeSelection(doc, op, fp, f.sel, sub, schema, issues)
	}
}

type shapeIntroType struct {
	Kind   string          `json:"kind"`
	Name   string          `json:"name"`
	OfType *shapeIntroType `json:"ofType"`
}

func namedShapeType(t shapeIntroType) string {
	for i := 0; i < 6; i++ {
		if t.Kind != "NON_NULL" && t.Kind != "LIST" {
			break
		}
		if t.OfType == nil {
			return ""
		}
		t = *t.OfType
	}
	if t.Kind == "NON_NULL" || t.Kind == "LIST" {
		return ""
	}
	return t.Name
}

type liveShapeSchema struct {
	ctx   context.Context
	do    func(context.Context, *graphql.Request, any) error
	roots map[string]map[string]shapeServerField
	types map[string]shapeTypeInfo
}

type shapeTypeInfo struct {
	fields    map[string]shapeServerField
	composite bool
}

func newLiveShapeSchema(ctx context.Context, do func(context.Context, *graphql.Request, any) error) *liveShapeSchema {
	return &liveShapeSchema{ctx: ctx, do: do, roots: map[string]map[string]shapeServerField{}, types: map[string]shapeTypeInfo{}}
}

func (s *liveShapeSchema) introspect(name string) (kind string, fields map[string]shapeServerField, err error) {
	var resp struct {
		Type *struct {
			Kind   string `json:"kind"`
			Fields []struct {
				Name string `json:"name"`
				Args []struct {
					Name         string         `json:"name"`
					DefaultValue *string        `json:"defaultValue"`
					Type         shapeIntroType `json:"type"`
				} `json:"args"`
				Type shapeIntroType `json:"type"`
			} `json:"fields"`
		} `json:"__type"`
	}
	err = s.do(s.ctx, &graphql.Request{
		OperationName: "IntrospectShapeType",
		Query:         "query IntrospectShapeType($name: String!) { __type(name: $name) { kind fields { name args { name defaultValue type { kind } } type { kind name ofType { kind name ofType { kind name ofType { kind name } } } } } } }",
		Variables:     map[string]any{"name": name},
	}, &resp)
	if err != nil {
		return "", nil, err
	}
	if resp.Type == nil {
		return "", nil, fmt.Errorf("type %s missing from live schema", name)
	}
	fields = make(map[string]shapeServerField, len(resp.Type.Fields))
	for _, f := range resp.Type.Fields {
		sf := shapeServerField{args: map[string]bool{}, required: map[string]bool{}, returns: namedShapeType(f.Type)}
		for _, a := range f.Args {
			sf.args[a.Name] = true
			if a.Type.Kind == "NON_NULL" && a.DefaultValue == nil {
				sf.required[a.Name] = true
			}
		}
		fields[f.Name] = sf
	}
	return resp.Type.Kind, fields, nil
}

func (s *liveShapeSchema) rootFields(root string) (map[string]shapeServerField, error) {
	if fields, ok := s.roots[root]; ok {
		return fields, nil
	}
	_, fields, err := s.introspect(root)
	if err != nil {
		return nil, err
	}
	s.roots[root] = fields
	return fields, nil
}

func (s *liveShapeSchema) typeFields(name string) (fields map[string]shapeServerField, composite bool, err error) {
	if info, ok := s.types[name]; ok {
		return info.fields, info.composite, nil
	}
	kind, fields, err := s.introspect(name)
	if err != nil {
		return nil, false, err
	}
	composite = kind == "OBJECT" || kind == "INTERFACE"
	s.types[name] = shapeTypeInfo{fields: fields, composite: composite}
	return fields, composite, nil
}

func (p *liveProbe) operationShapeDrift() {
	p.check("shape/OperationDrift", func() error {
		var docs []shapeDocument
		err := fs.WalkDir(queries.FS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".graphql") {
				return nil
			}
			data, err := queries.FS.ReadFile(path)
			if err != nil {
				return err
			}
			doc, err := parseShapeDocument(path, string(data))
			if err != nil {
				return err
			}
			docs = append(docs, doc)
			return nil
		})
		if err != nil {
			return err
		}
		schema := newLiveShapeSchema(p.ctx, p.svc.Client.Do)
		var issues []string
		for _, doc := range docs {
			issues = append(issues, checkShapeDocument(doc, schema)...)
		}
		if len(issues) > 0 {
			return fmt.Errorf("live schema drift in %d place(s):\n%s", len(issues), strings.Join(issues, "\n"))
		}
		return nil
	})
}

type stubShapeSchema struct {
	roots map[string]map[string]shapeServerField
	types map[string]shapeTypeInfo
}

func serverField(argNames, required []string, returns string) shapeServerField {
	sf := shapeServerField{args: map[string]bool{}, required: map[string]bool{}, returns: returns}
	for _, a := range argNames {
		sf.args[a] = true
	}
	for _, r := range required {
		sf.required[r] = true
	}
	return sf
}

func (s stubShapeSchema) rootFields(root string) (map[string]shapeServerField, error) {
	return s.roots[root], nil
}

func (s stubShapeSchema) typeFields(name string) (fields map[string]shapeServerField, composite bool, err error) {
	info, ok := s.types[name]
	if !ok {
		return nil, false, fmt.Errorf("type %s missing from live schema", name)
	}
	return info.fields, info.composite, nil
}

func TestParseShapeDocumentAllEmbedded(t *testing.T) {
	count := 0
	err := fs.WalkDir(queries.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".graphql") {
			return nil
		}
		data, err := queries.FS.ReadFile(path)
		if err != nil {
			return err
		}
		doc, err := parseShapeDocument(path, string(data))
		if err != nil {
			return err
		}
		count++
		var unresolved []string
		var walk func(sel shapeSelection)
		walk = func(sel shapeSelection) {
			for _, name := range sel.spreads {
				frag, ok := doc.frags[name]
				if !ok {
					unresolved = append(unresolved, name)
					continue
				}
				walk(frag)
			}
			for _, f := range sel.fields {
				if f.hasSel {
					walk(f.sel)
				}
			}
		}
		for _, op := range doc.ops {
			walk(op.sel)
		}
		if len(unresolved) > 0 {
			return fmt.Errorf("%s: unresolved fragments %v", path, unresolved)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parse embedded documents: %v", err)
	}
	if count == 0 {
		t.Fatal("no embedded documents parsed")
	}
}

func TestParseShapeDocumentSample(t *testing.T) {
	doc, err := parseShapeDocument("sample.graphql", `
mutation Web_CreateManualAccount($input: CreateManualAccountMutationInput!) {
  createManualAccount(input: $input) {
    account {
      id
      displayName
    }
    errors {
      ...PayloadErrorFields
    }
  }
}

fragment PayloadErrorFields on PayloadError {
  message
  code
}

query Grouped($filters: TransactionFilterInput) {
  aggregates(filters: $filters, groupBy: ["category"], fillEmptyValues: false) {
    summary {
      sum
    }
  }
}
`)
	if err != nil {
		t.Fatalf("parseShapeDocument() error = %v", err)
	}
	if len(doc.ops) != 2 {
		t.Fatalf("ops = %d, want 2", len(doc.ops))
	}
	mut := doc.ops[0]
	if mut.kind != "mutation" || mut.name != "Web_CreateManualAccount" {
		t.Fatalf("mutation = %s %s", mut.kind, mut.name)
	}
	if len(mut.sel.fields) != 1 || mut.sel.fields[0].name != "createManualAccount" {
		t.Fatalf("top fields = %+v", mut.sel.fields)
	}
	top := mut.sel.fields[0]
	if len(top.args) != 1 || top.args[0] != "input" {
		t.Fatalf("args = %v, want [input]", top.args)
	}
	if len(top.sel.spreads) != 0 {
		t.Fatalf("spreads = %v, want fragment nested under errors", top.sel.spreads)
	}
	q := doc.ops[1]
	agg := q.sel.fields[0]
	if len(agg.args) != 3 || agg.args[0] != "filters" || agg.args[1] != "groupBy" || agg.args[2] != "fillEmptyValues" {
		t.Fatalf("args = %v, want filters/groupBy/fillEmptyValues", agg.args)
	}
}

func driftStub() stubShapeSchema {
	account := map[string]shapeServerField{
		"id":             serverField(nil, nil, ""),
		"displayName":    serverField(nil, nil, ""),
		"displayBalance": serverField(nil, nil, ""),
	}
	return stubShapeSchema{
		roots: map[string]map[string]shapeServerField{
			"Mutation": {
				"createManualAccount":  serverField([]string{"input"}, []string{"input"}, "CreateManualAccountPayload"),
				"updateAccount":        serverField([]string{"input"}, []string{"input"}, "UpdateAccountPayload"),
				"forceRefreshAccounts": serverField([]string{"input"}, []string{"input"}, "ForceRefreshAccountsPayload"),
				"deleteAccount":        serverField([]string{"id"}, []string{"id"}, "DeleteAccountPayload"),
			},
		},
		types: map[string]shapeTypeInfo{
			"Account":                     {fields: account, composite: true},
			"CreateManualAccountPayload":  {fields: map[string]shapeServerField{"account": serverField(nil, nil, "Account"), "errors": serverField(nil, nil, "PayloadError")}, composite: true},
			"UpdateAccountPayload":        {fields: map[string]shapeServerField{"account": serverField(nil, nil, "Account"), "errors": serverField(nil, nil, "PayloadError")}, composite: true},
			"ForceRefreshAccountsPayload": {fields: map[string]shapeServerField{"success": serverField(nil, nil, ""), "errors": serverField(nil, nil, "PayloadError")}, composite: true},
			"DeleteAccountPayload":        {fields: map[string]shapeServerField{"deleted": serverField(nil, nil, ""), "errors": serverField(nil, nil, "PayloadError")}, composite: true},
			"PayloadError":                {fields: map[string]shapeServerField{"message": serverField(nil, nil, "")}, composite: true},
		},
	}
}

func checkDoc(t *testing.T, src string, schema stubShapeSchema) []string {
	t.Helper()
	doc, err := parseShapeDocument("test.graphql", src)
	if err != nil {
		t.Fatalf("parseShapeDocument() error = %v", err)
	}
	return checkShapeDocument(doc, schema)
}

func TestCheckShapeDocumentCurrentPasses(t *testing.T) {
	issues := checkDoc(t, `mutation Web_CreateManualAccount($input: CreateManualAccountMutationInput!) {
  createManualAccount(input: $input) {
    account { id displayName displayBalance }
    errors { message }
  }
}`, driftStub())
	if len(issues) > 0 {
		t.Fatalf("issues = %v, want none", issues)
	}
}

func TestCheckShapeDocumentPositionalArgsDrift(t *testing.T) {
	issues := checkDoc(t, `mutation Web_CreateManualAccount($name: String!, $type: String!, $balance: Float!) {
  createManualAccount(name: $name, type: $type, balance: $balance) {
    account { id displayName displayBalance }
  }
}`, driftStub())
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "createManualAccount:input") || !strings.Contains(joined, "required") {
		t.Fatalf("issues = %v, want required input drift", issues)
	}
	if !strings.Contains(joined, "createManualAccount:name") {
		t.Fatalf("issues = %v, want unaccepted name arg", issues)
	}
}

func TestCheckShapeDocumentRenamedPayloadField(t *testing.T) {
	issues := checkDoc(t, `mutation Common_DeleteAccount($id: ID!) {
  deleteAccount(id: $id) {
    ok
  }
}`, driftStub())
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "deleteAccount.ok") || !strings.Contains(joined, "missing") {
		t.Fatalf("issues = %v, want missing ok field", issues)
	}
}

func TestCheckShapeDocumentMissingField(t *testing.T) {
	issues := checkDoc(t, `mutation Common_DeleteAccount($id: ID!) {
  removeAccount(id: $id) {
    deleted
  }
}`, driftStub())
	if len(issues) == 0 || !strings.Contains(issues[0], "removeAccount") {
		t.Fatalf("issues = %v, want missing removeAccount", issues)
	}
}

func TestNamedShapeType(t *testing.T) {
	obj := shapeIntroType{Kind: "OBJECT", Name: "Account"}
	if got := namedShapeType(obj); got != "Account" {
		t.Fatalf("namedShapeType() = %q", got)
	}
	wrapped := shapeIntroType{Kind: "NON_NULL", OfType: &shapeIntroType{Kind: "LIST", OfType: &shapeIntroType{Kind: "NON_NULL", OfType: &obj}}}
	if got := namedShapeType(wrapped); got != "Account" {
		t.Fatalf("namedShapeType() = %q", got)
	}
	scalar := shapeIntroType{Kind: "SCALAR", Name: "Boolean"}
	if got := namedShapeType(scalar); got != "Boolean" {
		t.Fatalf("namedShapeType() = %q", got)
	}
}
