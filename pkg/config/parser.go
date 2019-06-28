package config

import (
	"github.com/avinor/tau/pkg/hclcontext"
	gohcl2 "github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	hcl2parse "github.com/hashicorp/hcl2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

var (
	parser *hcl2parse.Parser
)

func init() {
	parser = hcl2parse.NewParser()
}

// Parse file content into interface val. It will not read the filename from disk
// so filename argument can be anything, but it will not try to load same file twice.
func Parse(content []byte, filename string, val interface{}) error {
	f, diags := parser.ParseHCL(content, filename)
	if diags.HasErrors() {
		return diags
	}

	if diags := gohcl2.DecodeBody(f.Body, hclcontext.Default, val); diags.HasErrors() {
		return diags
	}

	return nil
}

// ParseBody parses the hcl.Body into a value map.
func ParseBody(body hcl.Body) (map[string]cty.Value, error) {
	values := map[string]cty.Value{}
	diags := gohcl2.DecodeBody(body, hclcontext.Default, &values)

	if diags.HasErrors() {
		return nil, diags
	}

	return values, nil
}
