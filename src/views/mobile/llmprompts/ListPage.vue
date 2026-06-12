<template>
    <f7-page ptr @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt('AI Receipt Recognition Prompts')"></f7-nav-title>
            <f7-nav-right :class="{ 'disabled': loading }">
                <f7-link icon-f7="plus" href="/llm_prompt/add"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-block class="no-margin-bottom">
            <f7-block-footer class="no-margin no-padding">{{ tt('Custom prompts are not applied to the multi-item receipt recognition flow currently in use') }}</f7-block-footer>
        </f7-block>

        <f7-list strong inset dividers class="margin-top skeleton-text" v-if="loading">
            <f7-list-item title="Prompt Name"
                          :key="itemIdx" v-for="itemIdx in [ 1, 2, 3 ]">
                <template #media>
                    <f7-icon f7="doc_plaintext"></f7-icon>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-list strong inset dividers class="margin-top llm-prompt-list" v-else-if="!loading">
            <f7-list-item link="#"
                          :title="prompt.name"
                          :footer="isDefaultPrompt(prompt) ? tt('You cannot edit or delete the default prompt, but you can duplicate it') : undefined"
                          :swipeout="!isDefaultPrompt(prompt)"
                          :key="prompt.id"
                          v-for="prompt in allPromptsWithDefault"
                          @click.prevent="showPromptActionSheet(prompt)">
                <template #media>
                    <f7-icon :f7="isDefaultPrompt(prompt) ? 'wand_stars' : 'doc_plaintext'"></f7-icon>
                </template>
                <template #after>
                    <f7-icon f7="checkmark_alt" v-if="prompt.isActive"></f7-icon>
                </template>
                <f7-swipeout-actions right v-if="!isDefaultPrompt(prompt)">
                    <f7-swipeout-button color="orange" close :text="tt('Edit')" @click="edit(prompt)"></f7-swipeout-button>
                    <f7-swipeout-button color="red" class="padding-horizontal" @click="remove(prompt, false)">
                        <f7-icon f7="trash"></f7-icon>
                    </f7-swipeout-button>
                </f7-swipeout-actions>
            </f7-list-item>
        </f7-list>

        <f7-actions close-by-outside-click close-on-escape :opened="showActionSheet" @actions:closed="showActionSheet = false">
            <f7-actions-group>
                <f7-actions-button :class="{ 'disabled': !selectedPrompt || selectedPrompt.isActive }" @click="setActive(selectedPrompt)">{{ tt('Set as Active') }}</f7-actions-button>
                <f7-actions-button @click="showPromptPreview(selectedPrompt)">{{ tt('Preview') }}</f7-actions-button>
                <f7-actions-button @click="duplicate(selectedPrompt)">{{ tt('Duplicate') }}</f7-actions-button>
                <f7-actions-button v-if="selectedPrompt && !isDefaultPrompt(selectedPrompt)" @click="edit(selectedPrompt)">{{ tt('Edit') }}</f7-actions-button>
                <f7-actions-button color="red" v-if="selectedPrompt && !isDefaultPrompt(selectedPrompt)" @click="remove(selectedPrompt, false)">{{ tt('Delete') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>

        <f7-actions close-by-outside-click close-on-escape :opened="showDeleteActionSheet" @actions:closed="showDeleteActionSheet = false">
            <f7-actions-group>
                <f7-actions-label>{{ tt('Are you sure you want to delete this prompt?') }}</f7-actions-label>
                <f7-actions-button color="red" @click="remove(promptToDelete, true)">{{ tt('Delete') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>

        <f7-popup push :opened="showPreviewPopup" @popup:closed="showPreviewPopup = false">
            <f7-page>
                <f7-navbar>
                    <f7-nav-title :title="tt('Prompt Preview')"></f7-nav-title>
                    <f7-nav-right>
                        <f7-link popup-close>{{ tt('Done') }}</f7-link>
                    </f7-nav-right>
                </f7-navbar>
                <f7-block strong inset class="llm-prompt-preview-content">{{ previewedPopupContent }}</f7-block>
            </f7-page>
        </f7-popup>
    </f7-page>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useLlmPromptListPageBase } from '@/views/base/llmprompts/LlmPromptListPageBase.ts';

import { useLlmPromptsStore } from '@/stores/llmPrompt.ts';

import { DEFAULT_LLM_PROMPT_ID, type LlmPrompt } from '@/models/llm_prompt.ts';

import { isTransactionFromAIImageRecognitionEnabled } from '@/lib/server_settings.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showAlert, showToast, routeBackOnError } = useI18nUIComponents();

const {
    loading,
    loadingError,
    allPromptsWithDefault,
    isDefaultPrompt,
    getPromptContentForDuplicating,
    reload: reloadPromptList
} = useLlmPromptListPageBase();

const llmPromptsStore = useLlmPromptsStore();

const selectedPrompt = ref<LlmPrompt | null>(null);
const promptToDelete = ref<LlmPrompt | null>(null);
const showActionSheet = ref<boolean>(false);
const showDeleteActionSheet = ref<boolean>(false);
const showPreviewPopup = ref<boolean>(false);
const previewedPopupContent = ref<string>('');

function init(): void {
    if (!isTransactionFromAIImageRecognitionEnabled()) {
        props.f7router.back();
        return;
    }

    loading.value = true;

    reloadPromptList(true).then(() => {
        loading.value = false;
    }).catch(error => {
        if (error.processed) {
            loading.value = false;
        } else {
            loadingError.value = error;
            showToast(error.message || error);
        }
    });
}

function reload(done?: () => void): void {
    const force = !!done;

    reloadPromptList(force).then(() => {
        done?.();

        if (force) {
            showToast('Prompt list has been updated');
        }
    }).catch(error => {
        done?.();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function showPromptActionSheet(prompt: LlmPrompt): void {
    selectedPrompt.value = prompt;
    showActionSheet.value = true;
}

function setActive(prompt: LlmPrompt | null): void {
    showActionSheet.value = false;

    if (!prompt) {
        showAlert('An error occurred');
        return;
    }

    showLoading();

    llmPromptsStore.setActivePrompt({ id: isDefaultPrompt(prompt) ? DEFAULT_LLM_PROMPT_ID : prompt.id }).then(() => {
        hideLoading();
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function showPromptPreview(prompt: LlmPrompt | null): void {
    showActionSheet.value = false;

    if (!prompt) {
        showAlert('An error occurred');
        return;
    }

    showLoading();

    getPromptContentForDuplicating(prompt).then(content => {
        return llmPromptsStore.getPreviewedPromptContent({ content });
    }).then(renderedContent => {
        hideLoading();
        previewedPopupContent.value = renderedContent;
        showPreviewPopup.value = true;
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function duplicate(prompt: LlmPrompt | null): void {
    showActionSheet.value = false;

    if (!prompt) {
        showAlert('An error occurred');
        return;
    }

    props.f7router.navigate(`/llm_prompt/add?duplicateFrom=${isDefaultPrompt(prompt) ? DEFAULT_LLM_PROMPT_ID : prompt.id}`);
}

function edit(prompt: LlmPrompt): void {
    showActionSheet.value = false;
    props.f7router.navigate(`/llm_prompt/edit?id=${prompt.id}`);
}

function remove(prompt: LlmPrompt | null, confirm: boolean): void {
    showActionSheet.value = false;

    if (!prompt) {
        showAlert('An error occurred');
        return;
    }

    if (!confirm) {
        promptToDelete.value = prompt;
        showDeleteActionSheet.value = true;
        return;
    }

    showDeleteActionSheet.value = false;
    promptToDelete.value = null;
    showLoading();

    llmPromptsStore.deletePrompt({ prompt }).then(() => {
        hideLoading();
        reload();
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function onPageAfterIn(): void {
    if (llmPromptsStore.llmPromptListStateInvalid && !loading.value) {
        reload();
    }

    routeBackOnError(props.f7router, loadingError);
}

init();
</script>

<style>
.llm-prompt-list {
    --f7-list-item-footer-font-size: var(--ebk-large-footer-font-size);
}

.llm-prompt-preview-content {
    white-space: pre-wrap;
    font-family: monospace;
    font-size: 0.8125rem;
}
</style>
