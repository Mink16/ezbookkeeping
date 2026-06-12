<template>
    <f7-page ptr @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt('AI Recognition LLM Connections')"></f7-nav-title>
            <f7-nav-right :class="{ 'disabled': loading }">
                <f7-link icon-f7="plus" href="/llm_profile/add"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-list strong inset dividers class="margin-top skeleton-text" v-if="loading">
            <f7-list-item title="Connection Name" footer="Provider"
                          :key="itemIdx" v-for="itemIdx in [ 1, 2 ]">
                <template #media>
                    <f7-icon f7="link"></f7-icon>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-list strong inset dividers class="margin-top llm-profile-list" v-else-if="!loading">
            <f7-list-item link="#"
                          :title="isDefaultProfile(profile) ? tt('Default') : profile.name"
                          :footer="getProfileFooter(profile)"
                          :swipeout="!isDefaultProfile(profile)"
                          :key="profile.id"
                          v-for="profile in allProfilesWithDefault"
                          @click.prevent="showProfileActionSheet(profile)">
                <template #media>
                    <f7-icon :f7="isDefaultProfile(profile) ? 'gear_alt' : 'link'"></f7-icon>
                </template>
                <template #after>
                    <f7-icon f7="checkmark_alt" v-if="profile.isActive"></f7-icon>
                </template>
                <f7-swipeout-actions right v-if="!isDefaultProfile(profile)">
                    <f7-swipeout-button color="orange" close :text="tt('Edit')" @click="edit(profile)"></f7-swipeout-button>
                    <f7-swipeout-button color="red" class="padding-horizontal" @click="remove(profile, false)">
                        <f7-icon f7="trash"></f7-icon>
                    </f7-swipeout-button>
                </f7-swipeout-actions>
            </f7-list-item>
        </f7-list>

        <f7-actions close-by-outside-click close-on-escape :opened="showActionSheet" @actions:closed="showActionSheet = false">
            <f7-actions-group>
                <f7-actions-button :class="{ 'disabled': !selectedProfile || selectedProfile.isActive || (selectedProfile.isDefault && !selectedProfile.provider) }" @click="setActive(selectedProfile)">{{ tt('Set as Active') }}</f7-actions-button>
                <f7-actions-button v-if="selectedProfile && !isDefaultProfile(selectedProfile)" @click="edit(selectedProfile)">{{ tt('Edit') }}</f7-actions-button>
                <f7-actions-button color="red" v-if="selectedProfile && !isDefaultProfile(selectedProfile)" @click="remove(selectedProfile, false)">{{ tt('Delete') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>

        <f7-actions close-by-outside-click close-on-escape :opened="showDeleteActionSheet" @actions:closed="showDeleteActionSheet = false">
            <f7-actions-group>
                <f7-actions-label>{{ tt('Are you sure you want to delete this connection?') }}</f7-actions-label>
                <f7-actions-button color="red" @click="remove(profileToDelete, true)">{{ tt('Delete') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>
    </f7-page>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useLlmProfileListPageBase } from '@/views/base/llmprofiles/LlmProfileListPageBase.ts';

import { useServerAdminStore } from '@/stores/serverAdmin.ts';

import { DEFAULT_LLM_PROFILE_ID, type LlmProfile } from '@/models/llm_profile.ts';

import { isTransactionFromAIImageRecognitionEnabled } from '@/lib/server_settings.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showAlert, showToast, routeBackOnError } = useI18nUIComponents();

const {
    llmProfilesStore,
    loading,
    loadingError,
    allProfilesWithDefault,
    isDefaultProfile,
    getProviderDisplayName,
    reload: reloadProfileList
} = useLlmProfileListPageBase();

const serverAdminStore = useServerAdminStore();

const selectedProfile = ref<LlmProfile | null>(null);
const profileToDelete = ref<LlmProfile | null>(null);
const showActionSheet = ref<boolean>(false);
const showDeleteActionSheet = ref<boolean>(false);

function getProfileFooter(profile: LlmProfile): string | undefined {
    if (isDefaultProfile(profile)) {
        if (!profile.provider) {
            return tt('Not configured in server settings');
        }

        return getProviderDisplayName(profile) + ' · ' + tt('You cannot edit or delete the default connection');
    }

    if (profile.modelId) {
        return getProviderDisplayName(profile) + ' · ' + profile.modelId;
    }

    return getProviderDisplayName(profile);
}

function init(): void {
    if (!isTransactionFromAIImageRecognitionEnabled()) {
        props.f7router.back();
        return;
    }

    loading.value = true;

    serverAdminStore.loadPermission({}).then(administrable => {
        if (!administrable) {
            props.f7router.back();
            return;
        }

        reloadProfileList(true).then(() => {
            loading.value = false;
        }).catch(error => {
            if (error.processed) {
                loading.value = false;
            } else {
                loadingError.value = error;
                showToast(error.message || error);
            }
        });
    });
}

function reload(done?: () => void): void {
    const force = !!done;

    reloadProfileList(force).then(() => {
        done?.();

        if (force) {
            showToast('Connection list has been updated');
        }
    }).catch(error => {
        done?.();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function showProfileActionSheet(profile: LlmProfile): void {
    selectedProfile.value = profile;
    showActionSheet.value = true;
}

function setActive(profile: LlmProfile | null): void {
    showActionSheet.value = false;

    if (!profile) {
        showAlert('An error occurred');
        return;
    }

    showLoading();

    llmProfilesStore.setActiveProfile({ id: isDefaultProfile(profile) ? DEFAULT_LLM_PROFILE_ID : profile.id }).then(() => {
        hideLoading();
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function edit(profile: LlmProfile): void {
    showActionSheet.value = false;
    props.f7router.navigate(`/llm_profile/edit?id=${profile.id}`);
}

function remove(profile: LlmProfile | null, confirm: boolean): void {
    showActionSheet.value = false;

    if (!profile) {
        showAlert('An error occurred');
        return;
    }

    if (!confirm) {
        profileToDelete.value = profile;
        showDeleteActionSheet.value = true;
        return;
    }

    showDeleteActionSheet.value = false;
    profileToDelete.value = null;
    showLoading();

    llmProfilesStore.deleteProfile({ profile }).then(() => {
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
    if (llmProfilesStore.llmProfileListStateInvalid && !loading.value) {
        reload();
    }

    routeBackOnError(props.f7router, loadingError);
}

init();
</script>

<style>
.llm-profile-list {
    --f7-list-item-footer-font-size: var(--ebk-large-footer-font-size);
}
</style>
