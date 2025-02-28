/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package transformer

import (
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

// ReadMeGenerator implements Transformer interface
type ReadMeGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init initializes the translator
func (t *ReadMeGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the config of the transformer
func (t *ReadMeGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect executes detect in directories respecting the m2kignore
func (t *ReadMeGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *ReadMeGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:    transformertypes.TemplatePathMappingType,
		SrcPath: filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
	})
	return pathMappings, nil, nil
}
