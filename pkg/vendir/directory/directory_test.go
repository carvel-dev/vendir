package directory

import "testing"

func Test_handleLazySync(t *testing.T) {
	type args struct {
		oldConfigDigest   string
		newConfigDigest   string
		fetchLazyOverride bool
		fetchContentLazy  bool
	}
	tests := []struct {
		name            string
		args            args
		skipFetching    bool
		addConfigDigest bool
	}{
		{
			"no skipping of sync or adding of config digest, if lazy sync is disabled globally and locally",
			args{"same", "same", false, false},
			false,
			false,
		},
		{
			"no skipping of sync or adding of config digest, if lazy sync is enabled locally",
			args{"same", "same", true, false},
			false,
			false,
		},
		{
			"no skipping of sync but adding of config digest, if lazy sync is disabled globally",
			args{"same", "same", false, true},
			false,
			true,
		},
		{
			"no skipping of sync but adding of config digest, if lazy sync is enabled but config digests don't match",
			args{"same", "not-same", true, true},
			false,
			true,
		},
		{
			"skipping of sync and adding of config digest, if lazy sync is enabled and config digests match",
			args{"same", "same", true, true},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skipFetching, addConfigDigest := handleLazySync(tt.args.oldConfigDigest, tt.args.newConfigDigest, tt.args.fetchLazyOverride, tt.args.fetchContentLazy)
			if skipFetching != tt.skipFetching {
				t.Errorf("handleLazySync() got = %v, want %v", skipFetching, tt.skipFetching)
			}
			if addConfigDigest != tt.addConfigDigest {
				t.Errorf("handleLazySync() got1 = %v, want %v", addConfigDigest, tt.addConfigDigest)
			}
		})
	}
}
