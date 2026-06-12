<template>
    <f7-page>
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading || submitting || testing }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt(title)"></f7-nav-title>
            <f7-nav-right :class="{ 'disabled': loading }">
                <f7-link icon-f7="checkmark_alt" :class="{ 'disabled': inputIsNotChangedOrInvalid || submitting || testing }" @click="save"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-list strong inset dividers class="margin-top skeleton-text" v-if="loading">
            <f7-list-input label="Connection Name" placeholder="Your connection name"></f7-list-input>
            <f7-list-input label="Provider" placeholder="Provider"></f7-list-input>
            <f7-list-input label="Model ID" placeholder="Your model ID"></f7-list-input>
        </f7-list>

        <f7-list form strong inset dividers class="margin-top" v-else-if="!loading">
            <f7-list-input
                type="text"
                clear-button
                :label="tt('Connection Name')"
                :placeholder="tt('Your connection name')"
                v-model:value="profile.name"
            ></f7-list-input>

            <f7-list-item
                class="list-item-with-header-and-title list-item-no-item-after"
                link="#"
                :class="{ 'disabled': submitting || testing }"
                :header="tt('Provider')"
                :title="getLlmProviderDisplayName(profile.provider)"
                @click="showProviderPopup = true">
                <list-item-selection-popup value-type="item"
                                           key-field="value" value-field="value"
                                           title-field="name"
                                           :title="tt('Provider')"
                                           :items="allProviderItems"
                                           v-model:show="showProviderPopup"
                                           v-model="profile.provider"
                                           @update:model-value="onProviderChanged">
                </list-item-selection-popup>
            </f7-list-item>

            <f7-list-input
                :key="fieldDef.field"
                :type="fieldDef.secret ? 'password' : 'text'"
                :clear-button="!fieldDef.secret"
                :label="tt(fieldDef.labelKey)"
                :placeholder="getFieldPlaceholder(fieldDef, tt)"
                v-for="fieldDef in visibleFields"
                v-model:value="profile[fieldDef.field]"
            ></f7-list-input>
        </f7-list>

        <f7-block class="no-margin-top margin-bottom llm-profile-api-key-hint" v-if="!loading && !isNewProfile && profile.apiKeyConfigured">
            {{ tt('API key is configured, leave blank to keep it unchanged') }}
        </f7-block>

        <f7-list strong inset dividers v-if="!loading">
            <f7-list-button :class="{ 'disabled': !canTest || testing || submitting }" @click="test">{{ tt('Test Connection') }}</f7-list-button>
        </f7-list>

        <f7-block class="no-margin-top margin-bottom llm-profile-test-result" v-if="!loading && testResult"
                  :class="testResult.success ? 'llm-profile-test-result-success' : 'llm-profile-test-result-failure'">
            <p v-if="testResult.success">{{ tt('Connection test succeeded') }}</p>
            <p v-else>{{ tt('format.misc.llmProfileTestFailed', { reason: tt(getLlmProfileTestResultMessageKey(testResult.result)) }) }}</p>
            <p v-if="testResult.recognizedText">{{ tt('format.misc.llmProfileTestRecognizedText', { text: testResult.recognizedText }) }}</p>
        </f7-block>
    </f7-page>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useLlmProfileEditPageBase } from '@/views/base/llmprofiles/LlmProfileEditPageBase.ts';

import { getLlmProviderDisplayName, getLlmProfileTestResultMessageKey } from '@/lib/llm_profile.ts';

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showToast, routeBackOnError } = useI18nUIComponents();

const {
    llmProfilesStore,
    profile,
    loading,
    submitting,
    testing,
    testResult,
    allProviderItems,
    isNewProfile,
    title,
    visibleFields,
    inputIsNotChangedOrInvalid,
    canTest,
    getFieldPlaceholder,
    setLoadedProfile,
    setNewProfile,
    onProviderChanged,
    testConnection
} = useLlmProfileEditPageBase();

const loadingError = ref<unknown>(null);
const showProviderPopup = ref<boolean>(false);

function init(): void {
    const query = props.f7route.query;

    if (query['id']) {
        loading.value = true;

        llmProfilesStore.getProfile({ id: query['id'] }).then(loadedProfile => {
            setLoadedProfile(loadedProfile);
            loading.value = false;
        }).catch(error => {
            if (error.processed) {
                loading.value = false;
            } else {
                loadingError.value = error;
                showToast(error.message || error);
            }
        });
    } else {
        setNewProfile();
    }
}

function save(): void {
    submitting.value = true;
    showLoading();

    llmProfilesStore.saveProfile({ profile: profile.value }).then(() => {
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

function test(): void {
    showLoading();

    testConnection().then(() => {
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
.llm-profile-api-key-hint {
    color: var(--f7-block-footer-text-color);
    font-size: var(--ebk-large-footer-font-size);
}

.llm-profile-test-result {
    font-size: var(--ebk-large-footer-font-size);
}

.llm-profile-test-result p {
    margin: 0 0 2px 0;
}

.llm-profile-test-result-success {
    color: var(--f7-color-green);
}

.llm-profile-test-result-failure {
    color: var(--f7-color-red);
}
</style>
