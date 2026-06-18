package config

func StringMapFromAnyMap(in map[string]any) map[string]string {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v.(string)
	}

	return out
}

func MergeLabels(defaultLabels, resourceLabels map[string]string) map[string]string {
	if len(defaultLabels) == 0 && len(resourceLabels) == 0 {
		return nil
	}

	out := make(map[string]string, len(defaultLabels)+len(resourceLabels))

	for k, v := range defaultLabels {
		out[k] = v
	}

	for k, v := range resourceLabels {
		out[k] = v
	}

	return out
}

func StripDefaultLabels(labels, defaultLabels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}

	out := make(map[string]string, len(labels))

	for k, v := range labels {
		if dv, ok := defaultLabels[k]; ok && dv == v {
			continue
		}
		out[k] = v
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
