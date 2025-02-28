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

package parameterizer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/transformer/kubernetes/parameterizer"
	"github.com/sirupsen/logrus"
)

func TestGettingAndParameterizingResources(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	relBaseDir := "testdata"
	baseDir, err := filepath.Abs(relBaseDir)
	if err != nil {
		t.Fatalf("Failed to make the base directory %s absolute path. Error: %q", relBaseDir, err)
	}

	parameterizersPath := filepath.Join(baseDir, "parameterizers")
	k8sResourcesPath := filepath.Join(baseDir, "k8s-resources")
	outputPath := t.TempDir()
	psp := parameterizer.ParameterizerConfigT{
		Helm:        "helm-chart",
		Kustomize:   "kustomize",
		OCTemplates: "openshift-templates",
	}
	ps, err := parameterizer.CollectParamsFromPath(parameterizersPath)
	if err != nil {
		t.Fatalf("Unable to collect parameterizers. Error: %q", err)
	}
	psl := []parameterizer.ParameterizerT{}
	for _, p := range ps {
		psl = append(psl, p...)
	}
	filesWritten, err := parameterizer.Parameterize(k8sResourcesPath, outputPath, psp, psl)
	if err != nil {
		t.Fatalf("Failed to apply all the parameterizations. Error: %q", err)
	}
	if len(filesWritten) != 26 {
		t.Fatalf("Expected %d files to be written. Actual: %d", 26, len(filesWritten))
	}
	wantDataDir := filepath.Join(baseDir, "want")
	for _, fileWritten := range filesWritten {
		relFilePath, err := filepath.Rel(outputPath, fileWritten)
		if err != nil {
			t.Fatalf("failed to make the file path %s relative to the output path %s . Error: %q", fileWritten, outputPath, err)
		}
		if !strings.HasPrefix(relFilePath, "helm-chart/") {
			continue
		}
		wantDataPath := filepath.Join(wantDataDir, relFilePath)
		wantBytes, err := os.ReadFile(wantDataPath)
		if err != nil {
			t.Fatalf("Failed to read the test data at path %s . Error: %q", wantDataPath, err)
		}
		actualBytes, err := os.ReadFile(fileWritten)
		if err != nil {
			t.Fatalf("Failed to read the output data at path %s . Error: %q", fileWritten, err)
		}
		if !cmp.Equal(string(actualBytes), string(wantBytes)) {
			t.Fatalf("The file %s is different from expected. Differences:\n%s", relFilePath, cmp.Diff(string(wantBytes), string(actualBytes)))
		}
	}
}
