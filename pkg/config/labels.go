package config

func DefaultLabels(meta any) map[string]string {
	c, ok := meta.(map[string]any)
	if !ok {
		return nil
	}

	v, ok := c["default_labels"]
	if !ok {
		return nil
	}

	labels, ok := v.(map[string]string)
	if !ok {
		return nil
	}

	return labels
}

func LabelsWithDefaults(meta any, labels map[string]string) map[string]string {
	defaultLabels := DefaultLabels(meta)
	if len(defaultLabels) == 0 && len(labels) == 0 {
		return nil
	}

	out := make(map[string]string, len(defaultLabels)+len(labels))
	for k, v := range defaultLabels {
		out[k] = v
	}
	for k, v := range labels {
		out[k] = v
	}
	return out
}

func LabelsWithoutDefaults(meta any, labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}

	defaultLabels := DefaultLabels(meta)
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
