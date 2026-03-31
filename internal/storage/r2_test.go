package storage

import "testing"

func TestNormalizeR2PublicURLAddsBucketPathWhenEndpointCarriesBucket(t *testing.T) {
	got := normalizeR2PublicURL(
		"https://pub-example.r2.dev",
		"https://accountid.r2.cloudflarestorage.com/myevent",
		"myevent",
	)

	want := "https://pub-example.r2.dev/myevent"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestNormalizeR2PublicURLKeepsExplicitPublicPath(t *testing.T) {
	got := normalizeR2PublicURL(
		"https://pub-example.r2.dev/myevent",
		"https://accountid.r2.cloudflarestorage.com/myevent",
		"myevent",
	)

	want := "https://pub-example.r2.dev/myevent"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestNormalizeR2PublicURLIgnoresEndpointWithoutBucketPath(t *testing.T) {
	got := normalizeR2PublicURL(
		"https://pub-example.r2.dev",
		"https://accountid.r2.cloudflarestorage.com",
		"myevent",
	)

	want := "https://pub-example.r2.dev"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
