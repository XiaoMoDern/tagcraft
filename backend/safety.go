package main

import (
	"regexp"
	"strings"
)

// bannedTerms 是「封店级」硬词表：一旦出现几乎必然违规/违法，与上下文无关。
//
// 刻意排除 "ivory" / "coral" / "rosewood" / "tortoiseshell" 这类「既是颜色/花纹、又是
// 受限材料」的歧义词（"ivory wedding dress" 是合法颜色用法），交给 prompt 层用语境判断。
var bannedTerms = []string{
	// 原住民/部落起源声明（Indian Arts and Crafts Act，Etsy 封店级）
	"native american", "navajo", "zuni", "hopi", "cherokee", "sioux", "apache", "inuit",
	// 大牌 IP / 商标
	"disney", "harry potter", "pokemon", "marvel", "nike", "chanel", "louis vuitton",
	"star wars", "lego", "barbie", "gucci", "nintendo", "mario", "mickey mouse", "hello kitty",
	// 医疗疗效声明（Etsy Medical Claims 政策）
	"healing", "cures", "cure", "anti-aging", "antiaging", "fda approved", "therapeutic", "medicinal",
}

var bannedRegex = regexp.MustCompile(`(?i)\b(?:` + strings.Join(quoteAll(bannedTerms), "|") + `)\b`)
var multiSpaceRe = regexp.MustCompile(`\s+`)

func quoteAll(terms []string) []string {
	out := make([]string, len(terms))
	for i, t := range terms {
		out[i] = regexp.QuoteMeta(t)
	}
	return out
}

// sanitizeSEO 是 prompt 之外的最后一道兜底：
//   - 命中硬词的 tag 整条丢弃（Etsy 允许少于 13 个 tag，少比违禁好）
//   - 命中硬词的 title/description 移除该词（宁可句子稍别扭，不留法律风险）
func sanitizeSEO(seo *seoContent) {
	seo.Title = multiSpaceRe.ReplaceAllString(bannedRegex.ReplaceAllString(seo.Title, " "), " ")
	seo.Description = multiSpaceRe.ReplaceAllString(bannedRegex.ReplaceAllString(seo.Description, " "), " ")

	kept := seo.Tags[:0]
	for _, tag := range seo.Tags {
		if !bannedRegex.MatchString(tag) {
			kept = append(kept, tag)
		}
	}
	seo.Tags = kept

	seo.Title = strings.TrimSpace(seo.Title)
	seo.Description = strings.TrimSpace(seo.Description)
}

// claimTerms 是 Tier-2「声明词」：绿色/道德、安全合规、材质等级。
// 这些词只有当逐字出现在卖家输入里才允许出现在输出中，防止模型把 "handmade soap"
// 默认成 "vegan soap"、把 "silver ring" 升级成 "sterling silver"。
// 刻意排除 "organic"/"natural"（"organic shapes"、"natural stone" 是合法描述，非声明）。
var claimTerms = []string{
	// 绿色/道德
	"vegan", "cruelty free", "cruelty-free", "eco friendly", "eco-friendly",
	"sustainable", "fair trade",
	// 安全/合规
	"hypoallergenic", "nickel free", "nickel-free", "lead free", "lead-free",
	"bpa free", "bpa-free", "food safe", "food-safe", "fda approved",
	// 材质等级
	"solid gold", "solid silver", "14k", "18k", "24k", "sterling silver",
	"genuine leather", "real leather", "real diamond", "100% silk", "pure silk",
}

// sanitizeClaims 逐请求校验声明词：只保留「卖家输入里逐字出现过的」声明词，其余剔除。
func sanitizeClaims(seo *seoContent, product, keywords, category string) {
	input := strings.ToLower(product + " " + keywords + " " + category)

	disallowed := make([]string, 0, len(claimTerms))
	for _, term := range claimTerms {
		if !strings.Contains(input, term) {
			disallowed = append(disallowed, term)
		}
	}
	if len(disallowed) == 0 {
		return
	}

	re := regexp.MustCompile(`(?i)\b(?:` + strings.Join(quoteAll(disallowed), "|") + `)\b`)

	seo.Title = multiSpaceRe.ReplaceAllString(re.ReplaceAllString(seo.Title, " "), " ")
	seo.Description = multiSpaceRe.ReplaceAllString(re.ReplaceAllString(seo.Description, " "), " ")

	kept := seo.Tags[:0]
	for _, tag := range seo.Tags {
		if !re.MatchString(tag) {
			kept = append(kept, tag)
		}
	}
	seo.Tags = kept

	seo.Title = strings.TrimSpace(seo.Title)
	seo.Description = strings.TrimSpace(seo.Description)
}
