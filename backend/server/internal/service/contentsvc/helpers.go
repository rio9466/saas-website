package contentsvc

import "github.com/rio9466/easy-admin/server/internal/domain/content"

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func clonePayload(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func clonePayloadMap(in map[string]map[string]any) map[string]map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string]any, len(in))
	for locale, payload := range in {
		out[locale] = clonePayload(payload)
	}
	return out
}

func clonePlanTranslations(in map[string]content.PricingPlanTranslation) map[string]content.PricingPlanTranslation {
	if in == nil {
		return nil
	}
	out := make(map[string]content.PricingPlanTranslation, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneFeatureLists(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}

func sanitizeFeatureTranslations(in map[string]content.FeatureTranslation) (map[string]content.FeatureTranslation, error) {
	if in == nil {
		return nil, nil
	}
	out := make(map[string]content.FeatureTranslation, len(in))
	for locale, t := range in {
		if err := validateOptionalURL("image_url", t.ImageURL, false); err != nil {
			return nil, err
		}
		t.BodyMD = sanitizeMarkdown(t.BodyMD)
		out[locale] = t
	}
	return out, nil
}

func sanitizePageTranslations(in map[string]content.PageTranslation) (map[string]content.PageTranslation, error) {
	if in == nil {
		return nil, nil
	}
	out := make(map[string]content.PageTranslation, len(in))
	for locale, t := range in {
		t.BodyMD = sanitizeMarkdown(t.BodyMD)
		out[locale] = t
	}
	return out, nil
}

func sanitizeArticleTranslations(in map[string]content.DocArticleTranslation) (map[string]content.DocArticleTranslation, error) {
	if in == nil {
		return nil, nil
	}
	out := make(map[string]content.DocArticleTranslation, len(in))
	for locale, t := range in {
		t.BodyMD = sanitizeMarkdown(t.BodyMD)
		out[locale] = t
	}
	return out, nil
}
