<template>
    <f7-page>
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading || submitting }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt(title)"></f7-nav-title>
            <f7-nav-right :class="{ 'disabled': loading }">
                <f7-link icon-f7="checkmark_alt" :class="{ 'disabled': inputIsNotChangedOrInvalid || submitting }" @click="save"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-list strong inset dividers class="margin-top skeleton-text" v-if="loading">
            <f7-list-input label="Prompt Name" placeholder="Your prompt name"></f7-list-input>
            <f7-list-input label="Prompt Content" type="textarea" placeholder="Your prompt content"></f7-list-input>
        </f7-list>

        <f7-list form strong inset dividers class="margin-top" v-else-if="!loading">
            <f7-list-input
                type="text"
                clear-button
                :label="tt('Prompt Name')"
                :placeholder="tt('Your prompt name')"
                v-model:value="prompt.name"
            ></f7-list-input>

            <f7-list-input
                class="llm-prompt-content-input"
                type="textarea"
                style="height: auto"
                :label="tt('Prompt Content')"
                :placeholder="tt('Your prompt content')"
                v-textarea-auto-size
                v-model:value="prompt.content"
            ></f7-list-input>
        </f7-list>

        <f7-block class="no-margin-top margin-bottom llm-prompt-placeholder-warning" v-if="!loading && missingPlaceholders.length > 0">
            {{ tt('format.misc.llmPromptMissingPlaceholders', { placeholders: missingPlaceholders.join(', ') }) }}
        </f7-block>

        <f7-list strong inset dividers v-if="!loading">
            <f7-list-button :class="{ 'disabled': !prompt.content || previewing }" @click="preview">{{ tt('Preview Final Prompt') }}</f7-list-button>
        </f7-list>

        <f7-popup push :opened="showPreview" @popup:closed="showPreview = false">
            <f7-page>
                <f7-navbar>
                    <f7-nav-title :title="tt('Prompt Preview')"></f7-nav-title>
                    <f7-nav-right>
                        <f7-link popup-close>{{ tt('Done') }}</f7-link>
                    </f7-nav-right>
                </f7-navbar>
                <f7-block strong inset class="llm-prompt-preview-content">{{ previewedContent }}</f7-block>
            </f7-page>
        </f7-popup>
    </f7-page>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useLlmPromptEditPageBase } from '@/views/base/llmprompts/LlmPromptEditPageBase.ts';

import { DEFAULT_LLM_PROMPT_ID } from '@/models/llm_prompt.ts';

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showToast, routeBackOnError } = useI18nUIComponents();

const {
    llmPromptsStore,
    prompt,
    loading,
    submitting,
    previewing,
    showPreview,
    previewedContent,
    title,
    missingPlaceholders,
    inputIsNotChangedOrInvalid,
    setLoadedPrompt,
    setNewPrompt,
    loadPreview
} = useLlmPromptEditPageBase();

const loadingError = ref<unknown>(null);

function init(): void {
    const query = props.f7route.query;

    if (query['id']) {
        loading.value = true;

        llmPromptsStore.getPrompt({ id: query['id'] }).then(loadedPrompt => {
            setLoadedPrompt(loadedPrompt);
            loading.value = false;
        }).catch(error => {
            if (error.processed) {
                loading.value = false;
            } else {
                loadingError.value = error;
                showToast(error.message || error);
            }
        });
    } else if (query['duplicateFrom']) {
        if (query['duplicateFrom'] === DEFAULT_LLM_PROMPT_ID) {
            setNewPrompt(llmPromptsStore.defaultPromptContent);
        } else {
            loading.value = true;

            llmPromptsStore.getPrompt({ id: query['duplicateFrom'] }).then(loadedPrompt => {
                setNewPrompt(loadedPrompt.content);
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
    } else {
        setNewPrompt();
    }
}

function save(): void {
    submitting.value = true;
    showLoading();

    llmPromptsStore.savePrompt({ prompt: prompt.value }).then(() => {
        submitting.value = false;
        hideLoading();

        props.f7router.back();
    }).catch(error => {
        submitting.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function preview(): void {
    showLoading();

    loadPreview().then(() => {
        hideLoading();
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

routeBackOnError(props.f7router, loadingError);
init();
</script>

<style>
.llm-prompt-content-input textarea {
    font-family: monospace;
    font-size: 0.8125rem;
    min-height: 200px;
}

.llm-prompt-placeholder-warning {
    color: var(--f7-color-orange);
    font-size: var(--ebk-large-footer-font-size);
}
</style>
