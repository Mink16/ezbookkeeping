<template>
    <v-row>
        <v-col cols="12">
            <v-card>
                <template #title>
                    <div class="title-and-toolbar d-flex align-center">
                        <span>{{ tt('AI Recognition LLM Connections') }}</span>
                        <v-btn class="ms-3" color="default" variant="outlined"
                               :disabled="loading || updating" @click="add">{{ tt('Add') }}</v-btn>
                        <v-btn density="compact" color="default" variant="text" size="24"
                               class="ms-2" :icon="true" :disabled="loading || updating"
                               :loading="loading" @click="reloadProfiles(true)">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="24" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </template>

                <v-card-text>
                    <v-table class="llm-profiles-table" :hover="!loading">
                        <thead>
                        <tr>
                            <th class="text-uppercase" style="width: 80px">{{ tt('Active') }}</th>
                            <th class="text-uppercase">{{ tt('Connection Name') }}</th>
                            <th class="text-uppercase">{{ tt('Model ID') }}</th>
                            <th class="text-uppercase">{{ tt('API Key') }}</th>
                            <th class="text-uppercase text-right">{{ tt('Operation') }}</th>
                        </tr>
                        </thead>
                        <tbody v-if="loading && allProfilesWithDefault.length < 1">
                        <tr :key="itemIdx" v-for="itemIdx in [ 1, 2 ]">
                            <td class="px-0" colspan="5">
                                <v-skeleton-loader type="text" :loading="true"></v-skeleton-loader>
                            </td>
                        </tr>
                        </tbody>
                        <tbody v-else>
                        <tr :key="profile.id" v-for="profile in allProfilesWithDefault">
                            <td>
                                <v-radio density="compact" hide-details
                                         :disabled="loading || updating || (isDefaultProfile(profile) && !profile.provider)"
                                         :model-value="(isDefaultProfile(profile) && activeProfileId === DEFAULT_LLM_PROFILE_ID) || (!isDefaultProfile(profile) && activeProfileId === profile.id)"
                                         @click="setActive(profile)"></v-radio>
                            </td>
                            <td>
                                <span v-if="!isDefaultProfile(profile)">{{ profile.name }}</span>
                                <v-chip size="small" v-if="isDefaultProfile(profile)">{{ tt('Default') }}</v-chip>
                                <span class="text-caption text-medium-emphasis ms-2" v-if="profile.provider">{{ getProviderDisplayName(profile) }}</span>
                                <span class="text-caption text-medium-emphasis ms-2" v-else-if="isDefaultProfile(profile)">{{ tt('Not configured in server settings') }}</span>
                            </td>
                            <td>
                                <span class="llm-profile-model-id">{{ profile.modelId }}</span>
                            </td>
                            <td>
                                <v-chip size="small" color="secondary" v-if="profile.apiKeyConfigured">{{ tt('Configured') }}</v-chip>
                            </td>
                            <td class="text-right">
                                <v-btn density="comfortable" color="default" variant="text" :icon="true"
                                       :disabled="loading || updating" v-if="!isDefaultProfile(profile)" @click="edit(profile)">
                                    <v-icon :icon="mdiPencilOutline" size="20" />
                                    <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                </v-btn>
                                <v-btn density="comfortable" color="error" variant="text" :icon="true"
                                       :disabled="loading || updating" v-if="!isDefaultProfile(profile)" @click="remove(profile)">
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

    <llm-profile-edit-dialog ref="editDialog" />
    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';
import LlmProfileEditDialog from '@/views/desktop/app/settings/dialogs/LlmProfileEditDialog.vue';

import { ref, computed, useTemplateRef, onMounted } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useLlmProfileListPageBase } from '@/views/base/llmprofiles/LlmProfileListPageBase.ts';

import { DEFAULT_LLM_PROFILE_ID, type LlmProfile } from '@/models/llm_profile.ts';

import {
    mdiRefresh,
    mdiPencilOutline,
    mdiDeleteOutline
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;
type LlmProfileEditDialogType = InstanceType<typeof LlmProfileEditDialog>;

const { tt } = useI18n();

const {
    llmProfilesStore,
    loading,
    allProfilesWithDefault,
    isDefaultProfile,
    getProviderDisplayName,
    reload
} = useLlmProfileListPageBase();

const editDialog = useTemplateRef<LlmProfileEditDialogType>('editDialog');
const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const updating = ref<boolean>(false);

const activeProfileId = computed<string>(() => llmProfilesStore.activeLlmProfileId);

function reloadProfiles(force: boolean): void {
    loading.value = true;

    reload(force).then(() => {
        loading.value = false;

        if (force) {
            snackbar.value?.showMessage('Connection list has been updated');
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
        reloadProfiles(false);
    }).catch(() => {
        // dialog dismissed
    });
}

function edit(profile: LlmProfile): void {
    editDialog.value?.open({ id: profile.id }).then(() => {
        reloadProfiles(false);
    }).catch(error => {
        if (error && !error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function remove(profile: LlmProfile): void {
    confirmDialog.value?.open('Are you sure you want to delete this connection?').then(() => {
        updating.value = true;

        llmProfilesStore.deleteProfile({ profile }).then(() => {
            updating.value = false;
            reloadProfiles(false);
        }).catch(error => {
            updating.value = false;

            if (!error.processed) {
                snackbar.value?.showError(error);
            }
        });
    });
}

function setActive(profile: LlmProfile): void {
    const targetId = isDefaultProfile(profile) ? DEFAULT_LLM_PROFILE_ID : profile.id;

    if (updating.value || loading.value || activeProfileId.value === targetId) {
        return;
    }

    updating.value = true;

    llmProfilesStore.setActiveProfile({ id: targetId }).then(() => {
        updating.value = false;
    }).catch(error => {
        updating.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

onMounted(() => {
    reloadProfiles(false);
});
</script>

<style>
.llm-profiles-table .llm-profile-model-id {
    font-family: monospace;
    font-size: 0.8125rem;
}
</style>
