package parser

import (
	"bytes"
	"strings"

	bkparser "github.com/moby/buildkit/frontend/dockerfile/parser"

	"github.com/alfacentaur1/dockopt/internal/model"
)

package parser

import (
	"bytes"
	"strings"

	bkparser "github.com/moby/buildkit/frontend/dockerfile/parser"

	"github.com/alfacentaur1/dockopt/internal/model"
)

// Parse converts buildkit's node tree into our model
//e.g. COPY --from=build /app /app buildkit return Node
//Value:     "copy"           ← malými písmeny
//Flags:     ["--from=build"]
//StartLine: 9
//EndLine:   9
//Next ──→ Node{Value: "/app"} ──→ Node{Value: "/app"} ──→ nil
func Parse(src []byte) (*model.Dockerfile, error) {
	res, err := bkparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}

	df := &model.Dockerfile{
		Source: strings.Split(strings.TrimRight(string(src), "\n"), "\n"),
	}

	for _, n := range res.AST.Children {
		in := &model.Instruction{
			Cmd:       strings.ToUpper(n.Value),
			Flags:     n.Flags,
			JSON:      n.Attributes["json"],
			StartLine: n.StartLine,
			EndLine:   n.EndLine,
		}
		for a := n.Next; a != nil; a = a.Next {
			in.Args = append(in.Args, a.Value)
		}
		df.Instructions = append(df.Instructions, in)
	}
	return df, nil
}