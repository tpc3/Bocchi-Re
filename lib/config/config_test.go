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
				Chat:  "gpt-6-sol",
				Image: "gpt-image-2",
			},
			want: Model{
				Chat:  "gpt-6-sol",
				Image: "gpt-image-2",
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
