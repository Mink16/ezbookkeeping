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
                        <v-col cols="12" md="6">
                            <v-text-field
                                type="text"
                                persistent-placeholder
                                :disabled="loading || submitting"
                                :label="tt('Connection Name')"
                                :placeholder="tt('Your connection name')"
                                v-model="profile.name"
                            />
                        </v-col>
                        <v-col cols="12" md="6">
                            <v-select
                                item-title="name"
                                item-value="value"
                                persistent-placeholder
                                :disabled="loading || submitting"
                                :label="tt('Provider')"
                                :items="allProviderItems"
                                v-model="profile.provider"
                                @update:model-value="onProviderChanged"
                            />
                        </v-col>
                        <v-col cols="12" :key="fieldDef.field" v-for="fieldDef in visibleFields">
                            <v-text-field
                                persistent-placeholder
                                :type="fieldDef.secret ? 'password' : (fieldDef.type === 'number' ? 'number' : 'text')"
                                :autocomplete="fieldDef.secret ? 'new-password' : undefined"
                                :disabled="loading || submitting"
                                :label="tt(fieldDef.labelKey)"
                                :placeholder="getFieldPlaceholder(fieldDef, tt)"
                                v-model="profile[fieldDef.field]"
                            />
                            <div class="text-caption text-medium-emphasis mt-1" v-if="fieldDef.secret && !isNewProfile && profile.apiKeyConfigured">
                                {{ tt('API key is configured, leave blank to keep it unchanged') }}
                            </div>
                        </v-col>
                        <v-col cols="12" v-if="testResult">
                            <v-alert density="compact" variant="tonal"
                                     :type="testResult.success ? 'success' : 'error'">
                                <span v-if="testResult.success">{{ tt('Connection test succeeded') }}</span>
                                <span v-else>{{ tt('format.misc.llmProfileTestFailed', { reason: tt(getLlmProfileTestResultMessageKey(testResult.result)) }) }}</span>
                                <div class="text-caption" v-if="testResult.recognizedText">{{ tt('format.misc.llmProfileTestRecognizedText', { text: testResult.recognizedText }) }}</div>
                            </v-alert>
                        </v-col>
                    </v-row>
                </v-form>
            </v-card-text>
            <v-card-text class="overflow-y-visible">
                <div class="w-100 d-flex justify-center gap-4">
                    <v-btn color="secondary" variant="tonal" :disabled="loading || submitting || testing || !canTest"
                           @click="test">
                        {{ tt('Test Connection') }}
                        <v-progress-circular indeterminate size="22" class="ms-2" v-if="testing"></v-progress-circular>
                    </v-btn>
                    <v-btn :disabled="loading || submitting || testing || inputIsNotChangedOrInvalid" @click="save">
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

import { useLlmProfileEditPageBase } from '@/views/base/llmprofiles/LlmProfileEditPageBase.ts';

import { getLlmProfileTestResultMessageKey } from '@/lib/llm_profile.ts';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;

export interface LlmProfileEditDialogOpenOptions {
    id?: string;
}

const { tt } = useI18n();

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
    inputIsNotChanged,
    inputIsNotChangedOrInvalid,
    canTest,
    getFieldPlaceholder,
    setLoadedProfile,
    setNewProfile,
    onProviderChanged,
    testConnection
} = useLlmProfileEditPageBase();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

let resolveFunc: ((value: boolean) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

const showState = ref<boolean>(false);

function open(options?: LlmProfileEditDialogOpenOptions): Promise<boolean> {
    showState.value = true;
    setNewProfile();

    if (options?.id) {
        loading.value = true;

        llmProfilesStore.getProfile({ id: options.id }).then(loadedProfile => {
            setLoadedProfile(loadedProfile);
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

function test(): void {
    testConnection().catch(error => {
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function save(): void {
    submitting.value = true;

    llmProfilesStore.saveProfile({ profile: profile.value }).then(() => {
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
    if (inputIsNotChanged.value || (isNewProfile.value && !profile.value.name && !profile.value.modelId)) {
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
