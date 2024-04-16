package main

import (
	"reflect"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
)

func TestNewAwsLanguageToolStack(t *testing.T) {
	type args struct {
		scope constructs.Construct
		id    *string
		props *AwsLanguageToolStackProps
	}
	tests := []struct {
		name string
		args args
		want awscdk.Stack
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAwsLanguageToolStack(tt.args.scope, tt.args.id, tt.args.props); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAwsLanguageToolStack() = %v, want %v", got, tt.want)
			}
		})
	}
}
