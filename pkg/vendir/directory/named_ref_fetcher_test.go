// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"reflect"
	"testing"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
)

func TestNamedRefFetcher_GetSecret(t *testing.T) {

	secret := ctlconf.Secret{
		Metadata: ctlconf.GenericMetadata{
			Name: "my-secret-1",
		},
		Data: map[string][]byte{"foo": []byte("bar")},
	}

	type fields struct {
		secrets    []ctlconf.Secret
		configMaps []ctlconf.ConfigMap
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    ctlconf.Secret
		wantErr bool
	}{
		{
			name: "not found secret over no secrets",
			fields: fields{
				secrets: []ctlconf.Secret{},
			},
			args: args{
				name: "non-secret",
			},
			wantErr: true,
		},
		{
			name: "not found secret over one secret",
			fields: fields{
				secrets: []ctlconf.Secret{secret},
			},
			args: args{
				name: "non-secret",
			},
			wantErr: true,
		},
		{
			name: "found secret over one secret",
			fields: fields{
				secrets: []ctlconf.Secret{
					{
						Metadata: ctlconf.GenericMetadata{
							Name: "my-secret-1",
						},
						Data: map[string][]byte{"foo": []byte("bar")},
					},
				},
			},
			args: args{
				name: "my-secret-1",
			},
			want: secret,
		},
		{
			name: "found secret over similar (3) secrets",
			fields: fields{
				secrets: []ctlconf.Secret{
					{
						Metadata: ctlconf.GenericMetadata{
							Name: "my-secret-1",
						},
						Data: map[string][]byte{"foo": []byte("bar")},
					},
					secret,
					{
						Metadata: ctlconf.GenericMetadata{
							Name: "my-secret-1",
						},
						Data: map[string][]byte{"foo": []byte("bar")},
					},
				},
			},
			args: args{
				name: "my-secret-1",
			},
			want: secret,
		},
		{
			name: "found secret over different (3) secrets",
			fields: fields{
				secrets: []ctlconf.Secret{
					{
						Metadata: ctlconf.GenericMetadata{
							Name: "my-secret-2",
						},
						Data: map[string][]byte{"foo": []byte("bar")},
					},
					secret,
					{
						Metadata: ctlconf.GenericMetadata{
							Name: "my-secret-3",
						},
						Data: map[string][]byte{"foo": []byte("bar")},
					},
				},
			},
			args: args{
				name: "my-secret-1",
			},
			want: secret,
		},
		{
			name: "error due to different data over secrets",
			fields: fields{
				secrets: []ctlconf.Secret{
					secret,
					{
						Metadata: ctlconf.GenericMetadata{
							Name: "my-secret-1",
						},
						Data: map[string][]byte{"foo": []byte("baz")},
					},
					{
						Metadata: ctlconf.GenericMetadata{
							Name: "my-secret-1",
						},
						Data: map[string][]byte{"foo": []byte("bar")},
					},
				},
			},
			args: args{
				name: "my-secret-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NamedRefFetcher{
				secrets:    tt.fields.secrets,
				configMaps: tt.fields.configMaps,
			}
			got, err := f.GetSecret(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("NamedRefFetcher.GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NamedRefFetcher.GetSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}
