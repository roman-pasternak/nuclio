/*
Copyright 2017 The Nuclio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dotnetcore

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/nuclio/nuclio/pkg/common"
	"github.com/nuclio/nuclio/pkg/processor/runtime"
	"github.com/nuclio/nuclio/pkg/processor/runtime/rpc"

	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
)

type dotnetcore struct {
	*rpc.AbstractRuntime
	Logger        logger.Logger
	configuration *runtime.Configuration
}

// NewRuntime returns a new .NET core runtime
func NewRuntime(parentLogger logger.Logger, configuration *runtime.Configuration) (runtime.Runtime, error) {
	newDotnetCoreRuntime := &dotnetcore{
		configuration: configuration,
		Logger:        parentLogger.GetChild("logger"),
	}

	var err error
	newDotnetCoreRuntime.AbstractRuntime, err = rpc.NewAbstractRuntime(newDotnetCoreRuntime.Logger,
		configuration,
		newDotnetCoreRuntime)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to create runtime")
	}

	return newDotnetCoreRuntime, nil
}

func (d *dotnetcore) RunWrapper(socketPath, controlSocketPath string) (*os.Process, error) {
	wrapperDLLPath := d.getWrapperDLLPath()
	d.Logger.DebugWith("Using dotnet core wrapper dll path", "path", wrapperDLLPath)
	if !common.IsFile(wrapperDLLPath) {
		return nil, fmt.Errorf("Can't find wrapper at %q", wrapperDLLPath)
	}

	handler := d.getHandler()
	d.Logger.DebugWith("Using dotnet core handler", "handler", handler)

	// pass global environment onto the process, and sprinkle in some added env vars
	env := os.Environ()
	env = append(env, d.GetEnvFromConfiguration()...)

	args := []string{
		"dotnet", wrapperDLLPath, socketPath,
	}

	d.Logger.DebugWith("Running wrapper", "command", strings.Join(args, " "))

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	return cmd.Process, cmd.Start()
}

func (d *dotnetcore) getHandler() string {
	return d.configuration.Spec.Handler
}

// TODO: Global processor configuration, where should this go?
func (d *dotnetcore) getWrapperDLLPath() string {
	scriptPath := os.Getenv("NUCLIO_DOTNETCORE_WRAPPER_PATH")
	if len(scriptPath) == 0 {
		return "/opt/nuclio/wrapper/wrapper.dll"
	}

	return scriptPath
}

func (d *dotnetcore) GetEventEncoder(writer io.Writer) rpc.EventEncoder {
	return rpc.NewEventJSONEncoder(d.Logger, writer)
}
