package provider

import "testing"

func TestProviderSettingsFromAllowsMissingProviderData(t *testing.T) {
	t.Parallel()
	if _, diags := providerSettingsFrom(nil); len(diags) != 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}

func TestProviderSettingsFromRejectsOtherType(t *testing.T) {
	t.Parallel()
	if _, diags := providerSettingsFrom(&providerSettings{}); !diags.HasError() {
		t.Error("expected an error for provider data that is not providerSettings")
	}
}
