/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package serve

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestServePkg(t *testing.T) {
	spec.Run(t, "RunHandler", testRunHandler, spec.Report(report.Terminal{}), spec.Sequential())

	//TODO https://github.com/pivotal/skenario/issues/83
	//var server *SkenarioServer
	//server = &SkenarioServer{IndexRoot: "."}
	//server.Serve()
	//
	//spec.Run(t, "Acceptance test", testAcceptance, spec.Report(report.Terminal{}), spec.Sequential())
	//
	//server.Shutdown()
}
