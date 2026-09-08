package content

import (
	"errors"
	"fmt"
	"strings"

	contentdomain "github.com/savisaluwadana/DevRelOS/internal/domain/content"
	workdomain "github.com/savisaluwadana/DevRelOS/internal/domain/workitems"
)

var channelFormats = map[string]map[string]bool{
	"blog":        {"article": true, "tutorial": true},
	"linkedin":    {"social_post": true},
	"x":           {"social_post": true, "thread": true},
	"newsletter":  {"newsletter": true},
	"youtube":     {"video_script": true, "tutorial": true},
	"short_video": {"short_script": true},
	"docs":        {"documentation": true, "tutorial": true},
	"talk":        {"talk_outline": true},
	"community":   {"community_post": true},
}

func ValidChannelFormat(channel, format string) bool {
	formats, ok := channelFormats[channel]
	return ok && formats[format]
}

func FromWorkItem(item workdomain.WorkItem, channel, format, audience string) (contentdomain.Asset, error) {
	channel = strings.TrimSpace(channel)
	format = strings.TrimSpace(format)
	if !ValidChannelFormat(channel, format) {
		return contentdomain.Asset{}, errors.New("invalid channel/format combination")
	}
	if strings.TrimSpace(audience) == "" {
		audience = audienceFromMetadata(item.Metadata)
	}
	if audience == "" {
		audience = "developers"
	}

	topics := topicsFromMetadata(item.Metadata)
	objective := objectiveFor(channel, item.Kind)
	brief := buildBrief(item, channel, format, audience, topics)
	metadata := map[string]any{
		"source_type":   item.SourceType,
		"source_id":     item.SourceID,
		"work_kind":     item.Kind,
		"work_priority": item.Priority,
		"work_status":   item.Status,
	}
	if item.Owner != "" {
		metadata["work_owner"] = item.Owner
	}
	if item.DueAt != nil {
		metadata["work_due_at"] = item.DueAt
	}
	for _, key := range []string{"persona", "evidenceCount", "severity", "trendScore"} {
		if value, ok := item.Metadata[key]; ok {
			metadata[key] = value
		}
	}

	return contentdomain.Asset{
		ProjectID:  item.ProjectID,
		WorkItemID: item.ID,
		Channel:    channel,
		Format:     format,
		Title:      editorialTitle(item.Title),
		Audience:   audience,
		Objective:  objective,
		Brief:      brief,
		Status:     "brief",
		Topics:     topics,
		SourceURL:  stringMetadata(item.Metadata, "sourceUrl"),
		Metadata:   metadata,
	}, nil
}

func buildBrief(item workdomain.WorkItem, channel, format, audience string, topics []string) string {
	var b strings.Builder
	b.WriteString("Problem / opportunity\n")
	if strings.TrimSpace(item.Description) != "" {
		b.WriteString(strings.TrimSpace(item.Description))
	} else {
		b.WriteString(strings.TrimSpace(item.Title))
	}
	b.WriteString("\n\nAudience\n")
	b.WriteString(audience)
	b.WriteString("\n\nEditorial goal\n")
	b.WriteString(objectiveFor(channel, item.Kind))
	b.WriteString("\n\nEvidence / provenance\n")
	b.WriteString(fmt.Sprintf("Source: %s", item.SourceType))
	if item.SourceID != "" {
		b.WriteString(" · ")
		b.WriteString(item.SourceID)
	}
	b.WriteString(fmt.Sprintf("\nWork priority: %d/100", item.Priority))
	if evidenceCount, ok := numberMetadata(item.Metadata, "evidenceCount"); ok {
		b.WriteString(fmt.Sprintf("\nEvidence records: %d", evidenceCount))
	}
	if severity, ok := numberMetadata(item.Metadata, "severity"); ok {
		b.WriteString(fmt.Sprintf("\nPain severity: %d/100", severity))
	}
	if len(topics) > 0 {
		b.WriteString("\n\nTopics\n")
		b.WriteString(strings.Join(topics, ", "))
	}
	b.WriteString("\n\nRecommended structure\n")
	b.WriteString(structureFor(format))
	b.WriteString("\n\nReview gate\n")
	b.WriteString("Verify every technical claim against source evidence, remove unsupported assumptions, and keep the final call-to-action appropriate to the channel.")
	return b.String()
}

func objectiveFor(channel, kind string) string {
	switch channel {
	case "docs":
		return "Resolve the developer friction with precise, task-oriented guidance that is easy to verify and maintain."
	case "blog":
		return "Teach the underlying problem and solution with enough technical depth to create durable search and learning value."
	case "linkedin":
		return "Turn the evidence into a concise professional insight that invites practitioner discussion rather than product promotion."
	case "x":
		return "Compress the evidence into a sharp, useful technical observation with a clear reason to engage or learn more."
	case "newsletter":
		return "Explain why this developer issue matters now and give readers a useful next step or deeper resource."
	case "youtube", "short_video":
		return "Translate the developer problem into a visual, demo-friendly explanation with a clear learning payoff."
	case "talk":
		return "Build a practitioner-first talk around the recurring problem, concrete lessons and reusable takeaways."
	case "community":
		return "Start a useful community conversation grounded in observed developer evidence and invite comparable experiences."
	default:
		if kind == "docs_improvement" {
			return "Resolve the documented developer friction."
		}
		return "Turn the evidence-backed work item into useful developer-facing material."
	}
}

func structureFor(format string) string {
	switch format {
	case "article":
		return "1. Strong problem-led hook\n2. Context and evidence\n3. Technical explanation\n4. Practical solution or trade-offs\n5. Concrete example\n6. Takeaway and conversational CTA"
	case "tutorial":
		return "1. Outcome and prerequisites\n2. Reproduce the problem\n3. Step-by-step implementation\n4. Verification\n5. Failure modes / troubleshooting\n6. Next steps"
	case "social_post":
		return "1. One-sentence hook\n2. Evidence-backed observation\n3. Why it matters\n4. Practical takeaway\n5. Conversation-starting CTA"
	case "thread":
		return "1. Hook\n2. Problem evidence\n3. Technical context\n4. Key lessons in short steps\n5. Example\n6. Summary / resource"
	case "newsletter":
		return "1. Why this matters this week\n2. Developer evidence\n3. Explanation\n4. Recommended action\n5. Resource / CTA"
	case "video_script":
		return "1. 5–10 second hook\n2. Show the pain\n3. Explain why it happens\n4. Demo the solution\n5. Verify outcome\n6. Closing takeaway"
	case "short_script":
		return "1. Immediate hook\n2. One problem\n3. One useful explanation or demo\n4. One memorable takeaway\n5. Short CTA"
	case "documentation":
		return "1. Goal\n2. Prerequisites\n3. Exact procedure\n4. Expected result\n5. Troubleshooting\n6. Related references"
	case "talk_outline":
		return "1. Audience problem\n2. Why existing approaches fail\n3. Mental model\n4. Technical walkthrough\n5. Lessons / trade-offs\n6. Practical takeaways"
	case "community_post":
		return "1. Specific observation\n2. Evidence/context\n3. Open question\n4. Optional resource for deeper discussion"
	default:
		return "Problem → evidence → explanation → practical takeaway → reviewable CTA"
	}
}

func editorialTitle(title string) string {
	title = strings.TrimSpace(title)
	prefixes := []string{"Content brief: ", "Docs improvement: ", "Product feedback: ", "Talk idea: ", "Community research: ", "Engineering task: "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(title, prefix) {
			title = strings.TrimSpace(strings.TrimPrefix(title, prefix))
			break
		}
	}
	if title == "" {
		title = "Developer content asset"
	}
	return title
}

func topicsFromMetadata(metadata map[string]any) []string {
	if metadata == nil {
		return []string{}
	}
	value, ok := metadata["topics"]
	if !ok {
		return []string{}
	}
	items := make([]string, 0)
	switch typed := value.(type) {
	case []string:
		items = append(items, typed...)
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				items = append(items, text)
			}
		}
	case string:
		items = append(items, strings.Split(typed, ",")...)
	}
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item = strings.TrimSpace(strings.ToLower(item))
		item = strings.ReplaceAll(item, " ", "-")
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func audienceFromMetadata(metadata map[string]any) string {
	persona := stringMetadata(metadata, "persona")
	if persona != "" {
		return persona
	}
	return stringMetadata(metadata, "audience")
}

func stringMetadata(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func numberMetadata(metadata map[string]any, key string) (int, bool) {
	if metadata == nil {
		return 0, false
	}
	switch value := metadata[key].(type) {
	case int:
		return value, true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}
