package process

import (
	"github.com/nikiforov-soft/yasp/output"
	outputtransform "github.com/nikiforov-soft/yasp/output/transform"
)

type outputGroup struct {
	Output           output.Output
	OutputTransforms []outputtransform.Transform
}
