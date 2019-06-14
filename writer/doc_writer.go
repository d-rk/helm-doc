package writer

import (
	"github.com/random-dwi/helm-doc/generator"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type DocumentationWriter interface {
	WriteChapter(title string, layer int)
	WriteMetaData(metaData *chart.Metadata, layer int)
	WriteDocs(docs map[string]*generator.ConfigDoc)
}
