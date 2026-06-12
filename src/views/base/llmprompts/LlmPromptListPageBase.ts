import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useLlmPromptsStore } from '@/stores/llmPrompt.ts';

import { DEFAULT_LLM_PROMPT_ID, LlmPrompt } from '@/models/llm_prompt.ts';

export function useLlmPromptListPageBase() {
    const { tt } = useI18n();

    const llmPromptsStore = useLlmPromptsStore();

    const loading = ref<boolean>(true);
    const loadingError = ref<unknown>(null);

    const defaultVirtualPrompt = computed<LlmPrompt>(() => {
        const prompt = LlmPrompt.createNewPrompt(tt('Default Prompt'), llmPromptsStore.defaultPromptContent);
        prompt.id = DEFAULT_LLM_PROMPT_ID;
        prompt.isActive = llmPromptsStore.activeLlmPromptId === DEFAULT_LLM_PROMPT_ID;
        return prompt;
    });

    const allPromptsWithDefault = computed<LlmPrompt[]>(() => {
        const allPrompts: LlmPrompt[] = [];
        allPrompts.push(defaultVirtualPrompt.value);
        allPrompts.push(...llmPromptsStore.allLlmPrompts);
        return allPrompts;
    });

    const noAvailableUserPrompt = computed<boolean>(() => llmPromptsStore.allLlmPrompts.length < 1);

    function isDefaultPrompt(prompt: LlmPrompt): boolean {
        return prompt.id === DEFAULT_LLM_PROMPT_ID;
    }

    function getPromptContentForDuplicating(prompt: LlmPrompt): Promise<string> {
        if (isDefaultPrompt(prompt)) {
            return Promise.resolve(llmPromptsStore.defaultPromptContent);
        }

        return llmPromptsStore.getPrompt({ id: prompt.id }).then(loadedPrompt => loadedPrompt.content);
    }

    function reload(force: boolean): Promise<LlmPrompt[]> {
        return llmPromptsStore.loadAllPrompts({ force });
    }

    return {
        // states
        loading,
        loadingError,
        // computed states
        defaultVirtualPrompt,
        allPromptsWithDefault,
        noAvailableUserPrompt,
        // functions
        isDefaultPrompt,
        getPromptContentForDuplicating,
        reload
    };
}
