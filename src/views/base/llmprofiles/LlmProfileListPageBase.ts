import { ref, computed } from 'vue';

import { useLlmProfilesStore } from '@/stores/llmProfile.ts';

import { DEFAULT_LLM_PROFILE_ID, LlmProfile } from '@/models/llm_profile.ts';

import { getLlmProviderDisplayName } from '@/lib/llm_profile.ts';

export function useLlmProfileListPageBase() {
    const llmProfilesStore = useLlmProfilesStore();

    const loading = ref<boolean>(true);
    const loadingError = ref<unknown>(null);

    // the virtual default entry (id "0") is built by the server as the first item of the list
    const allProfilesWithDefault = computed<LlmProfile[]>(() => llmProfilesStore.allLlmProfiles);

    const noAvailableUserProfile = computed<boolean>(() => {
        for (const profile of llmProfilesStore.allLlmProfiles) {
            if (!profile.isDefault) {
                return false;
            }
        }

        return true;
    });

    function isDefaultProfile(profile: LlmProfile): boolean {
        return profile.isDefault;
    }

    function getProviderDisplayName(profile: LlmProfile): string {
        if (!profile.provider) {
            return '';
        }

        return getLlmProviderDisplayName(profile.provider);
    }

    function getActiveRadioValue(profile: LlmProfile): string {
        return profile.isDefault ? DEFAULT_LLM_PROFILE_ID : profile.id;
    }

    function reload(force: boolean): Promise<LlmProfile[]> {
        return llmProfilesStore.loadAllProfiles({ force });
    }

    return {
        // dependent stores
        llmProfilesStore,
        // states
        loading,
        loadingError,
        // computed states
        allProfilesWithDefault,
        noAvailableUserProfile,
        // functions
        isDefaultProfile,
        getProviderDisplayName,
        getActiveRadioValue,
        reload
    };
}
