package messageBuilder

// MirrorFileAsset describes a static media asset that may need per-mirror file_id caching.
type MirrorFileAsset struct {
	PrimaryFileID string
	FallbackPath  string
	MimeType      string
	MirrorFileKey string
}
