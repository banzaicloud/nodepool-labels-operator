// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errorhandler

import (
	"fmt"
	"sync"

	"emperror.dev/emperror"
	"emperror.dev/errors"

	"github.com/banzaicloud/nodepool-labels-operator/internal/platform/log"
)

var errorHandler emperror.Handler
var errorHandlerOnce sync.Once

// ErrorHandler returns an error handler.
func ErrorHandler(logger log.Logger) emperror.Handler {
	errorHandlerOnce.Do(func() {
		errorHandler = newErrorHandler(logger)
	})

	return errorHandler
}

func newErrorHandler(logger log.Logger) emperror.Handler {
	loggerHandler := NewHandler(logger)

	return emperror.HandlerFunc(func(err error) {
		var stackTracer interface{ StackTrace() errors.StackTrace }
		if errors.As(err, &stackTracer) {
			stackTrace := stackTracer.StackTrace()
			if len(stackTrace) > 0 {
				frame := stackTrace[0]

				err = errors.WithDetails(
					err,
					"func", fmt.Sprintf("%n", frame), // nolint: govet
					"file", fmt.Sprintf("%v", frame), // nolint: govet
				)
			}
		}

		loggerHandler.Handle(err)
	})
}
