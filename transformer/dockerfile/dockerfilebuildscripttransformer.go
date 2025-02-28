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

package dockerfile

import (
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// DockerfileImageBuildScript implements Transformer interface
type DockerfileImageBuildScript struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// DockerfileImageBuildScriptTemplateConfig represents template config used by ImagePush script
type DockerfileImageBuildScriptTemplateConfig struct {
	DockerfileName   string
	ImageName        string
	ContextUnix      string
	ContextWindows   string
	ContainerRuntime string
}

// Init Initializes the transformer
func (t *DockerfileImageBuildScript) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *DockerfileImageBuildScript) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *DockerfileImageBuildScript) DirectoryDetect(dir string) (namedServices map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *DockerfileImageBuildScript) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	dfs := []DockerfileImageBuildScriptTemplateConfig{}
	nartifacts := []transformertypes.Artifact{}
	processedImages := map[string]bool{}
	for _, a := range append(newArtifacts, alreadySeenArtifacts...) {
		if a.Type != artifacts.DockerfileArtifactType {
			continue
		}
		sImageName := artifacts.ImageName{}
		err := a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
		}
		if processedImages[sImageName.ImageName] {
			continue
		}
		processedImages[sImageName.ImageName] = true
		for _, path := range a.Paths[artifacts.DockerfilePathType] {
			relPath := ""
			dockerfileName := filepath.Base(path)
			if len(a.Paths[artifacts.DockerfileContextPathType]) > 0 {
				relPath = a.Paths[artifacts.DockerfileContextPathType][0]
				dfrelPath, err := filepath.Rel(relPath, path)
				if err != nil {
					logrus.Errorf("Unable to convert dockerfile path as a relative path : %s", err)
				} else {
					dockerfileName = dfrelPath
				}
			} else {
				relPath = filepath.Dir(path)
			}
			if common.IsParent(path, t.Env.GetEnvironmentSource()) {
				relPath, err = filepath.Rel(t.Env.GetEnvironmentSource(), filepath.Dir(path))
				if err != nil {
					logrus.Errorf("Unable to make path relative : %s", err)
					continue
				}
				dfs = append(dfs, DockerfileImageBuildScriptTemplateConfig{
					ImageName:        sImageName.ImageName,
					ContextUnix:      common.GetUnixPath(filepath.Join(common.DefaultSourceDir, relPath)),
					ContextWindows:   common.GetWindowsPath(filepath.Join(common.DefaultSourceDir, relPath)),
					DockerfileName:   dockerfileName,
					ContainerRuntime: commonqa.GetContainerRuntime(),
				})
			} else if common.IsParent(path, t.Env.GetEnvironmentOutput()) {
				relPath, err = filepath.Rel(t.Env.GetEnvironmentOutput(), filepath.Dir(path))
				if err != nil {
					logrus.Errorf("Unable to make path relative : %s", err)
					continue
				}
				dfs = append(dfs, DockerfileImageBuildScriptTemplateConfig{
					ImageName:        sImageName.ImageName,
					ContextUnix:      common.GetUnixPath(relPath),
					ContextWindows:   common.GetWindowsPath(relPath),
					DockerfileName:   dockerfileName,
					ContainerRuntime: commonqa.GetContainerRuntime(),
				})
			} else {
				dfs = append(dfs, DockerfileImageBuildScriptTemplateConfig{
					ImageName:        sImageName.ImageName,
					ContextUnix:      common.GetUnixPath(filepath.Join(common.DefaultSourceDir, relPath)),
					ContextWindows:   common.GetWindowsPath(filepath.Join(common.DefaultSourceDir, relPath)),
					DockerfileName:   dockerfileName,
					ContainerRuntime: commonqa.GetContainerRuntime(),
				})
			}
			nartifacts = append(nartifacts, transformertypes.Artifact{
				Name: t.Env.ProjectName,
				Type: artifacts.NewImagesArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.NewImagesConfigType: artifacts.NewImages{
						ImageNames: []string{sImageName.ImageName},
					},
				},
			})
		}
	}
	if len(dfs) == 0 {
		return nil, nil, nil
	}
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
		DestPath:       common.ScriptsDir,
		TemplateConfig: dfs,
	})
	nartifacts = append(nartifacts, transformertypes.Artifact{
		Name: string(artifacts.ContainerImageBuildScriptArtifactType),
		Type: artifacts.ContainerImageBuildScriptArtifactType,
		Paths: map[transformertypes.PathType][]string{artifacts.ContainerImageBuildShScriptPathType: {filepath.Join(common.ScriptsDir, "builddockerimages.sh")},
			artifacts.ContainerImageBuildShScriptContextPathType:  {"."},
			artifacts.ContainerImageBuildBatScriptPathType:        {filepath.Join(common.ScriptsDir, "builddockerimages.bat")},
			artifacts.ContainerImageBuildBatScriptContextPathType: {"."},
		},
	})
	return pathMappings, nartifacts, nil
}
