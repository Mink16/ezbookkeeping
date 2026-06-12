import { type LlmProfileFieldName, getLlmProviderFields } from '@/lib/llm_profile.ts';

export const DEFAULT_LLM_PROFILE_ID: string = '0';

export class LlmProfile {
    public id: string;
    public name: string;
    public provider: string;
    public baseUrl: string;
    public apiKey: string; // input-only, the stored key is never returned by the server
    public apiVersion: string;
    public modelId: string;
    public maxTokens: string;
    public isActive: boolean;
    public apiKeyConfigured: boolean;
    public isDefault: boolean;
    public editable: boolean;

    private constructor(id: string, name: string, provider: string, baseUrl: string, apiVersion: string, modelId: string, maxTokens: string, isActive: boolean, apiKeyConfigured: boolean, isDefault: boolean, editable: boolean) {
        this.id = id;
        this.name = name;
        this.provider = provider;
        this.baseUrl = baseUrl;
        this.apiKey = '';
        this.apiVersion = apiVersion;
        this.modelId = modelId;
        this.maxTokens = maxTokens;
        this.isActive = isActive;
        this.apiKeyConfigured = apiKeyConfigured;
        this.isDefault = isDefault;
        this.editable = editable;
    }

    public toCreateRequest(): LlmProfileCreateRequest {
        return {
            name: this.name,
            provider: this.provider,
            ...buildProviderFields(this)
        };
    }

    public toModifyRequest(): LlmProfileModifyRequest {
        return {
            id: this.id,
            name: this.name,
            provider: this.provider,
            ...buildProviderFields(this)
        };
    }

    public toTestRequest(): LlmProfileTestRequest {
        return {
            id: this.id || undefined,
            provider: this.provider,
            ...buildProviderFields(this)
        };
    }

    public clone(): LlmProfile {
        const profile = new LlmProfile(this.id, this.name, this.provider, this.baseUrl, this.apiVersion, this.modelId, this.maxTokens, this.isActive, this.apiKeyConfigured, this.isDefault, this.editable);
        profile.apiKey = this.apiKey;
        return profile;
    }

    public static of(profileResponse: LlmProfileInfoResponse): LlmProfile {
        return new LlmProfile(
            profileResponse.id,
            profileResponse.name,
            profileResponse.provider,
            profileResponse.baseUrl || '',
            profileResponse.apiVersion || '',
            profileResponse.modelId || '',
            profileResponse.maxTokens ? profileResponse.maxTokens.toString() : '',
            profileResponse.isActive,
            profileResponse.hasApiKey,
            profileResponse.isDefault,
            profileResponse.editable
        );
    }

    public static ofMulti(profileResponses: LlmProfileInfoResponse[]): LlmProfile[] {
        const profiles: LlmProfile[] = [];

        for (const profileResponse of profileResponses) {
            profiles.push(LlmProfile.of(profileResponse));
        }

        return profiles;
    }

    public static createNewProfile(): LlmProfile {
        return new LlmProfile('', '', 'openai', '', '', '', '', false, false, false, true);
    }
}

// only the fields defined for the current provider are submitted, so that stale values
// entered before switching the provider can never leak into the request.
// this is a module-level function instead of a private method because vue's UnwrapRef
// type is structurally incompatible with classes having private methods
function buildProviderFields(profile: LlmProfile): { baseUrl: string, apiKey: string, apiVersion: string, modelId: string, maxTokens: number } {
    const fields = {
        baseUrl: '',
        apiKey: '',
        apiVersion: '',
        modelId: '',
        maxTokens: 0
    };

    for (const fieldDefinition of getLlmProviderFields(profile.provider)) {
        const field: LlmProfileFieldName = fieldDefinition.field;

        if (field === 'maxTokens') {
            fields.maxTokens = profile.maxTokens ? parseInt(profile.maxTokens, 10) || 0 : 0;
        } else {
            fields[field] = profile[field];
        }
    }

    return fields;
}

export interface LlmProfileCreateRequest {
    readonly name: string;
    readonly provider: string;
    readonly baseUrl: string;
    readonly apiKey: string;
    readonly apiVersion: string;
    readonly modelId: string;
    readonly maxTokens: number;
}

export interface LlmProfileModifyRequest {
    readonly id: string;
    readonly name: string;
    readonly provider: string;
    readonly baseUrl: string;
    readonly apiKey: string;
    readonly apiVersion: string;
    readonly modelId: string;
    readonly maxTokens: number;
}

export interface LlmProfileDeleteRequest {
    readonly id: string;
}

export interface LlmProfileSetActiveRequest {
    readonly id: string;
}

export interface LlmProfileTestRequest {
    readonly id?: string;
    readonly provider: string;
    readonly baseUrl: string;
    readonly apiKey: string;
    readonly apiVersion: string;
    readonly modelId: string;
    readonly maxTokens: number;
}

export interface LlmProfileInfoResponse {
    readonly id: string;
    readonly name: string;
    readonly provider: string;
    readonly baseUrl?: string;
    readonly hasApiKey: boolean;
    readonly apiVersion?: string;
    readonly modelId: string;
    readonly maxTokens?: number;
    readonly isActive: boolean;
    readonly isDefault: boolean;
    readonly editable: boolean;
    readonly createdTime?: number;
    readonly updatedTime?: number;
}

export interface LlmProfileListResponse {
    readonly profiles: LlmProfileInfoResponse[];
}

export interface LlmProfileTestResponse {
    readonly success: boolean;
    readonly result: string;
    readonly recognizedText?: string;
}
