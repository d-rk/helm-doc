package writer

import (
	"fmt"
	"github.com/random-dwi/helm-doc/generator"
	"github.com/random-dwi/helm-doc/output"
	"gopkg.in/yaml.v2"
	"io"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"regexp"
	"sort"
	"strings"
)

type MarkdownWriter struct {
	writer io.Writer
}

func NewMarkdownWriter(writer io.Writer) MarkdownWriter {
	return MarkdownWriter{writer: writer}
}

func (g MarkdownWriter) WriteChapter(title string, layer int) {

	for i := 0; i < layer; i++ {
		g.fprintf("#")
	}

	g.fprintf(" %s\n\n", title)
}

func (g MarkdownWriter) WriteMetaData(metaData *chart.Metadata, layer int) {

	for i := 0; i < layer; i++ {
		g.fprintf("#")
	}

	g.fprintf(" %s\n\n", metaData.Name)
	g.fprintf("- **Version:** %s\n\n", metaData.Version)
	g.fprintf("- **Description:** %s\n", metaData.Description)
	g.fprintf("\n")
}

func (g MarkdownWriter) WriteDocs(docs map[string]*generator.ConfigDoc) {

	g.fprintf("|%s|%s|%s|%s|\n", "KEY", "DESCRIPTION", "DEFAULT", "EXAMPLE")
	g.fprintf("|---|---|---|---|\n")

	var keysSorted []string

	for key := range docs {
		keysSorted = append(keysSorted, key)
	}
	sort.Strings(keysSorted)

	for _, key := range keysSorted {
		var configDoc = docs[key]
		row := []string{"`" + key + "`", sanitize(configDoc.Description), toMarkdown(configDoc.DefaultValue), toMarkdown(configDoc.ExampleValue)}
		g.fprintf("|%s|\n", strings.Join(row, "|"))
	}
}

func toMarkdown(object interface{}) string {
	if object == nil {
		//to avoid removal of table cell
		return " "
	}

	mapObject, isMap := object.(map[string]interface{})
	if isMap {
		serialized, err := yaml.Marshal(mapObject)
		if err != nil {
			output.Failf("unable to serialize object: %v", mapObject)
		}
		return sanitize(fmt.Sprintf("<code>%v</code>", string(serialized)))
	} else {
		return sanitize(fmt.Sprintf("<code>%v</code>", object))
	}
}

func sanitize(value string) string {
	newLineRegex := regexp.MustCompile(`\r?\n`)
	whiteSpaceRegex := regexp.MustCompile(`\s`)
	value = newLineRegex.ReplaceAllString(value, "<br>")
	value = whiteSpaceRegex.ReplaceAllString(value, "&nbsp;")
	if value == "" {
		//to avoid removal of table cell
		return " "
	} else {
		return value
	}
}

func (g MarkdownWriter) fprintf(format string, a ...interface{}) {

	var err error

	_, err = fmt.Fprintf(g.writer, format, a...)

	if err != nil {
		output.Failf("Failed to write output: %v", err)
	}
}
