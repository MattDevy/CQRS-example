package reservations

import (
	"testing"
	"time"
)

func Test_timeIntersect(t *testing.T) {
	timeNow, err := time.Parse(time.RFC3339, "2021-06-30T18:42:28.320Z")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		start1 time.Time
		end1   time.Time
		start2 time.Time
		end2   time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No intersect",
			args: args{
				start1: timeNow.Add(-1 * time.Hour),
				end1:   timeNow.Add(-30 * time.Minute),
				start2: timeNow,
				end2:   timeNow.Add(1 * time.Hour),
			},
			want: false,
		},
		{
			name: "intersect end1 > start2",
			args: args{
				start1: timeNow.Add(-1 * time.Hour),
				end1:   timeNow.Add(1 * time.Second),
				start2: timeNow,
				end2:   timeNow.Add(1 * time.Hour),
			},
			want: true,
		},
		{
			name: "intersect end1 > end2 && start1 < start2 ",
			args: args{
				start1: timeNow.Add(-1 * time.Hour),
				end1:   timeNow.Add(2 * time.Hour),
				start2: timeNow,
				end2:   timeNow.Add(1 * time.Hour),
			},
			want: true,
		},
		{
			name: "intersect end2 > end1 && start2 < start1",
			args: args{
				start2: timeNow.Add(-1 * time.Hour),
				end2:   timeNow.Add(2 * time.Hour),
				start1: timeNow,
				end1:   timeNow.Add(1 * time.Hour),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeIntersect(tt.args.start1, tt.args.end1, tt.args.start2, tt.args.end2); got != tt.want {
				t.Errorf("timeIntersect() = %v, want %v", got, tt.want)
			}
		})
	}
}
