import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import {
    type LlmPromptInfoResponse,
    type LlmPromptListResponse,
    type LlmPromptPreviewResponse,
    DEFAULT_LLM_PROMPT_ID,
    LlmPrompt
} from '@/models/llm_prompt.ts';

import logger from '@/lib/logger.ts';
import services, { type ApiResponsePromise } from '@/lib/services.ts';

export const useLlmPromptsStore = defineStore('llmPrompts', () => {
    const allLlmPrompts = ref<LlmPrompt[]>([]);
    const defaultPromptContent = ref<string>('');
    const llmPromptListStateInvalid = ref<boolean>(true);

    const activeLlmPromptId = computed<string>(() => {
        for (const prompt of allLlmPrompts.value) {
            if (prompt.isActive) {
                return prompt.id;
            }
        }

        return DEFAULT_LLM_PROMPT_ID;
    });

    function updateLlmPromptListInvalidState(invalidState: boolean): void {
        llmPromptListStateInvalid.value = invalidState;
    }

    function resetLlmPrompts(): void {
        allLlmPrompts.value = [];
        defaultPromptContent.value = '';
        llmPromptListStateInvalid.value = true;
    }

    function loadAllPrompts({ force }: { force?: boolean }): Promise<LlmPrompt[]> {
        if (!force && !llmPromptListStateInvalid.value) {
            return Promise.resolve(allLlmPrompts.value);
        }

        return new Promise((resolve, reject) => {
            services.getAllLlmPrompts().then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve prompt list' });
                    return;
                }

                const listResponse: LlmPromptListResponse = data.result;

                defaultPromptContent.value = listResponse.defaultContent;
                allLlmPrompts.value = LlmPrompt.ofMulti(listResponse.prompts);

                if (llmPromptListStateInvalid.value) {
                    updateLlmPromptListInvalidState(false);
                }

                resolve(allLlmPrompts.value);
            }).catch(error => {
                logger.error('failed to load prompt list', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to retrieve prompt list' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function getPrompt({ id }: { id: string }): Promise<LlmPrompt> {
        return new Promise((resolve, reject) => {
            services.getLlmPrompt({ id }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve prompt' });
                    return;
                }

                resolve(LlmPrompt.of(data.result));
            }).catch(error => {
                logger.error('failed to load prompt info', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to retrieve prompt' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function savePrompt({ prompt }: { prompt: LlmPrompt }): Promise<LlmPrompt> {
        return new Promise((resolve, reject) => {
            let promise: ApiResponsePromise<LlmPromptInfoResponse>;

            if (!prompt.id) {
                promise = services.addLlmPrompt(prompt.toCreateRequest());
            } else {
                promise = services.modifyLlmPrompt(prompt.toModifyRequest());
            }

            promise.then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    if (!prompt.id) {
                        reject({ message: 'Unable to add prompt' });
                    } else {
                        reject({ message: 'Unable to save prompt' });
                    }
                    return;
                }

                updateLlmPromptListInvalidState(true);

                resolve(LlmPrompt.of(data.result));
            }).catch(error => {
                logger.error('failed to save prompt', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    if (!prompt.id) {
                        reject({ message: 'Unable to add prompt' });
                    } else {
                        reject({ message: 'Unable to save prompt' });
                    }
                } else {
                    reject(error);
                }
            });
        });
    }

    function deletePrompt({ prompt }: { prompt: LlmPrompt }): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.deleteLlmPrompt({ id: prompt.id }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to delete this prompt' });
                    return;
                }

                updateLlmPromptListInvalidState(true);

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to delete prompt', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to delete this prompt' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function setActivePrompt({ id }: { id: string }): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.setActiveLlmPrompt({ id }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to set active prompt' });
                    return;
                }

                for (const prompt of allLlmPrompts.value) {
                    prompt.isActive = prompt.id === id;
                }

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to set active prompt', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to set active prompt' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function getPreviewedPromptContent({ content }: { content: string }): Promise<string> {
        return new Promise((resolve, reject) => {
            services.previewLlmPrompt({ content }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to preview prompt' });
                    return;
                }

                const previewResponse: LlmPromptPreviewResponse = data.result;
                resolve(previewResponse.renderedContent);
            }).catch(error => {
                logger.error('failed to preview prompt', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to preview prompt' });
                } else {
                    reject(error);
                }
            });
        });
    }

    return {
        // states
        allLlmPrompts,
        defaultPromptContent,
        llmPromptListStateInvalid,
        // computed states
        activeLlmPromptId,
        // functions
        updateLlmPromptListInvalidState,
        resetLlmPrompts,
        loadAllPrompts,
        getPrompt,
        savePrompt,
        deletePrompt,
        setActivePrompt,
        getPreviewedPromptContent
    };
});
