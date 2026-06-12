<template>
    <v-row>
        <v-col cols="12">
            <v-card>
                <template #title>
                    <div class="title-and-toolbar d-flex align-center">
                        <span>{{ tt('AI Receipt Recognition Prompts') }}</span>
                        <v-btn class="ms-3" color="default" variant="outlined"
                               :disabled="loading || updating" @click="add">{{ tt('Add') }}</v-btn>
                        <v-btn density="compact" color="default" variant="text" size="24"
                               class="ms-2" :icon="true" :disabled="loading || updating"
                               :loading="loading" @click="reloadPrompts(true)">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="24" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </template>

                <v-card-text>
                    <v-table class="llm-prompts-table" :hover="!loading">
                        <thead>
                        <tr>
                            <th class="text-uppercase" style="width: 80px">{{ tt('Active') }}</th>
                            <th class="text-uppercase">{{ tt('Prompt Name') }}</th>
                            <th class="text-uppercase text-right">{{ tt('Operation') }}</th>
                        </tr>
                        </thead>
                        <tbody v-if="loading && allPromptsWithDefault.length < 2">
                        <tr :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                            <td class="px-0" colspan="3">
                                <v-skeleton-loader type="text" :loading="true"></v-skeleton-loader>
                            </td>
                        </tr>
                        </tbody>
                        <tbody v-else>
                        <tr :key="prompt.id" v-for="prompt in allPromptsWithDefault">
                            <td>
                                <v-radio density="compact" hide-details
                                         :disabled="loading || updating"
                                         :model-value="activePromptId === prompt.id"
                                         @click="setActive(prompt)"></v-radio>
                            </td>
                            <td>
                                <span>{{ prompt.name }}</span>
                                <v-chip class="ms-2" size="small" v-if="isDefaultPrompt(prompt)">{{ tt('Default') }}</v-chip>
                            </td>
                            <td class="text-right">
                                <v-btn density="comfortable" color="default" variant="text" :icon="true"
                                       :disabled="loading || updating" @click="showPromptPreview(prompt)">
                                    <v-icon :icon="mdiEyeOutline" size="20" />
                                    <v-tooltip activator="parent">{{ tt('Preview') }}</v-tooltip>
                                </v-btn>
                                <v-btn density="comfortable" color="default" variant="text" :icon="true"
                                       :disabled="loading || updating" @click="duplicate(prompt)">
                                    <v-icon :icon="mdiContentCopy" size="20" />
                                    <v-tooltip activator="parent">{{ tt('Duplicate') }}</v-tooltip>
                                </v-btn>
                                <v-btn density="comfortable" color="default" variant="text" :icon="true"
                                       :disabled="loading || updating" v-if="!isDefaultPrompt(prompt)" @click="edit(prompt)">
                                    <v-icon :icon="mdiPencilOutline" size="20" />
                                    <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                </v-btn>
                                <v-btn density="comfortable" color="error" variant="text" :icon="true"
                                       :disabled="loading || updating" v-if="!isDefaultPrompt(prompt)" @click="remove(prompt)">
                                    <v-icon :icon="mdiDeleteOutline" size="20" />
                                    <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                </v-btn>
                            </td>
                        </tr>
                        </tbody>
                    </v-table>
                </v-card-text>
            </v-card>
        </v-col>
    </v-row>

    <v-dialog width="800" v-model="showPreviewDialog">
        <v-card class="pa-2 pa-sm-4 pa-md-4">
            <template #title>
                <div class="d-flex align-center justify-center">
                    <h4 class="text-h4">{{ tt('Prompt Preview') }}</h4>
                </div>
            </template>
            <v-card-text class="my-md-4 w-100">
                <v-textarea class="llm-prompt-content-input" persistent-placeholder readonly rows="16"
                            v-model="previewDialogContent"/>
            </v-card-text>
            <v-card-text class="overflow-y-visible">
                <div class="w-100 d-flex justify-center">
                    <v-btn color="secondary" variant="tonal" @click="showPreviewDialog = false">{{ tt('Close') }}</v-btn>
                </div>
            </v-card-text>
        </v-card>
    </v-dialog>

    <llm-prompt-edit-dialog ref="editDialog" />
    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';
import LlmPromptEditDialog from '@/views/desktop/app/settings/dialogs/LlmPromptEditDialog.vue';

import { ref, computed, useTemplateRef, onMounted } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useLlmPromptListPageBase } from '@/views/base/llmprompts/LlmPromptListPageBase.ts';

import { useLlmPromptsStore } from '@/stores/llmPrompt.ts';

import { DEFAULT_LLM_PROMPT_ID, type LlmPrompt } from '@/models/llm_prompt.ts';

import {
    mdiRefresh,
    mdiEyeOutline,
    mdiContentCopy,
    mdiPencilOutline,
    mdiDeleteOutline
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;
type LlmPromptEditDialogType = InstanceType<typeof LlmPromptEditDialog>;

const { tt } = useI18n();

const {
    loading,
    allPromptsWithDefault,
    isDefaultPrompt,
    getPromptContentForDuplicating,
    reload
} = useLlmPromptListPageBase();

const llmPromptsStore = useLlmPromptsStore();

const editDialog = useTemplateRef<LlmPromptEditDialogType>('editDialog');
const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const updating = ref<boolean>(false);
const showPreviewDialog = ref<boolean>(false);
const previewDialogContent = ref<string>('');

const activePromptId = computed<string>(() => llmPromptsStore.activeLlmPromptId);

function reloadPrompts(force: boolean): void {
    loading.value = true;

    reload(force).then(() => {
        loading.value = false;

        if (force) {
            snackbar.value?.showMessage('Prompt list has been updated');
        }
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function add(): void {
    editDialog.value?.open().then(() => {
        reloadPrompts(false);
    }).catch(() => {
        // dialog dismissed
    });
}

function edit(prompt: LlmPrompt): void {
    editDialog.value?.open({ id: prompt.id }).then(() => {
        reloadPrompts(false);
    }).catch(error => {
        if (error && !error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function duplicate(prompt: LlmPrompt): void {
    updating.value = true;

    getPromptContentForDuplicating(prompt).then(content => {
        updating.value = false;

        editDialog.value?.open({ duplicateFromContent: content }).then(() => {
            reloadPrompts(false);
        }).catch(() => {
            // dialog dismissed
        });
    }).catch(error => {
        updating.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function remove(prompt: LlmPrompt): void {
    confirmDialog.value?.open('Are you sure you want to delete this prompt?').then(() => {
        updating.value = true;

        llmPromptsStore.deletePrompt({ prompt }).then(() => {
            updating.value = false;
            reloadPrompts(false);
        }).catch(error => {
            updating.value = false;

            if (!error.processed) {
                snackbar.value?.showError(error);
            }
        });
    });
}

function setActive(prompt: LlmPrompt): void {
    if (updating.value || loading.value || activePromptId.value === prompt.id) {
        return;
    }

    updating.value = true;

    llmPromptsStore.setActivePrompt({ id: isDefaultPrompt(prompt) ? DEFAULT_LLM_PROMPT_ID : prompt.id }).then(() => {
        updating.value = false;
    }).catch(error => {
        updating.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function showPromptPreview(prompt: LlmPrompt): void {
    updating.value = true;

    getPromptContentForDuplicating(prompt).then(content => {
        return llmPromptsStore.getPreviewedPromptContent({ content });
    }).then(renderedContent => {
        updating.value = false;
        previewDialogContent.value = renderedContent;
        showPreviewDialog.value = true;
    }).catch(error => {
        updating.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

onMounted(() => {
    reloadPrompts(false);
});
</script>
