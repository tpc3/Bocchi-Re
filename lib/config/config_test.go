package config

import "testing"

func TestMigrateUnavailableModels(t *testing.T) {
	tests := []struct {
		name      string
		model     Model
		want      Model
		wantChat  bool
		wantImage bool
	}{
		{
			name: "retired models",
			model: Model{
				Chat:  "gpt-5-chat-latest",
				Image: "gpt-image-1",
			},
			want: Model{
				Chat:  DefaultChatModel,
				Image: DefaultImageModel,
			},
			wantChat:  true,
			wantImage: true,
		},
		{
			name: "unknown models",
			model: Model{
				Chat:  "unknown-chat-model",
				Image: "unknown-image-model",
			},
			want: Model{
				Chat:  DefaultChatModel,
				Image: DefaultImageModel,
			},
			wantChat:  true,
			wantImage: true,
		},
		{
			name: "available models",
			model: Model{
				Chat:  "gpt-6.1-terra",
				Image: "gpt-image-2.6-sunburst",
			},
			want: Model{
				Chat:  "gpt-6.1-terra",
				Image: "gpt-image-2.6-sunburst",
			},
		},
		{
			name:  "empty models",
			model: Model{},
			want:  Model{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, chatMigrated, imageMigrated := migrateUnavailableModels(test.model)
			if got != test.want {
				t.Fatalf("migrateUnavailableModels(%+v) = %+v, want %+v", test.model, got, test.want)
			}
			if chatMigrated != test.wantChat {
				t.Fatalf("chat migration = %t, want %t", chatMigrated, test.wantChat)
			}
			if imageMigrated != test.wantImage {
				t.Fatalf("image migration = %t, want %t", imageMigrated, test.wantImage)
			}
		})
	}
}

func TestCurrentModelCatalog(t *testing.T) {
	chat, ok := AllModels["gpt-6.1-terra"]
	if !ok {
		t.Fatal("gpt-6.1-terra is missing from the model catalog")
	}
	if chat.Type != ModelTypeText || chat.InputCost != 1.0 || chat.OutputCost != 5.0 || !chat.SupportVision {
		t.Fatalf("unexpected gpt-6.1-terra metadata: %+v", chat)
	}

	for _, model := range []string{"gpt-image-2.6-flare", "gpt-image-2.6-sunburst"} {
		info, ok := AllModels[model]
		if !ok {
			t.Fatalf("%s is missing from the model catalog", model)
		}
		if info.Type != ModelTypeImage || info.InputCost != 5 || info.OutputCost != 30 {
			t.Fatalf("unexpected %s metadata: %+v", model, info)
		}
		for size, cost := range info.ImageCost {
			if cost != 0 {
				t.Fatalf("%s image size %s has cost %v, want 0", model, size, cost)
			}
		}
	}

	if DefaultChatModel != "gpt-6.1-terra" || DefaultImageModel != "gpt-image-2.6-sunburst" {
		t.Fatalf("unexpected defaults: chat=%q image=%q", DefaultChatModel, DefaultImageModel)
	}
}
