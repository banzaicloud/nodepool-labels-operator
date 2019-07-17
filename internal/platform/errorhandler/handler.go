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
	"emperror.dev/errors"
	"emperror.dev/errors/utils/keyval"

	"github.com/banzaicloud/nodepool-labels-operator/internal/platform/log"
)

type handler struct {
	logger log.Logger
}

// NewHandler returns a handler which logs errors using the platform logger
func NewHandler(logger log.Logger) *handler {
	return &handler{logger: logger}
}

// Handle logs an error
func (h *handler) Handle(err error) {
	var ctx map[string]interface{}

	// Extract context from the error and attach it to the log
	if details := errors.GetDetails(err); len(details) > 0 {
		ctx = keyval.ToMap(details)
	}

	logger := h.logger.WithFields(log.Fields(ctx))

	if errs := errors.GetErrors(err); len(errs) > 1 {
		for _, err := range errs {
			var ctx map[string]interface{}

			// Extract context from the error and attach it to the log
			if details := errors.GetDetails(err); len(details) > 0 {
				ctx = keyval.ToMap(details)
			}

			h.logger.WithFields(log.Fields(ctx)).Error(err.Error())
		}
	} else {
		logger.Error(err.Error())
	}
}
