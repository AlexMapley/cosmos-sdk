package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
)

// testing of invalid version formats exist within 24-host/validate_test.go
func TestUnpackVersion(t *testing.T) {
	testCases := []struct {
		name          string
		version       string
		expIdentifier string
		expFeatures   []string
		expPass       bool
	}{
		{"valid version", "(1,[ORDERED channel,UNORDERED channel])", "1", []string{"ORDERED channel", "UNORDERED channel"}, true},
		{"valid empty features", "(1,[])", "1", []string{}, true},
		{"empty identifier", "(,[features])", "", []string{}, false},
		{"invalid version", "identifier,[features]", "", []string{}, false},
		{"empty string", "  ", "", []string{}, false},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			identifier, features, err := types.UnpackVersion(tc.version)

			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, tc.expIdentifier, identifier)
				require.Equal(t, tc.expFeatures, features)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestFindSupportedVersion(t *testing.T) {
	testCases := []struct {
		name              string
		version           string
		supportedVersions []string
		expVersion        string
		expFound          bool
	}{
		{"valid supported version", types.DefaultConnectionVersion, types.GetCompatibleVersions(), types.DefaultConnectionVersion, true},
		{"empty (invalid) version", "", types.GetCompatibleVersions(), "", false},
		{"empty supported versions", types.DefaultConnectionVersion, []string{}, "", false},
		{"desired version is last", types.DefaultConnectionVersion, []string{"(validversion,[])", "(2,[feature])", "(3,[])", types.DefaultConnectionVersion}, types.DefaultConnectionVersion, true},
		{"desired version identifier with different feature set", "(1,[features])", types.GetCompatibleVersions(), types.DefaultConnectionVersion, true},
		{"version not supported", "(2,[DAG])", types.GetCompatibleVersions(), "", false},
	}

	for i, tc := range testCases {
		version, found := types.FindSupportedVersion(tc.version, tc.supportedVersions)

		require.Equal(t, tc.expVersion, version, "test case %d: %s", i, tc.name)
		require.Equal(t, tc.expFound, found, "test case %d: %s", i, tc.name)
	}
}

func TestPickVersion(t *testing.T) {
	testCases := []struct {
		name                 string
		counterpartyVersions []string
		expVer               string
		expPass              bool
	}{
		{"valid default ibc version", types.GetCompatibleVersions(), types.DefaultConnectionVersion, true},
		{"valid version in counterparty versions", []string{"(version1,[])", "(2.0.0,[DAG,ZK])", types.DefaultConnectionVersion}, types.DefaultConnectionVersion, true},
		{"valid identifier match but empty feature set not allowed", []string{"(1,[DAG,ORDERED-ZK,UNORDERED-zk])"}, "(1,[])", false},
		{"empty counterparty versions", []string{}, "", false},
		{"non-matching counterparty versions", []string{"(2.0.0,[])"}, "", false},
	}

	for i, tc := range testCases {
		version, err := types.PickVersion(types.GetCompatibleVersions(), tc.counterpartyVersions, types.AllowNilFeatureSetMap)

		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
			require.Equal(t, tc.expVer, version, "valid test case %d falied: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
			require.Equal(t, "", version, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func TestVerifyProposedFeatureSet(t *testing.T) {
	testCases := []struct {
		name             string
		proposedVersion  string
		supportedVersion string
		expPass          bool
	}{
		{"entire feature set supported", types.DefaultConnectionVersion, types.CreateVersionString("1", []string{"ORDERED", "UNORDERED", "DAG"}), true},
		{"empty feature sets not supported", types.CreateVersionString("1", []string{}), types.DefaultConnectionVersion, false},
		{"one feature missing", types.DefaultConnectionVersion, types.CreateVersionString("1", []string{"UNORDERED", "DAG"}), false},
		{"both features missing", types.DefaultConnectionVersion, types.CreateVersionString("1", []string{"DAG"}), false},
	}

	for i, tc := range testCases {
		supported := types.VerifyProposedFeatureSet(tc.proposedVersion, tc.supportedVersion, types.AllowNilFeatureSetMap)

		require.Equal(t, tc.expPass, supported, "test case %d: %s", i, tc.name)
	}

}
