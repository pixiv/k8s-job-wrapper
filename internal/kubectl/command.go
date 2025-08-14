/*
Copyright 2025 pixiv Inc.

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

package kubectl

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"al.essio.dev/pkg/shellescape"
)

type Runner interface {
	// kubectl コマンドを実行する
	Run(ctx context.Context, arg ...string) (string, error)
}

var _ Runner = &Command{}

// kubectl バイナリのパスから [Command] を作る
func NewCommand(executable string) *Command {
	return &Command{
		executable: executable,
	}
}

type Command struct {
	executable string
}

func (c Command) Run(ctx context.Context, arg ...string) (string, error) {
	escapedArgs := make([]string, len(arg))
	for i, x := range arg {
		escapedArgs[i] = shellescape.Quote(x)
	}
	escapedExecutable := shellescape.Quote(c.executable)

	cmd := exec.CommandContext(ctx, escapedExecutable, escapedArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to open stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to open stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	out, err := io.ReadAll(stdout)
	if err != nil {
		_ = cmd.Wait()
		return "", fmt.Errorf("failed to read stdout: %w", err)
	}
	errOut, err := io.ReadAll(stderr)
	if err != nil {
		_ = cmd.Wait()
		return "", fmt.Errorf("failed to read stderr: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("%w: stderr=%s", err, errOut)
	}

	return string(out), nil
}
