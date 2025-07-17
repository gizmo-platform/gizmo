package config

var (
	allIntegrations  = IntegrationSlice{IntegrationPCSM}
	integrationNames = map[Integration]string{
		IntegrationPCSM: "BEST Robotics PCSM",
	}

	integrationKeys = func(m map[Integration]string) map[string]Integration {
		out := make(map[string]Integration, len(m))
		for k, v := range m {
			out[v] = k
		}
		return out
	}(integrationNames)
)

// ToStrings converts integrations into a slice of strings.
func (is IntegrationSlice) ToStrings() []string {
	out := make([]string, len(is))
	for idx, integration := range is {
		switch integration {
		case IntegrationPCSM:
			out[idx] = integrationNames[integration]
		}
	}
	return out
}

// Enabled tests to see if a given integration is enabled or not.
func (is IntegrationSlice) Enabled(t Integration) bool {
	for _, i := range is {
		if i == t {
			return true
		}
	}
	return false
}

// IntegrationsFromStrings parses a list of strings and returns the
// corresponding integrations.
func IntegrationsFromStrings(s []string) IntegrationSlice {
	out := IntegrationSlice{}

	for _, name := range s {
		i, ok := integrationKeys[name]
		if !ok {
			continue
		}
		out = append(out, i)
	}

	return out
}
