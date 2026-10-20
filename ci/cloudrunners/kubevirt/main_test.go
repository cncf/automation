package main

import "testing"

func TestExpandCommand(t *testing.T) {
	t.Setenv("ACTIONS_RUNNER_INPUT_JITCONFIG", "s3cr3t")

	tests := []struct {
		name          string
		cmd           string
		wantExpanded  string
		wantSensitive bool
	}{
		{
			name:          "jitconfig is substituted and flagged sensitive",
			cmd:           `bash run.sh --jitconfig "${ACTIONS_RUNNER_INPUT_JITCONFIG}"`,
			wantExpanded:  `bash run.sh --jitconfig "s3cr3t"`,
			wantSensitive: true,
		},
		{
			name:         "plain command is untouched",
			cmd:          "sudo sysctl fs.inotify.max_user_instances=1280",
			wantExpanded: "sudo sysctl fs.inotify.max_user_instances=1280",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, sensitive := expandCommand(tt.cmd)
			if expanded != tt.wantExpanded {
				t.Errorf("expanded = %q, want %q", expanded, tt.wantExpanded)
			}
			if sensitive != tt.wantSensitive {
				t.Errorf("sensitive = %v, want %v", sensitive, tt.wantSensitive)
			}
		})
	}
}
