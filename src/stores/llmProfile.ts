import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import {
    type LlmProfileInfoResponse,
    type LlmProfileListResponse,
    type LlmProfileTestResponse,
    DEFAULT_LLM_PROFILE_ID,
    LlmProfile
} from '@/models/llm_profile.ts';

import logger from '@/lib/logger.ts';
import services, { type ApiResponsePromise } from '@/lib/services.ts';

export const useLlmProfilesStore = defineStore('llmProfiles', () => {
    const allLlmProfiles = ref<LlmProfile[]>([]);
    const llmProfileListStateInvalid = ref<boolean>(true);

    const activeLlmProfileId = computed<string>(() => {
        for (const profile of allLlmProfiles.value) {
            if (!profile.isDefault && profile.isActive) {
                return profile.id;
            }
        }

        return DEFAULT_LLM_PROFILE_ID;
    });

    function updateLlmProfileListInvalidState(invalidState: boolean): void {
        llmProfileListStateInvalid.value = invalidState;
    }

    function resetLlmProfiles(): void {
        allLlmProfiles.value = [];
        llmProfileListStateInvalid.value = true;
    }

    function loadAllProfiles({ force }: { force?: boolean }): Promise<LlmProfile[]> {
        if (!force && !llmProfileListStateInvalid.value) {
            return Promise.resolve(allLlmProfiles.value);
        }

        return new Promise((resolve, reject) => {
            services.getAllLlmProfiles().then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve connection list' });
                    return;
                }

                const listResponse: LlmProfileListResponse = data.result;

                allLlmProfiles.value = LlmProfile.ofMulti(listResponse.profiles);

                if (llmProfileListStateInvalid.value) {
                    updateLlmProfileListInvalidState(false);
                }

                resolve(allLlmProfiles.value);
            }).catch(error => {
                logger.error('failed to load llm connection profile list', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to retrieve connection list' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function getProfile({ id }: { id: string }): Promise<LlmProfile> {
        return new Promise((resolve, reject) => {
            services.getLlmProfile({ id }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve connection' });
                    return;
                }

                resolve(LlmProfile.of(data.result));
            }).catch(error => {
                logger.error('failed to load llm connection profile info', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to retrieve connection' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function saveProfile({ profile }: { profile: LlmProfile }): Promise<LlmProfile> {
        return new Promise((resolve, reject) => {
            let promise: ApiResponsePromise<LlmProfileInfoResponse>;

            if (!profile.id) {
                promise = services.addLlmProfile(profile.toCreateRequest());
            } else {
                promise = services.modifyLlmProfile(profile.toModifyRequest());
            }

            promise.then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    if (!profile.id) {
                        reject({ message: 'Unable to add connection' });
                    } else {
                        reject({ message: 'Unable to save connection' });
                    }
                    return;
                }

                updateLlmProfileListInvalidState(true);

                resolve(LlmProfile.of(data.result));
            }).catch(error => {
                logger.error('failed to save llm connection profile', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    if (!profile.id) {
                        reject({ message: 'Unable to add connection' });
                    } else {
                        reject({ message: 'Unable to save connection' });
                    }
                } else {
                    reject(error);
                }
            });
        });
    }

    function deleteProfile({ profile }: { profile: LlmProfile }): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.deleteLlmProfile({ id: profile.id }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to delete this connection' });
                    return;
                }

                updateLlmProfileListInvalidState(true);

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to delete llm connection profile', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to delete this connection' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function setActiveProfile({ id }: { id: string }): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.setActiveLlmProfile({ id }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to set active connection' });
                    return;
                }

                for (const profile of allLlmProfiles.value) {
                    if (profile.isDefault) {
                        profile.isActive = id === DEFAULT_LLM_PROFILE_ID;
                    } else {
                        profile.isActive = profile.id === id;
                    }
                }

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to set active llm connection profile', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to set active connection' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function testProfile({ profile }: { profile: LlmProfile }): Promise<LlmProfileTestResponse> {
        return new Promise((resolve, reject) => {
            services.testLlmProfile(profile.toTestRequest()).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to test connection' });
                    return;
                }

                // the test result itself (success or failure) is resolved, callers display it
                resolve(data.result);
            }).catch(error => {
                logger.error('failed to test llm connection profile', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to test connection' });
                } else {
                    reject(error);
                }
            });
        });
    }

    return {
        // states
        allLlmProfiles,
        llmProfileListStateInvalid,
        // computed states
        activeLlmProfileId,
        // functions
        updateLlmProfileListInvalidState,
        resetLlmProfiles,
        loadAllProfiles,
        getProfile,
        saveProfile,
        deleteProfile,
        setActiveProfile,
        testProfile
    };
});
