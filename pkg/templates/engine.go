package templates

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/amorin24/llmproxy/pkg/models"
	"gopkg.in/yaml.v3"
)

type Template struct {
	Name        string            `yaml:"name"`
	Prompt      string            `yaml:"prompt"`
	Model       models.ModelType  `yaml:"model"`
	MaxTokens   int               `yaml:"max_tokens"`
	Temperature float64           `yaml:"temperature"`
	Variables   []string          `yaml:"variables"`
	Description string            `yaml:"description"`
}

type TemplateConfig struct {
	Templates map[string]Template `yaml:"templates"`
}

type Engine struct {
	templates map[string]Template
}

func NewEngine() *Engine {
	return &Engine{
		templates: make(map[string]Template),
	}
}

func (e *Engine) LoadFromYAML(yamlData []byte) error {
	var config TemplateConfig
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	for name, template := range config.Templates {
		template.Name = name
		if len(template.Variables) == 0 {
			template.Variables = extractVariables(template.Prompt)
		}
		e.templates[name] = template
	}

	return nil
}

func (e *Engine) AddTemplate(name string, template Template) {
	template.Name = name
	if len(template.Variables) == 0 {
		template.Variables = extractVariables(template.Prompt)
	}
	e.templates[name] = template
}

func (e *Engine) GetTemplate(name string) (Template, error) {
	template, exists := e.templates[name]
	if !exists {
		return Template{}, fmt.Errorf("template not found: %s", name)
	}
	return template, nil
}

func (e *Engine) ListTemplates() []string {
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}

func (e *Engine) Render(name string, variables map[string]string) (string, error) {
	template, err := e.GetTemplate(name)
	if err != nil {
		return "", err
	}

	return renderTemplate(template.Prompt, variables)
}

func (e *Engine) RenderWithDefaults(name string, variables map[string]string, defaults map[string]string) (string, error) {
	template, err := e.GetTemplate(name)
	if err != nil {
		return "", err
	}

	merged := make(map[string]string)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range variables {
		merged[k] = v
	}

	return renderTemplate(template.Prompt, merged)
}

func (e *Engine) ValidateTemplate(name string, variables map[string]string) error {
	template, err := e.GetTemplate(name)
	if err != nil {
		return err
	}

	missing := []string{}
	for _, varName := range template.Variables {
		if _, exists := variables[varName]; !exists {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func renderTemplate(template string, variables map[string]string) (string, error) {
	result := template

	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	unreplaced := extractVariables(result)
	if len(unreplaced) > 0 {
		return "", fmt.Errorf("unreplaced variables: %s", strings.Join(unreplaced, ", "))
	}

	return result, nil
}

func extractVariables(template string) []string {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	variables := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			varName := strings.TrimSpace(match[1])
			if !seen[varName] {
				variables = append(variables, varName)
				seen[varName] = true
			}
		}
	}

	return variables
}

type TemplateRequest struct {
	TemplateName string            `json:"template_name"`
	Variables    map[string]string `json:"variables"`
	Model        models.ModelType  `json:"model,omitempty"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
}

type TemplateResponse struct {
	Prompt      string           `json:"prompt"`
	Model       models.ModelType `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature float64          `json:"temperature"`
}

func (e *Engine) RenderRequest(req TemplateRequest) (*TemplateResponse, error) {
	template, err := e.GetTemplate(req.TemplateName)
	if err != nil {
		return nil, err
	}

	if err := e.ValidateTemplate(req.TemplateName, req.Variables); err != nil {
		return nil, err
	}

	prompt, err := e.Render(req.TemplateName, req.Variables)
	if err != nil {
		return nil, err
	}

	model := template.Model
	if req.Model != "" {
		model = req.Model
	}

	maxTokens := template.MaxTokens
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}

	return &TemplateResponse{
		Prompt:      prompt,
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: template.Temperature,
	}, nil
}

func DefaultTemplates() *Engine {
	engine := NewEngine()

	engine.AddTemplate("summarize", Template{
		Prompt:      "Summarize the following text:\n\n{{text}}\n\nSummary:",
		Model:       models.Claude,
		MaxTokens:   200,
		Temperature: 0.3,
		Description: "Summarize a given text",
	})

	engine.AddTemplate("translate", Template{
		Prompt:      "Translate the following text from {{source_lang}} to {{target_lang}}:\n\n{{text}}\n\nTranslation:",
		Model:       models.OpenAI,
		MaxTokens:   500,
		Temperature: 0.3,
		Description: "Translate text between languages",
	})

	engine.AddTemplate("explain_code", Template{
		Prompt:      "Explain the following {{language}} code:\n\n```{{language}}\n{{code}}\n```\n\nExplanation:",
		Model:       models.OpenAI,
		MaxTokens:   300,
		Temperature: 0.5,
		Description: "Explain code in a given programming language",
	})

	engine.AddTemplate("qa", Template{
		Prompt:      "Context: {{context}}\n\nQuestion: {{question}}\n\nAnswer:",
		Model:       models.Gemini,
		MaxTokens:   150,
		Temperature: 0.7,
		Description: "Answer questions based on context",
	})

	engine.AddTemplate("creative_writing", Template{
		Prompt:      "Write a {{genre}} story about {{topic}}. Style: {{style}}.\n\nStory:",
		Model:       models.Claude,
		MaxTokens:   1000,
		Temperature: 0.9,
		Description: "Generate creative writing",
	})

	engine.AddTemplate("email", Template{
		Prompt:      "Write a {{tone}} email to {{recipient}} about {{subject}}.\n\nEmail:",
		Model:       models.OpenAI,
		MaxTokens:   300,
		Temperature: 0.7,
		Description: "Generate professional emails",
	})

	engine.AddTemplate("extract_data", Template{
		Prompt:      "Extract {{data_type}} from the following text:\n\n{{text}}\n\nExtracted {{data_type}}:",
		Model:       models.Gemini,
		MaxTokens:   200,
		Temperature: 0.2,
		Description: "Extract structured data from text",
	})

	engine.AddTemplate("sentiment", Template{
		Prompt:      "Analyze the sentiment of the following text and classify it as positive, negative, or neutral:\n\n{{text}}\n\nSentiment:",
		Model:       models.Mistral,
		MaxTokens:   50,
		Temperature: 0.1,
		Description: "Analyze text sentiment",
	})

	return engine
}
