package utils

import "testing"

func Test_checkAddr(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test checkAddr with valid IPv6 address",
			args: args{
				addr: "[2001:db8::1]:8080",
			},
			want: true,
		},
		{
			name: "Test checkAddr with invalid port",
			args: args{
				addr: "127.0.0.1:99999",
			},
			want: false,
		},
		{
			name: "Test checkAddr with non-numeric port",
			args: args{
				addr: "127.0.0.1:port",
			},
			want: false,
		},
		{
			name: "Test checkAddr with missing port",
			args: args{
				addr: "127.0.0.1:",
			},
			want: false,
		},
		{
			name: "Test checkAddr with empty address",
			args: args{
				addr: "",
			},
			want: false,
		},
		{
			name: "Test checkAddr with only empty ip and port",
			args: args{
				addr: ":",
			},
			want: false,
		},
		{
			name: "Test checkAddr with only port",
			args: args{
				addr: ":8080",
			},
			want: true,
		},
		{
			name: "Test checkAddr with valid address and port 0",
			args: args{
				addr: "127.0.0.1:0",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckAddr(tt.args.addr); got != tt.want {
				t.Errorf("checkAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}
