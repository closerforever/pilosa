// Copyright 2017 Pilosa Corp.
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

package ctl

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pilosa/pilosa"
)

// ConfigCommand represents a command for printing a default config.
type ConfigCommand struct {
	*pilosa.CmdIO
}

// NewConfigCommand returns a new instance of ConfigCommand.
func NewConfigCommand(stdin io.Reader, stdout, stderr io.Writer) *ConfigCommand {
	return &ConfigCommand{
		CmdIO: pilosa.NewCmdIO(stdin, stdout, stderr),
	}
}

// Run prints out the default config.
func (cmd *ConfigCommand) Run(ctx context.Context) error {
	fmt.Fprintln(cmd.Stdout, strings.TrimSpace(`
data-dir = "~/.pilosa"
bind = "localhost:10101"

[cluster]
  poll-interval = "2m0s"
  replicas = 1
  hosts = [
    "localhost:10101",
  ]

[anti-entropy]
  interval = "10m0s"

[metric]
	service = "statsd"
	host = "127.0.0.1:8125"

[profile]
  cpu = ""
  cpu-time = "30s"

[plugins]
  path = ""
`)+"\n")
	return nil
}
