package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseEntry(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Record
		wantErr bool
	}{
		{
			name: "simple",
			in:   ": 1700000000:0;ls -la",
			want: Record{Timestamp: 1700000000, Elapsed: 0, Command: "ls -la"},
		},
		{
			name: "nonzero elapsed",
			in:   ": 1700000000:5;sleep 5",
			want: Record{Timestamp: 1700000000, Elapsed: 5, Command: "sleep 5"},
		},
		{
			name: "command with semicolons",
			in:   ": 1700000000:0;echo a; echo b",
			want: Record{Timestamp: 1700000000, Elapsed: 0, Command: "echo a; echo b"},
		},
		{
			name: "command with colons",
			in:   ": 1700000000:0;echo a:b:c",
			want: Record{Timestamp: 1700000000, Elapsed: 0, Command: "echo a:b:c"},
		},
		{
			name: "unicode command",
			in:   ": 1700000000:0;echo 你好",
			want: Record{Timestamp: 1700000000, Elapsed: 0, Command: "echo 你好"},
		},
		{
			name: "wide timestamp (post-2286)",
			in:   ": 99999999999:0;date",
			want: Record{Timestamp: 99999999999, Elapsed: 0, Command: "date"},
		},
		{name: "no prefix", in: "ls -la", wantErr: true},
		{name: "missing colon after ts", in: ": 1700000000;ls", wantErr: true},
		{name: "missing semicolon", in: ": 1700000000:0 ls -la", wantErr: true},
		{name: "non-numeric ts", in: ": abcdef:0;ls", wantErr: true},
		{name: "non-numeric elapsed", in: ": 1700000000:x;ls", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEntry(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseAll_Multiline(t *testing.T) {
	// Two entries; the first command is continued across two physical lines.
	input := strings.Join([]string{
		`: 1700000000:0;echo first \`,
		`continued`,
		`: 1700000100:0;echo second`,
		``,
	}, "\n")

	got, skipped, err := ParseAll(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	if skipped != 0 {
		t.Errorf("skipped=%d, want 0", skipped)
	}
	want := []Record{
		{Timestamp: 1700000000, Elapsed: 0, Command: "echo first \\\ncontinued"},
		{Timestamp: 1700000100, Elapsed: 0, Command: "echo second"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseAll_SkipsJunk(t *testing.T) {
	input := strings.Join([]string{
		`garbage with no prefix`,
		`: notnumber:0;cmd`,
		`: 1700000000:0;good`,
		``,
	}, "\n")

	got, skipped, err := ParseAll(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	if skipped != 2 {
		t.Errorf("skipped=%d, want 2", skipped)
	}
	if len(got) != 1 || got[0].Command != "good" {
		t.Fatalf("got %+v", got)
	}
}

func TestRoundTrip(t *testing.T) {
	src := []Record{
		{Timestamp: 1700000000, Elapsed: 0, Command: "ls -la"},
		{Timestamp: 1700000001, Elapsed: 3, Command: "echo a; echo b"},
		{Timestamp: 1700000002, Elapsed: 0, Command: "echo 你好"},
	}
	var b strings.Builder
	for _, r := range src {
		b.WriteString(r.Marshal())
		b.WriteByte('\n')
	}
	got, skipped, err := ParseAll(strings.NewReader(b.String()))
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	if skipped != 0 {
		t.Errorf("skipped=%d, want 0", skipped)
	}
	if !reflect.DeepEqual(got, src) {
		t.Fatalf("got %+v\nwant %+v", got, src)
	}
}
