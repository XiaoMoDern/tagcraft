package main

import (
	"strings"
	"testing"
)

func TestSanitizeSEODropsBanned(t *testing.T) {
	seo := seoContent{
		Title:       "Navajo Style Turquoise Ring | Handmade",
		Tags:        []string{"navajo ring", "sterling silver", "healing crystal", "disney mug"},
		Description: "A healing crystal that cures anxiety. Inspired by Disney.",
	}
	sanitizeSEO(&seo)

	if bannedRegex.MatchString(seo.Title) {
		t.Errorf("title still contains banned term: %q", seo.Title)
	}
	if bannedRegex.MatchString(seo.Description) {
		t.Errorf("description still contains banned term: %q", seo.Description)
	}
	// 三个命中词都应被丢弃，只有 "sterling silver" 幸存
	if len(seo.Tags) != 1 || seo.Tags[0] != "sterling silver" {
		t.Errorf("expected only 'sterling silver' to survive, got %v", seo.Tags)
	}
}

func TestSanitizeKeepsColorWords(t *testing.T) {
	seo := seoContent{
		Title:       "Ivory Wedding Guest Book | Coral Ribbon",
		Tags:        []string{"ivory guest book", "coral ribbon", "rosewood frame"},
		Description: "An ivory colored guest book with coral ribbon.",
	}
	sanitizeSEO(&seo)

	// ivory / coral / rosewood 作为颜色/花纹不应被误杀
	if len(seo.Tags) != 3 {
		t.Errorf("color words should not be stripped, got %v", seo.Tags)
	}
}

func TestSanitizeWordBoundary(t *testing.T) {
	// "secure" 里含 "cure"，但不该被命中（\b 边界）
	seo := seoContent{
		Title:       "Secure Packaging",
		Tags:        []string{"secure clasp", "curated gift"},
		Description: "Secure packaging.",
	}
	sanitizeSEO(&seo)

	if len(seo.Tags) != 2 {
		t.Errorf("word-boundary false positive: got %v", seo.Tags)
	}
}

func TestSanitizeClaimsStripsUnclaimed(t *testing.T) {
	// 卖家输入没有 vegan/eco friendly，输出里出现应被剔除
	seo := seoContent{
		Title:       "Vegan Soap | Eco Friendly",
		Tags:        []string{"vegan soap", "eco friendly soap", "lavender soap"},
		Description: "A vegan, eco friendly soap bar.",
	}
	sanitizeClaims(&seo, "handmade lavender soap bar", "", "Bath & Beauty")

	joined := strings.ToLower(seo.Title + " " + strings.Join(seo.Tags, " ") + " " + seo.Description)
	if strings.Contains(joined, "vegan") || strings.Contains(joined, "eco friendly") {
		t.Errorf("claim words not stripped: title=%q tags=%v", seo.Title, seo.Tags)
	}
	found := false
	for _, tag := range seo.Tags {
		if tag == "lavender soap" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'lavender soap' to survive, got %v", seo.Tags)
	}
}

func TestSanitizeClaimsKeepsEchoed(t *testing.T) {
	// 卖家明确写了 sterling silver，输出里的 sterling silver 应保留
	seo := seoContent{
		Title:       "Sterling Silver Ring",
		Tags:        []string{"sterling silver ring", "silver ring"},
		Description: "A sterling silver ring.",
	}
	sanitizeClaims(&seo, "sterling silver ring", "sterling silver", "Jewelry")
	if len(seo.Tags) != 2 {
		t.Errorf("echoed claim should be kept, got %v", seo.Tags)
	}
}
