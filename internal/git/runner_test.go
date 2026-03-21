package git

import "testing"

func TestParseStatusPorcelainV1Z(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []FileStatus
	}{
		{
			name:  "preserves spaces in file names",
			input: " M config.yaml\x00?? file with spaces.yaml\x00",
			want: []FileStatus{
				{IndexStatus: ' ', WorkTreeStatus: 'M', Path: "config.yaml"},
				{IndexStatus: '?', WorkTreeStatus: '?', Path: "file with spaces.yaml"},
			},
		},
		{
			name:  "uses destination path for renames",
			input: "R  old name.txt\x00new name.txt\x00",
			want: []FileStatus{
				{IndexStatus: 'R', WorkTreeStatus: ' ', Path: "new name.txt"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStatusPorcelainV1Z(tt.input)
			if len(got.Files) != len(tt.want) {
				t.Fatalf("got %d files, want %d", len(got.Files), len(tt.want))
			}
			for i := range tt.want {
				if got.Files[i] != tt.want[i] {
					t.Fatalf("file %d = %#v, want %#v", i, got.Files[i], tt.want[i])
				}
			}
		})
	}
}
