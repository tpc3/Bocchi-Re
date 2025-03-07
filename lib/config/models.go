package config

// Type to distinguish model categories
type ModelType string

const (
	ModelTypeText  ModelType = "text"
	ModelTypeImage ModelType = "image"
)

// Structure to consolidate model information
type ModelInfo struct {
	Key          string
	Manufacturer string
	Type         ModelType

	// Text Model
	PromptCost     float64    // USD 1M tokens
	CompletionCost float64    // USD 1M tokens
	VisionCost     VisionCost // USD 1M tokens

	// Image Model
	ImageCost map[string]float64
}

type VisionCost struct {
	Fixed float64
	Base  int
	Tile  int
}

var AllModels = map[string]ModelInfo{
	"gpt-4.5-preview": {
		Key:            "gpt-4.5-preview",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     75.0,
		CompletionCost: 150.0,
		VisionCost: VisionCost{
			Fixed: 75,
			Base:  85,
			Tile:  170,
		},
		ImageCost: nil,
	}, "gpt-4.5-preview-2025-02-27": {
		Key:            "gpt-4.5-preview-2025-02-27",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     75.0,
		CompletionCost: 150.0,
		VisionCost: VisionCost{
			Fixed: 75,
			Base:  85,
			Tile:  170,
		},
		ImageCost: nil,
	},
	"gpt-4o": {
		Key:            "gpt-4o",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     2.50,
		CompletionCost: 10.0,
		VisionCost: VisionCost{
			Fixed: 2.5,
			Base:  85,
			Tile:  170,
		},
		ImageCost: nil,
	},
	"gpt-4o-2024-11-20": {
		Key:            "gpt-4o-2024-11-20",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     2.50,
		CompletionCost: 10.0,
		VisionCost: VisionCost{
			Fixed: 2.5,
			Base:  85,
			Tile:  170,
		},
		ImageCost: nil,
	},
	"gpt-4o-2024-08-06": {
		Key:            "gpt-4o-2024-08-06",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     2.50,
		CompletionCost: 10.0,
		VisionCost: VisionCost{
			Fixed: 2.5,
			Base:  85,
			Tile:  170,
		},
		ImageCost: nil,
	},
	"gpt-4o-2024-05-13": {
		Key:            "gpt-4o-2024-05-13",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     5.0,
		CompletionCost: 15.0,
		VisionCost: VisionCost{
			Fixed: 2.5,
			Base:  85,
			Tile:  170,
		},
		ImageCost: nil,
	},
	"gpt-4o-mini": {
		Key:            "gpt-4o-mini",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     0.150,
		CompletionCost: 0.600,
		VisionCost: VisionCost{
			Fixed: 0.15,
			Base:  2833,
			Tile:  5667,
		},
		ImageCost: nil,
	},
	"gpt-4o-mini-2024-07-18": {
		Key:            "gpt-4o-mini-2024-07-18",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     0.150,
		CompletionCost: 0.600,
		VisionCost: VisionCost{
			Fixed: 0.15,
			Base:  2833,
			Tile:  5667,
		},
		ImageCost: nil,
	},
	"o1": {
		Key:            "o1",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     15.0,
		CompletionCost: 60.0,
		VisionCost: VisionCost{
			Fixed: 15,
			Base:  75,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"o1-2024-12-17": {
		Key:            "o1-2024-12-17",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     15.0,
		CompletionCost: 60.0,
		VisionCost: VisionCost{
			Fixed: 15,
			Base:  75,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"o1-preview": {
		Key:            "o1-preview",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     15.0,
		CompletionCost: 60.0,
		VisionCost: VisionCost{
			Fixed: 15,
			Base:  75,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"o1-preview-2024-09-12": {
		Key:            "o1-preview-2024-09-12",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     15.0,
		CompletionCost: 60.0,
		VisionCost: VisionCost{
			Fixed: 15,
			Base:  75,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"o3-mini": {
		Key:            "o3-mini",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     1.10,
		CompletionCost: 4.40,
		ImageCost:      nil,
	},
	"o3-mini-2025-01-31": {
		Key:            "o3-mini-2025-01-31",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     1.10,
		CompletionCost: 4.40,
		ImageCost:      nil,
	},
	"gpt-4o-latest": {
		Key:            "gpt-4o-latest",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     5.0,
		CompletionCost: 15.0,
		VisionCost: VisionCost{
			Fixed: 2.5,
			Base:  85,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"gpt-4-turbo": {
		Key:            "gpt-4-turbo",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     10.0,
		CompletionCost: 30.0,
		VisionCost: VisionCost{
			Fixed: 10,
			Base:  85,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"gpt-4-turbo-2024-04-09": {
		Key:            "gpt-4-turbo-2024-04-09",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     10.0,
		CompletionCost: 30.0,
		VisionCost: VisionCost{
			Fixed: 10,
			Base:  85,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"gpt-4-0125-preview": {
		Key:            "gpt-4-0125-preview",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     10.0,
		CompletionCost: 30.0,
		ImageCost:      nil,
	},
	"gpt-4-1106-preview": {
		Key:            "gpt-4-1106-preview",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     10.0,
		CompletionCost: 30.0,
		ImageCost:      nil,
	},
	"gpt-4-1106-vision-preview": {
		Key:            "gpt-4-1106-vision-preview",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     10.0,
		CompletionCost: 30.0,
		VisionCost: VisionCost{
			Fixed: 10,
			Base:  85,
			Tile:  255,
		},
		ImageCost: nil,
	},
	"gpt-4": {
		Key:            "gpt-4",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     30.0,
		CompletionCost: 60.0,
		ImageCost:      nil,
	},
	"gpt-4-0613": {
		Key:            "gpt-4-0613",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     30.0,
		CompletionCost: 60.0,
		ImageCost:      nil,
	},
	"gpt-4-0314": {
		Key:            "gpt-4-0314",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     30.0,
		CompletionCost: 60.0,
		ImageCost:      nil,
	},
	"gpt-4-32k": {
		Key:            "gpt-4-32k",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     60.0,
		CompletionCost: 120.0,
		ImageCost:      nil,
	},
	"gpt-3.5-turbo": {
		Key:            "gpt-3.5-turbo",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     0.50,
		CompletionCost: 1.50,
		ImageCost:      nil,
	},
	"gpt-3.5-turbo-0125": {
		Key:            "gpt-3.5-turbo-0125",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     0.50,
		CompletionCost: 1.50,
		ImageCost:      nil,
	},
	"gpt-3.5-turbo-1106": {
		Key:            "gpt-3.5-turbo-1106",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     1.0,
		CompletionCost: 2.0,
		ImageCost:      nil,
	},
	"gpt-3.5-turbo-0613": {
		Key:            "gpt-3.5-turbo-0613",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     1.50,
		CompletionCost: 2.00,
		ImageCost:      nil,
	},
	"gpt-3.5-turbo-instruct": {
		Key:            "gpt-3.5-turbo-instruct",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     1.50,
		CompletionCost: 2.00,
		ImageCost:      nil,
	},
	"gpt-3.5-turbo-16k-0613": {
		Key:            "gpt-3.5-turbo-16k-0613",
		Manufacturer:   "OpenAI",
		Type:           ModelTypeText,
		PromptCost:     3.0,
		CompletionCost: 4.0,
		ImageCost:      nil,
	},
	// DALL-E 2
	"dall-e-2": {
		Key:          "dall-e-2",
		Manufacturer: "OpenAI",
		Type:         ModelTypeImage,
		ImageCost: map[string]float64{
			"small":  0.0160,
			"medium": 0.0180,
			"big":    0.0200,
		},
	},
	// DALL-E 3
	"dall-e-3": {
		Key:          "dall-e-3",
		Manufacturer: "OpenAI",
		Type:         ModelTypeImage,
		ImageCost: map[string]float64{
			"standard-square":    0.040,
			"standard-rectangle": 0.080,
			"hd-square":          0.080,
			"hd-rectangle":       0.120,
		},
	},
	// 追加予定の Claude, Gemini なども同様に記述可
	// "claude-3-opus": {...}
	// "gemini-1.5": {...}
}
