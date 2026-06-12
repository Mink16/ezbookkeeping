import { ref, computed } from 'vue';

import { useLlmPromptsStore } from '@/stores/llmPrompt.ts';

import { LlmPrompt } from '@/models/llm_prompt.ts';

import { type LlmPromptPlaceholderHint, getAllLlmPromptPlaceholderHints, getMissingLlmPromptPlaceholders } from '@/lib/llm_prompt.ts';

export function useLlmPromptEditPageBase() {
    const llmPromptsStore = useLlmPromptsStore();

    const prompt = ref<LlmPrompt>(LlmPrompt.createNewPrompt());
    const originalName = ref<string>('');
    const originalContent = ref<string>('');
    const loading = ref<boolean>(false);
    const submitting = ref<boolean>(false);
    const previewing = ref<boolean>(false);
    const showPreview = ref<boolean>(false);
    const previewedContent = ref<string>('');

    const isNewPrompt = computed<boolean>(() => !prompt.value.id);
    const title = computed<string>(() => isNewPrompt.value ? 'Add Prompt' : 'Edit Prompt');

    const allPlaceholderHints: LlmPromptPlaceholderHint[] = getAllLlmPromptPlaceholderHints();

    const missingPlaceholders = computed<string[]>(() => {
        if (!prompt.value.content) {
            return [];
        }

        return getMissingLlmPromptPlaceholders(prompt.value.content);
    });

    const inputIsNotChanged = computed<boolean>(() => prompt.value.name === originalName.value && prompt.value.content === originalContent.value);

    const inputEmptyProblemMessage = computed<string | null>(() => {
        if (!prompt.value.name) {
            return 'Prompt name cannot be blank';
        } else if (!prompt.value.content) {
            return 'Prompt content cannot be blank';
        } else {
            return null;
        }
    });

    const inputIsNotChangedOrInvalid = computed<boolean>(() => inputIsNotChanged.value || !!inputEmptyProblemMessage.value);

    function setLoadedPrompt(loadedPrompt: LlmPrompt): void {
        prompt.value = loadedPrompt;
        originalName.value = loadedPrompt.name;
        originalContent.value = loadedPrompt.content;
    }

    function setNewPrompt(duplicateFromContent?: string): void {
        prompt.value = LlmPrompt.createNewPrompt('', duplicateFromContent || '');
        originalName.value = '';
        originalContent.value = '';
    }

    function loadPreview(): Promise<string> {
        previewing.value = true;

        return llmPromptsStore.getPreviewedPromptContent({ content: prompt.value.content }).then(renderedContent => {
            previewing.value = false;
            previewedContent.value = renderedContent;
            showPreview.value = true;
            return renderedContent;
        }).catch(error => {
            previewing.value = false;
            throw error;
        });
    }

    return {
        // dependent stores
        llmPromptsStore,
        // states
        prompt,
        originalName,
        originalContent,
        loading,
        submitting,
        previewing,
        showPreview,
        previewedContent,
        // constants
        allPlaceholderHints,
        // computed states
        isNewPrompt,
        title,
        missingPlaceholders,
        inputIsNotChanged,
        inputEmptyProblemMessage,
        inputIsNotChangedOrInvalid,
        // functions
        setLoadedPrompt,
        setNewPrompt,
        loadPreview
    };
}
