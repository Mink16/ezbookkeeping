import { ref, computed } from 'vue';

import { useLlmProfilesStore } from '@/stores/llmProfile.ts';

import { LlmProfile, type LlmProfileTestResponse } from '@/models/llm_profile.ts';

import {
    type LlmProfileFieldDefinition,
    type LlmProviderSelectionItem,
    getAllLlmProviderSelectionItems,
    getLlmProviderFields,
    validateLlmProfileInput
} from '@/lib/llm_profile.ts';

export function useLlmProfileEditPageBase() {
    const llmProfilesStore = useLlmProfilesStore();

    const profile = ref<LlmProfile>(LlmProfile.createNewProfile());
    const originalProfile = ref<LlmProfile | null>(null);
    const loading = ref<boolean>(false);
    const submitting = ref<boolean>(false);
    const testing = ref<boolean>(false);
    const testResult = ref<LlmProfileTestResponse | null>(null);

    const isNewProfile = computed<boolean>(() => !profile.value.id);
    const title = computed<string>(() => isNewProfile.value ? 'Add Connection' : 'Edit Connection');

    const allProviderItems: LlmProviderSelectionItem[] = getAllLlmProviderSelectionItems();

    // the single source of the provider-specific field visibility, both desktop and mobile render this with v-for
    const visibleFields = computed<LlmProfileFieldDefinition[]>(() => getLlmProviderFields(profile.value.provider));

    const inputIsNotChanged = computed<boolean>(() => {
        const original = originalProfile.value;

        if (!original) {
            return false;
        }

        return profile.value.name === original.name
            && profile.value.provider === original.provider
            && profile.value.baseUrl === original.baseUrl
            && profile.value.apiKey === ''
            && profile.value.apiVersion === original.apiVersion
            && profile.value.modelId === original.modelId
            && profile.value.maxTokens === original.maxTokens;
    });

    const inputInvalidProblemMessage = computed<string | null>(() => validateLlmProfileInput({
        name: profile.value.name,
        provider: profile.value.provider,
        baseUrl: profile.value.baseUrl,
        apiKey: profile.value.apiKey,
        apiVersion: profile.value.apiVersion,
        modelId: profile.value.modelId,
        maxTokens: profile.value.maxTokens
    }, {
        isNew: isNewProfile.value,
        apiKeyConfigured: profile.value.apiKeyConfigured
    }));

    const inputIsNotChangedOrInvalid = computed<boolean>(() => inputIsNotChanged.value || !!inputInvalidProblemMessage.value);

    const canTest = computed<boolean>(() => !inputInvalidProblemMessage.value);

    function getFieldPlaceholder(fieldDefinition: LlmProfileFieldDefinition, tt: (key: string) => string): string {
        if (fieldDefinition.secret && !isNewProfile.value && profile.value.apiKeyConfigured) {
            return tt('API key is configured, leave blank to keep it unchanged');
        }

        if (fieldDefinition.placeholderText) {
            return fieldDefinition.placeholderText;
        }

        if (fieldDefinition.placeholderKey) {
            return tt(fieldDefinition.placeholderKey);
        }

        return '';
    }

    function setLoadedProfile(loadedProfile: LlmProfile): void {
        profile.value = loadedProfile;
        originalProfile.value = loadedProfile.clone();
        testResult.value = null;
    }

    function setNewProfile(): void {
        profile.value = LlmProfile.createNewProfile();
        originalProfile.value = null;
        testResult.value = null;
    }

    function onProviderChanged(): void {
        testResult.value = null;
    }

    function testConnection(): Promise<LlmProfileTestResponse> {
        testing.value = true;
        testResult.value = null;

        return llmProfilesStore.testProfile({ profile: profile.value }).then(result => {
            testing.value = false;
            testResult.value = result;
            return result;
        }).catch(error => {
            testing.value = false;
            throw error;
        });
    }

    return {
        // dependent stores
        llmProfilesStore,
        // states
        profile,
        originalProfile,
        loading,
        submitting,
        testing,
        testResult,
        // constants
        allProviderItems,
        // computed states
        isNewProfile,
        title,
        visibleFields,
        inputIsNotChanged,
        inputInvalidProblemMessage,
        inputIsNotChangedOrInvalid,
        canTest,
        // functions
        getFieldPlaceholder,
        setLoadedProfile,
        setNewProfile,
        onProviderChanged,
        testConnection
    };
}
