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

package irpreprocessor

import (
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
)

// ingressPreprocessor optimizes the ingress options of the application
type ingressPreprocessor struct {
}

func (opt *ingressPreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	for sn, s := range ir.Services {
		tempService := ir.Services[sn]
		for pfi, pf := range s.ServiceToPodPortForwardings {
			if pf.ServicePort.Number == 0 {
				continue
			}
			key := common.ConfigServicesKey + common.Delim + `"` + sn + `"` + common.Delim + `"` + fmt.Sprintf("%d", pf.ServicePort.Number) + `"` + common.Delim + "urlpath"
			message := fmt.Sprintf("What kind of service/ingress to create for %s's %d port?", sn, pf.ServicePort.Number)
			hints := []string{"Enter :- to not create service for the port", "For Ingress path, leave out leading / to use first part as subdomain", "Add :N as suffix for NodePort service type", "Add :L for Load Balancer service type", "Add :C for ClusterIP service type"}
			exposedServiceRelPath := ""
			if pf.ServiceRelPath != "" {
				exposedServiceRelPath = pf.ServiceRelPath
			} else {
				exposedServiceRelPath = "/" + sn
			}
			exposedServiceRelPath = strings.TrimSpace(qaengine.FetchStringAnswer(key, message, hints, exposedServiceRelPath))
			pf.ServiceRelPath = exposedServiceRelPath
			tempService.ServiceToPodPortForwardings[pfi] = pf
		}
		ir.Services[sn] = tempService
	}
	return ir, nil
}
