// fork-local: provider field metadata for llm connection profiles.
// NOTE: keep in sync with the provider-specific fields of settings.LLMConfig (pkg/settings/setting.go)
// and validateCustomLlmProfileFields / buildLLMConfigFromCustomProfile (pkg/api/custom_llm_profiles.go)

export const ALL_LLM_PROVIDERS = [
    'openai',
    'openai_compatible',
    'anthropic',
    'anthropic_compatible',
    'openrouter',
    'ollama',
    'lm_studio',
    'google_ai'
] as const;

export type LlmProvider = typeof ALL_LLM_PROVIDERS[number];

export type LlmProfileFieldName = 'baseUrl' | 'apiKey' | 'apiVersion' | 'modelId' | 'maxTokens';

export interface LlmProfileFieldDefinition {
    readonly field: LlmProfileFieldName;
    readonly type: 'text' | 'password' | 'number';
    readonly required: boolean; // for secret fields "required" means required when creating or when no key is stored yet
    readonly secret: boolean;
    readonly labelKey: string;
    readonly blankMessageKey: string;
    readonly placeholderKey?: string;
    readonly placeholderText?: string; // literal placeholder (e.g. an example url), not localized
}

// provider display names are proper nouns and intentionally not localized
export const LLM_PROVIDER_DISPLAY_NAMES: Record<LlmProvider, string> = {
    'openai': 'OpenAI',
    'openai_compatible': 'OpenAI Compatible',
    'anthropic': 'Anthropic',
    'anthropic_compatible': 'Anthropic Compatible',
    'openrouter': 'OpenRouter',
    'ollama': 'Ollama',
    'lm_studio': 'LM Studio',
    'google_ai': 'Google AI'
};

const BASE_URL_FIELD: LlmProfileFieldDefinition = {
    field: 'baseUrl',
    type: 'text',
    required: true,
    secret: false,
    labelKey: 'Base URL',
    blankMessageKey: 'Base URL cannot be blank'
};

const SERVER_URL_FIELD: LlmProfileFieldDefinition = {
    field: 'baseUrl',
    type: 'text',
    required: true,
    secret: false,
    labelKey: 'Server URL',
    blankMessageKey: 'Server URL cannot be blank'
};

const API_KEY_FIELD: LlmProfileFieldDefinition = {
    field: 'apiKey',
    type: 'password',
    required: true,
    secret: true,
    labelKey: 'API Key',
    blankMessageKey: 'API key cannot be blank',
    placeholderKey: 'Your API key'
};

const OPTIONAL_API_KEY_FIELD: LlmProfileFieldDefinition = {
    ...API_KEY_FIELD,
    required: false
};

const OPTIONAL_TOKEN_FIELD: LlmProfileFieldDefinition = {
    field: 'apiKey',
    type: 'password',
    required: false,
    secret: true,
    labelKey: 'Token',
    blankMessageKey: 'Token cannot be blank',
    placeholderKey: 'Your token'
};

const API_VERSION_FIELD: LlmProfileFieldDefinition = {
    field: 'apiVersion',
    type: 'text',
    required: false,
    secret: false,
    labelKey: 'API Version',
    blankMessageKey: 'API version cannot be blank',
    placeholderText: '2023-06-01'
};

const MODEL_ID_FIELD: LlmProfileFieldDefinition = {
    field: 'modelId',
    type: 'text',
    required: true,
    secret: false,
    labelKey: 'Model ID',
    blankMessageKey: 'Model ID cannot be blank',
    placeholderKey: 'Your model ID'
};

const MAX_TOKENS_FIELD: LlmProfileFieldDefinition = {
    field: 'maxTokens',
    type: 'number',
    required: false,
    secret: false,
    labelKey: 'Max Tokens',
    blankMessageKey: 'Max tokens is invalid',
    placeholderText: '1024'
};

const LLM_PROVIDER_FIELDS: Record<LlmProvider, LlmProfileFieldDefinition[]> = {
    'openai': [
        API_KEY_FIELD,
        MODEL_ID_FIELD
    ],
    'openai_compatible': [
        { ...BASE_URL_FIELD, placeholderText: 'https://api.openai.com/v1/' },
        API_KEY_FIELD,
        MODEL_ID_FIELD
    ],
    'anthropic': [
        API_KEY_FIELD,
        MODEL_ID_FIELD,
        MAX_TOKENS_FIELD
    ],
    'anthropic_compatible': [
        { ...BASE_URL_FIELD, placeholderText: 'https://api.anthropic.com/v1/' },
        API_VERSION_FIELD,
        OPTIONAL_API_KEY_FIELD,
        MODEL_ID_FIELD,
        MAX_TOKENS_FIELD
    ],
    'openrouter': [
        API_KEY_FIELD,
        MODEL_ID_FIELD
    ],
    'ollama': [
        { ...SERVER_URL_FIELD, placeholderText: 'http://127.0.0.1:11434/' },
        MODEL_ID_FIELD
    ],
    'lm_studio': [
        { ...SERVER_URL_FIELD, placeholderText: 'http://127.0.0.1:1234/' },
        OPTIONAL_TOKEN_FIELD,
        MODEL_ID_FIELD
    ],
    'google_ai': [
        API_KEY_FIELD,
        MODEL_ID_FIELD
    ]
};

export interface LlmProviderSelectionItem {
    readonly value: LlmProvider;
    readonly name: string;
}

export interface LlmProfileValidatableInput {
    readonly name: string;
    readonly provider: string;
    readonly baseUrl: string;
    readonly apiKey: string;
    readonly apiVersion: string;
    readonly modelId: string;
    readonly maxTokens: string;
}

export function isValidLlmProvider(provider: string): provider is LlmProvider {
    return (ALL_LLM_PROVIDERS as readonly string[]).indexOf(provider) >= 0;
}

export function getLlmProviderFields(provider: string): LlmProfileFieldDefinition[] {
    if (!isValidLlmProvider(provider)) {
        return [];
    }

    return LLM_PROVIDER_FIELDS[provider];
}

export function getLlmProviderDisplayName(provider: string): string {
    if (!isValidLlmProvider(provider)) {
        return provider;
    }

    return LLM_PROVIDER_DISPLAY_NAMES[provider];
}

export function getAllLlmProviderSelectionItems(): LlmProviderSelectionItem[] {
    return ALL_LLM_PROVIDERS.map(provider => ({
        value: provider,
        name: LLM_PROVIDER_DISPLAY_NAMES[provider]
    }));
}

// connection test result types returned by the server (pkg/models/custom_llm_profile.go)
const LLM_PROFILE_TEST_RESULT_MESSAGE_KEYS: Record<string, string> = {
    'invalid_config': 'The connection settings are invalid',
    'decrypt_failed': 'Unable to decrypt the stored API key',
    'request_failed': 'The LLM request failed',
    'vision_check_failed': 'The model could not read the test image'
};

export function getLlmProfileTestResultMessageKey(result: string): string {
    return LLM_PROFILE_TEST_RESULT_MESSAGE_KEYS[result] || 'The LLM request failed';
}

export function validateLlmProfileInput(input: LlmProfileValidatableInput, options: { isNew: boolean, apiKeyConfigured: boolean }): string | null {
    if (!input.name) {
        return 'Connection name cannot be blank';
    }

    if (!isValidLlmProvider(input.provider)) {
        return 'Connection provider is invalid';
    }

    const fieldDefinitions = LLM_PROVIDER_FIELDS[input.provider];

    for (const fieldDefinition of fieldDefinitions) {
        const value = input[fieldDefinition.field];

        if (fieldDefinition.required) {
            if (fieldDefinition.secret) {
                if (!value && (options.isNew || !options.apiKeyConfigured)) {
                    return fieldDefinition.blankMessageKey;
                }
            } else if (!value) {
                return fieldDefinition.blankMessageKey;
            }
        }

        if (fieldDefinition.field === 'maxTokens' && value && !/^[1-9][0-9]*$/.test(value)) {
            return 'Max tokens is invalid';
        }
    }

    return null;
}
