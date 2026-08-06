package main

import (
	"os"
	"strings"

	"github.com/hexsans/hexmagnet/internal/gql/enums"
)

func main() {
	gqlParts := make([]string, 0, len(enums.Enums))
	for _, e := range enums.Enums {
		gqlParts = append(gqlParts, genGql(e.Name, e.Values))
	}

	f, fErr := os.Create("./graphql/schema/enums.graphqls")
	checkErr(fErr)

	_, wErr := f.WriteString(strings.Join(gqlParts, "\n"))
	checkErr(wErr)
}

func genGql(name string, values []string) string {
	var str strings.Builder

	_, _ = str.WriteString("enum " + name + " {\n")

	for _, value := range values {
		_, _ = str.WriteString("  " + value + "\n")
	}

	_, _ = str.WriteString("}\n")

	return str.String()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
