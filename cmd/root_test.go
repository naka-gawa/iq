package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newVersionCommand(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		revision string
		want     string
	}{
		{
			name:     "Show version",
			version:  "1.0.0",
			revision: "abcde",
			want:     "iq version 1.0.0, revision abcde\n",
		},
		{
			name:     "Show version without revision",
			version:  "dev",
			revision: "",
			want:     "iq version dev, revision \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetVersionInfo(tt.version, tt.revision)

			cmd := newVersionCommand()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.Execute()

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestExecute(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name    string
		args    []string
		wantOut string
		wantErr bool
	}{
		{
			name:    "version command",
			args:    []string{"iq", "version"},
			wantOut: "iq version",
			wantErr: false,
		},
		{
			name:    "unknown command",
			args:    []string{"iq", "unknown"},
			wantOut: `Error: unknown command "unknown" for "iq"`,
			wantErr: true,
		},
		{
			name:    "help flag",
			args:    []string{"iq", "--help"},
			wantOut: "iq is a fast and flexible CLI tool for parsing INI files.",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			// Redirect stdout and stderr
			r, w, _ := os.Pipe()
			oldOut := os.Stdout
			oldErr := os.Stderr
			defer func() {
				os.Stdout = oldOut
				os.Stderr = oldErr
			}()
			os.Stdout = w
			os.Stderr = w

			err := Execute()

			w.Close()
			out, _ := io.ReadAll(r)
			output := string(out)

			assert.Equal(t, tt.wantErr, err != nil, "Execute() error = %v, wantErr %v", err, tt.wantErr)
			assert.Contains(t, output, tt.wantOut)
		})
	}
}
