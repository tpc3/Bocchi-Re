package cmds

import (
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func TestGPT6ReasoningSupport(t *testing.T) {
	tests := []struct {
		model          string
		supportsNone   bool
		supportsXHigh  bool
		supportsMax    bool
		rejectsMinimal bool
		skipSampling   bool
	}{
		{
			model:          "gpt-6-astra",
			supportsXHigh:  true,
			supportsMax:    true,
			rejectsMinimal: true,
			skipSampling:   true,
		},
		{
			model:          "gpt-6.1-sol",
			supportsXHigh:  true,
			supportsMax:    true,
			rejectsMinimal: true,
			skipSampling:   true,
		},
		{
			model:          "gpt-6-sol",
			supportsNone:   true,
			supportsXHigh:  true,
			supportsMax:    true,
			rejectsMinimal: true,
			skipSampling:   true,
		},
		{
			model:          "gpt-6-luna",
			supportsNone:   true,
			supportsXHigh:  true,
			supportsMax:    true,
			rejectsMinimal: true,
			skipSampling:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			if got := supportsNoReasoning(test.model); got != test.supportsNone {
				t.Fatalf("supportsNoReasoning(%q) = %t, want %t", test.model, got, test.supportsNone)
			}
			if got := supportsXHighReasoning(test.model); got != test.supportsXHigh {
				t.Fatalf("supportsXHighReasoning(%q) = %t, want %t", test.model, got, test.supportsXHigh)
			}
			if got := supportsMaxReasoning(test.model); got != test.supportsMax {
				t.Fatalf("supportsMaxReasoning(%q) = %t, want %t", test.model, got, test.supportsMax)
			}
			if got := rejectsMinimalReasoning(test.model); got != test.rejectsMinimal {
				t.Fatalf("rejectsMinimalReasoning(%q) = %t, want %t", test.model, got, test.rejectsMinimal)
			}
			if got := skipSamplingParams(test.model, "medium"); got != test.skipSampling {
				t.Fatalf("skipSamplingParams(%q, medium) = %t, want %t", test.model, got, test.skipSampling)
			}
			if got := skipSamplingParams(test.model, "none"); got {
				t.Fatalf("skipSamplingParams(%q, none) = true, want false", test.model)
			}
		})
	}
}

func TestGPT56MinimalReasoningIsRejected(t *testing.T) {
	if !rejectsMinimalReasoning("gpt-5.6") || !rejectsMinimalReasoning("gpt-5.6-terra") {
		t.Fatal("gpt-5.6 models must reject minimal reasoning")
	}
	if rejectsMinimalReasoning("gpt-5") {
		t.Fatal("existing gpt-5 minimal reasoning behavior should remain unchanged")
	}
}

func TestGPT6ReasoningModelsAreRecognized(t *testing.T) {
	for _, model := range []string{"gpt-6-astra", "gpt-6.1-sol", "gpt-6-sol", "gpt-6-luna"} {
		if !reasoningModelRegex.MatchString(model) {
			t.Errorf("reasoningModelRegex does not recognize %q", model)
		}
	}
	if reasoningModelRegex.MatchString("gpt-4o") {
		t.Error("reasoningModelRegex unexpectedly recognizes gpt-4o")
	}
}

func TestClearSamplingParamsOmitsValues(t *testing.T) {
	request := responses.ResponseNewParams{
		Temperature: openai.Float(0.7),
		TopP:        openai.Float(0.8),
	}

	clearSamplingParams(&request)

	if request.Temperature.Valid() {
		t.Error("temperature should be omitted")
	}
	if request.TopP.Valid() {
		t.Error("top_p should be omitted")
	}
}
