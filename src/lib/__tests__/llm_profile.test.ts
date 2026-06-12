import { describe, expect, test } from 'vitest';

import {
    ALL_LLM_PROVIDERS,
    LLM_PROVIDER_DISPLAY_NAMES,
    type LlmProfileValidatableInput,
    getAllLlmProviderSelectionItems,
    getLlmProviderDisplayName,
    getLlmProviderFields,
    isValidLlmProvider,
    validateLlmProfileInput
} from '@/lib/llm_profile.ts';

function buildInput(overrides?: Partial<LlmProfileValidatableInput>): LlmProfileValidatableInput {
    return {
        name: 'Test Connection',
        provider: 'ollama',
        baseUrl: 'http://127.0.0.1:11434/',
        apiKey: '',
        apiVersion: '',
        modelId: 'gemma4:latest',
        maxTokens: '',
        ...overrides
    };
}

describe('ALL_LLM_PROVIDERS', () => {
    test('contains all 8 upstream providers', () => {
        expect(ALL_LLM_PROVIDERS).toStrictEqual([
            'openai',
            'openai_compatible',
            'anthropic',
            'anthropic_compatible',
            'openrouter',
            'ollama',
            'lm_studio',
            'google_ai'
        ]);
    });

    test('every provider has a display name and field definitions', () => {
        for (const provider of ALL_LLM_PROVIDERS) {
            expect(LLM_PROVIDER_DISPLAY_NAMES[provider]).toBeTruthy();
            expect(getLlmProviderFields(provider).length).toBeGreaterThan(0);
        }
    });
});

describe('getLlmProviderFields', () => {
    test('returns expected field sets per provider', () => {
        const expectedFields: Record<string, string[]> = {
            'openai': ['apiKey', 'modelId'],
            'openai_compatible': ['baseUrl', 'apiKey', 'modelId'],
            'anthropic': ['apiKey', 'modelId', 'maxTokens'],
            'anthropic_compatible': ['baseUrl', 'apiVersion', 'apiKey', 'modelId', 'maxTokens'],
            'openrouter': ['apiKey', 'modelId'],
            'ollama': ['baseUrl', 'modelId'],
            'lm_studio': ['baseUrl', 'apiKey', 'modelId'],
            'google_ai': ['apiKey', 'modelId']
        };

        for (const provider of ALL_LLM_PROVIDERS) {
            expect(getLlmProviderFields(provider).map(fieldDefinition => fieldDefinition.field)).toStrictEqual(expectedFields[provider]);
        }
    });

    test('api key fields are secret password fields', () => {
        for (const provider of ALL_LLM_PROVIDERS) {
            for (const fieldDefinition of getLlmProviderFields(provider)) {
                if (fieldDefinition.field === 'apiKey') {
                    expect(fieldDefinition.secret).toBe(true);
                    expect(fieldDefinition.type).toBe('password');
                } else {
                    expect(fieldDefinition.secret).toBe(false);
                }
            }
        }
    });

    test('api key is optional only for lm_studio and anthropic_compatible', () => {
        for (const provider of ALL_LLM_PROVIDERS) {
            const apiKeyField = getLlmProviderFields(provider).find(fieldDefinition => fieldDefinition.field === 'apiKey');

            if (provider === 'ollama') {
                expect(apiKeyField).toBeUndefined();
            } else if (provider === 'lm_studio' || provider === 'anthropic_compatible') {
                expect(apiKeyField?.required).toBe(false);
            } else {
                expect(apiKeyField?.required).toBe(true);
            }
        }
    });

    test('returns empty array for unknown provider', () => {
        expect(getLlmProviderFields('unknown')).toStrictEqual([]);
        expect(getLlmProviderFields('')).toStrictEqual([]);
    });
});

describe('isValidLlmProvider / getLlmProviderDisplayName / getAllLlmProviderSelectionItems', () => {
    test('isValidLlmProvider', () => {
        expect(isValidLlmProvider('ollama')).toBe(true);
        expect(isValidLlmProvider('unknown')).toBe(false);
    });

    test('getLlmProviderDisplayName', () => {
        expect(getLlmProviderDisplayName('ollama')).toBe('Ollama');
        expect(getLlmProviderDisplayName('lm_studio')).toBe('LM Studio');
        expect(getLlmProviderDisplayName('unknown')).toBe('unknown');
    });

    test('getAllLlmProviderSelectionItems returns all providers', () => {
        const items = getAllLlmProviderSelectionItems();
        expect(items.length).toBe(ALL_LLM_PROVIDERS.length);
        expect(items[0]).toStrictEqual({ value: 'openai', name: 'OpenAI' });
    });
});

describe('validateLlmProfileInput', () => {
    const newOptions = { isNew: true, apiKeyConfigured: false };
    const editConfiguredOptions = { isNew: false, apiKeyConfigured: true };

    test('valid new ollama profile', () => {
        expect(validateLlmProfileInput(buildInput(), newOptions)).toBeNull();
    });

    test('blank name', () => {
        expect(validateLlmProfileInput(buildInput({ name: '' }), newOptions)).toBe('Connection name cannot be blank');
    });

    test('invalid provider', () => {
        expect(validateLlmProfileInput(buildInput({ provider: 'unknown' }), newOptions)).toBe('Connection provider is invalid');
    });

    test('blank model id', () => {
        expect(validateLlmProfileInput(buildInput({ modelId: '' }), newOptions)).toBe('Model ID cannot be blank');
    });

    test('blank server url for ollama', () => {
        expect(validateLlmProfileInput(buildInput({ baseUrl: '' }), newOptions)).toBe('Server URL cannot be blank');
    });

    test('blank base url for openai_compatible', () => {
        expect(validateLlmProfileInput(buildInput({ provider: 'openai_compatible', baseUrl: '', apiKey: 'key' }), newOptions)).toBe('Base URL cannot be blank');
    });

    test('new profile requires api key for openai', () => {
        expect(validateLlmProfileInput(buildInput({ provider: 'openai', apiKey: '' }), newOptions)).toBe('API key cannot be blank');
        expect(validateLlmProfileInput(buildInput({ provider: 'openai', apiKey: 'sk-xxx' }), newOptions)).toBeNull();
    });

    test('editing with configured api key allows blank api key input', () => {
        expect(validateLlmProfileInput(buildInput({ provider: 'openai', apiKey: '' }), editConfiguredOptions)).toBeNull();
    });

    test('editing without configured api key still requires api key input', () => {
        expect(validateLlmProfileInput(buildInput({ provider: 'openai', apiKey: '' }), { isNew: false, apiKeyConfigured: false })).toBe('API key cannot be blank');
    });

    test('lm_studio token is optional', () => {
        expect(validateLlmProfileInput(buildInput({ provider: 'lm_studio', baseUrl: 'http://127.0.0.1:1234/', apiKey: '' }), newOptions)).toBeNull();
    });

    test('max tokens must be a positive integer when filled', () => {
        expect(validateLlmProfileInput(buildInput({ provider: 'anthropic', apiKey: 'key', maxTokens: '' }), newOptions)).toBeNull();
        expect(validateLlmProfileInput(buildInput({ provider: 'anthropic', apiKey: 'key', maxTokens: '2048' }), newOptions)).toBeNull();
        expect(validateLlmProfileInput(buildInput({ provider: 'anthropic', apiKey: 'key', maxTokens: 'abc' }), newOptions)).toBe('Max tokens is invalid');
        expect(validateLlmProfileInput(buildInput({ provider: 'anthropic', apiKey: 'key', maxTokens: '0' }), newOptions)).toBe('Max tokens is invalid');
        expect(validateLlmProfileInput(buildInput({ provider: 'anthropic', apiKey: 'key', maxTokens: '-1' }), newOptions)).toBe('Max tokens is invalid');
    });
});
