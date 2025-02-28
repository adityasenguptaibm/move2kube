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

package dockerfilegenerator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"golang.org/x/mod/semver"
)

const (
	defaultNodeVersion = "12"
	packageJSONFile    = "package.json"
)

// NodejsDockerfileGenerator implements the Transformer interface
type NodejsDockerfileGenerator struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	NodejsConfig *NodejsDockerfileYamlConfig
}

// NodejsTemplateConfig implements Nodejs config interface
type NodejsTemplateConfig struct {
	Port        int32
	Build       bool
	NodeVersion string
}

// NodejsDockerfileYamlConfig represents the configuration of the Nodejs dockerfile
type NodejsDockerfileYamlConfig struct {
	DefaultNodejsVersion string `yaml:"defaultNodejsVersion"`
}

// PackageJSON represents NodeJS package.json
type PackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Keywords     []string          `json:"keywords"`
	Homepage     string            `json:"homepage"`
	License      string            `json:"license"`
	Files        []string          `json:"files"`
	Main         string            `json:"main"`
	Scripts      map[string]string `json:"scripts"`
	Os           []string          `json:"os"`
	CPU          []string          `json:"cpu"`
	Private      bool              `json:"private"`
	Engines      map[string]string `json:"engines"`
	Dependencies map[string]string `json:"dependencies"`
}

// Init Initializes the transformer
func (t *NodejsDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.NodejsConfig = &NodejsDockerfileYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.NodejsConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.NodejsConfig, err)
		return err
	}
	if t.NodejsConfig.DefaultNodejsVersion == "" {
		t.NodejsConfig.DefaultNodejsVersion = defaultNodeVersion
	}
	return nil
}

// GetConfig returns the transformer config
func (t *NodejsDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *NodejsDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	var packageJSON PackageJSON
	if err := common.ReadJSON(filepath.Join(dir, packageJSONFile), &packageJSON); err != nil {
		logrus.Debugf("unable to read the package.json file: %s", err)
		return nil, nil
	}
	if packageJSON.Name == "" {
		err = fmt.Errorf("unable to get name of nodejs service at %s. Ignoring", dir)
		return nil, err
	}
	services = map[string][]transformertypes.Artifact{
		packageJSON.Name: {{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceDirPathType: {dir},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *NodejsDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ServiceDirPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ServiceDirPathType][0], err)
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		ir := irtypes.IR{}
		irPresent := true
		err = a.GetConfig(irtypes.IRConfigType, &ir)
		if err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		build := false
		nodeVersion := t.NodejsConfig.DefaultNodejsVersion
		var packageJSON PackageJSON
		if err := common.ReadJSON(filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], packageJSONFile), &packageJSON); err != nil {
			logrus.Debugf("unable to read the package.json file: %s", err)
		} else {
			if _, ok := packageJSON.Scripts["build"]; ok {
				build = true
			}
			if node, ok := packageJSON.Engines["node"]; ok {
				if !strings.HasPrefix(node, "v") {
					node = "v" + node
				}
				nodeVersion = strings.TrimPrefix(semver.Major(node), "v")
			}
		}
		ports := ir.GetAllServicePorts()
		if len(ports) == 0 {
			envMap, err := godotenv.Read(filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], ".env"))
			if err == nil {
				portString, ok := envMap["PORT"]
				if ok {
					port, err := cast.ToInt32E(portString)
					if err == nil {
						ports = []int32{port}
					}
				}
			} else {
				logrus.Debugf("Could not parse the .env file, %s", err)
			}
		}
		port := commonqa.GetPortForService(ports, a.Name)
		var nodejsConfig NodejsTemplateConfig
		nodejsConfig.Build = build
		nodejsConfig.Port = port
		nodejsConfig.NodeVersion = nodeVersion
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: nodejsConfig,
		})
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
		p := transformertypes.Artifact{
			Name:  sImageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: sImageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:  sConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: a.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: sImageName,
				artifacts.ServiceConfigType:   sConfig,
			},
		}
		if irPresent {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
