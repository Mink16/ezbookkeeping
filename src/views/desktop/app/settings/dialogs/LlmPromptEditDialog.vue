<template>
    <v-dialog width="800" :persistent="true" v-model="showState">
        <v-card class="pa-2 pa-sm-4 pa-md-4">
            <template #title>
                <div class="d-flex align-center justify-center">
                    <h4 class="text-h4">{{ tt(title) }}</h4>
                </div>
            </template>
            <v-card-text class="my-md-4 w-100 d-flex justify-center">
                <v-form class="w-100">
                    <v-row>
                        <v-col cols="12">
                            <v-text-field
                                type="text"
                                persistent-placeholder
                                :disabled="loading || submitting"
                                :label="tt('Prompt Name')"
                                :placeholder="tt('Your prompt name')"
                                v-model="prompt.name"
                            />
                        </v-col>
                        <v-col cols="12">
                            <v-textarea
                                class="llm-prompt-content-input"
                                persistent-placeholder
                                rows="12"
                                :disabled="loading || submitting"
                                :label="tt('Prompt Content')"
                                :placeholder="tt('Your prompt content')"
                                v-model="prompt.content"
                            />
                        </v-col>
                        <v-col cols="12">
                            <div class="text-caption">
                                <div class="font-weight-bold mb-1">{{ tt('Available Placeholders') }}</div>
                                <div :key="hint.placeholder" v-for="hint in allPlaceholderHints">
                                    <code>{{ hint.placeholder }}</code><span> — {{ tt(hint.descriptionKey) }}</span>
                                </div>
                            </div>
                        </v-col>
                        <v-col cols="12" v-if="missingPlaceholders.length > 0">
                            <v-alert type="warning" variant="tonal" density="compact">
                                {{ tt('format.misc.llmPromptMissingPlaceholders', { placeholders: missingPlaceholders.join(', ') }) }}
                            </v-alert>
                        </v-col>
                        <v-col cols="12" v-if="showPreview">
                            <v-textarea
                                class="llm-prompt-content-input"
                                persistent-placeholder
                                readonly
                                rows="12"
                                :label="tt('Prompt Preview')"
                                v-model="previewedContent"
                            />
                        </v-col>
                    </v-row>
                </v-form>
            </v-card-text>
            <v-card-text class="overflow-y-visible">
                <div class="w-100 d-flex justify-center gap-4">
                    <v-btn color="secondary" variant="tonal" :disabled="loading || submitting || previewing || !prompt.content"
                           @click="preview">
                        {{ tt('Preview Final Prompt') }}
                        <v-progress-circular indeterminate size="22" class="ms-2" v-if="previewing"></v-progress-circular>
                    </v-btn>
                    <v-btn :disabled="loading || submitting || inputIsNotChangedOrInvalid" @click="save">
                        {{ tt('Save') }}
                        <v-progress-circular indeterminate size="22" class="ms-2" v-if="submitting"></v-progress-circular>
                    </v-btn>
                    <v-btn color="secondary" variant="tonal" :disabled="loading || submitting" @click="cancel">{{ tt('Cancel') }}</v-btn>
                </div>
            </v-card-text>
        </v-card>
    </v-dialog>

    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useLlmPromptEditPageBase } from '@/views/base/llmprompts/LlmPromptEditPageBase.ts';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;

export interface LlmPromptEditDialogOpenOptions {
    id?: string;
    duplicateFromContent?: string;
}

const { tt } = useI18n();

const {
    llmPromptsStore,
    prompt,
    loading,
    submitting,
    previewing,
    showPreview,
    previewedContent,
    allPlaceholderHints,
    title,
    missingPlaceholders,
    inputIsNotChanged,
    inputIsNotChangedOrInvalid,
    setLoadedPrompt,
    setNewPrompt,
    loadPreview
} = useLlmPromptEditPageBase();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

let resolveFunc: ((value: boolean) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

const showState = ref<boolean>(false);

function open(options?: LlmPromptEditDialogOpenOptions): Promise<boolean> {
    showState.value = true;
    showPreview.value = false;
    previewedContent.value = '';
    setNewPrompt(options?.duplicateFromContent);

    if (options?.id) {
        loading.value = true;

        llmPromptsStore.getPrompt({ id: options.id }).then(loadedPrompt => {
            setLoadedPrompt(loadedPrompt);
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            showState.value = false;
            rejectFunc?.(error);
        });
    }

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function preview(): void {
    loadPreview().catch(error => {
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function save(): void {
    submitting.value = true;

    llmPromptsStore.savePrompt({ prompt: prompt.value }).then(() => {
        submitting.value = false;
        showState.value = false;
        resolveFunc?.(true);
    }).catch(error => {
        submitting.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function cancel(): void {
    if (inputIsNotChanged.value) {
        rejectFunc?.();
        showState.value = false;
        return;
    }

    confirmDialog.value?.open('Are you sure you want to discard unsaved changes?').then(() => {
        rejectFunc?.();
        showState.value = false;
    });
}

defineExpose({
    open
});
</script>

<style>
.llm-prompt-content-input textarea {
    font-family: monospace;
    font-size: 0.875rem;
}
</style>
