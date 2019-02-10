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

package labeler

type Config struct {
	// ManagedLabelsAnnotation is name name of annotation which holds the managed labels
	ManagedLabelsAnnotation string `mapstructure:"managedLabelsAnnotation"`
	// ForbiddenLabelDomains holds the forbidden domain names, the labeler won't set matching labels
	ForbiddenLabelDomains []string `mapstructure:"forbiddenLabelDomains"`
}
