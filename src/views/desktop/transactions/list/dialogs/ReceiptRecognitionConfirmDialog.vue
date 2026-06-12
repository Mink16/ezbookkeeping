<template>
    <v-dialog width="800" :persistent="true" v-model="showState">
        <v-card class="pa-2 pa-sm-4 pa-md-4">
            <template #title>
                <div class="d-flex align-center justify-center">
                    <h4 class="text-h4">{{ tt('Confirm Recognition Result') }}</h4>
                </div>
            </template>
            <v-card-text class="my-md-4">
                <v-form class="w-100">
                    <v-row>
                        <v-col cols="12" md="4" v-if="imageSrc">
                            <v-img class="border rounded receipt-recognition-confirm-image" max-height="200" :src="imageSrc" />
                        </v-col>
                        <v-col cols="12" :md="imageSrc ? 8 : 12">
                            <v-row>
                                <v-col cols="12" md="6">
                                    <v-tooltip :disabled="!!allVisibleCategorizedAccounts.length" :text="allVisibleCategorizedAccounts.length ? '' : tt('No available account')">
                                        <template v-slot:activator="{ props }">
                                            <div v-bind="props" class="d-block">
                                                <two-column-select primary-key-field="id" primary-value-field="category"
                                                                   primary-title-field="name" primary-footer-field="displayBalance"
                                                                   primary-icon-field="icon" primary-icon-type="account"
                                                                   primary-sub-items-field="accounts"
                                                                   :primary-title-i18n="true"
                                                                   secondary-key-field="id" secondary-value-field="id"
                                                                   secondary-title-field="name" secondary-footer-field="displayBalance"
                                                                   secondary-icon-field="icon" secondary-icon-type="account" secondary-color-field="color"
                                                                   :disabled="loading || registering || !allVisibleCategorizedAccounts.length"
                                                                   :enable-filter="true" :filter-placeholder="tt('Find account')" :filter-no-items-text="tt('No available account')"
                                                                   :custom-selection-primary-text="sourceAccountName"
                                                                   :label="tt('Account')"
                                                                   :placeholder="tt('Account')"
                                                                   :items="allVisibleCategorizedAccounts"
                                                                   v-model="accountId">
                                                </two-column-select>
                                            </div>
                                        </template>
                                    </v-tooltip>
                                </v-col>
                                <v-col cols="12" md="6">
                                    <amount-input :currency="accountCurrency"
                                                  :show-currency="true"
                                                  :persistent-placeholder="true"
                                                  :disabled="loading || registering"
                                                  :label="tt('Receipt Total')"
                                                  :placeholder="tt('Receipt Total')"
                                                  :model-value="receiptTotal"
                                                  @update:model-value="updateReceiptTotal" />
                                </v-col>
                                <v-col cols="12" md="6">
                                    <v-text-field type="text" readonly persistent-placeholder
                                                  :label="tt('Transaction Time')"
                                                  :model-value="displayTime" />
                                </v-col>
                            </v-row>
                        </v-col>

                        <v-col cols="12" v-if="hasResolvedLocation">
                            <div class="text-subtitle-2">{{ tt('Resolved Location') }}</div>
                            <div class="text-body-2 text-medium-emphasis mb-2" v-if="resolvedLocation">
                                <v-icon class="me-1" size="16" :icon="mdiMapMarkerOutline" />
                                <span>{{ resolvedLocation.displayName }}</span>
                            </div>
                            <map-view ref="map" height="200px" map-class="receipt-recognition-confirm-map-view"
                                      :geo-location="recognizedGeoLocation ?? undefined"
                                      v-if="mapProviderAvailable" />
                            <v-switch color="primary" density="compact" hide-details
                                      :disabled="loading || registering"
                                      :label="tt('Attach Geolocation')"
                                      v-model="attachGeoLocation" />
                        </v-col>

                        <v-col cols="12">
                            <v-btn-toggle class="receipt-recognition-confirm-mode" color="primary" variant="outlined"
                                          density="comfortable" mandatory
                                          :disabled="loading || registering || savedRowCount > 0"
                                          v-model="mode">
                                <v-btn value="combined">{{ tt('Register as One Transaction') }}</v-btn>
                                <v-btn value="separate">{{ tt('Register Items Separately') }}</v-btn>
                            </v-btn-toggle>
                        </v-col>

                        <template v-if="mode === 'combined'">
                            <v-col cols="12">
                                <v-tooltip :disabled="hasVisibleCategories" :text="hasVisibleCategories ? '' : tt('No available category')">
                                    <template v-slot:activator="{ props }">
                                        <div v-bind="props" class="d-block">
                                            <two-column-select primary-key-field="id" primary-value-field="id" primary-title-field="name"
                                                               primary-icon-field="icon" primary-icon-type="category" primary-color-field="color"
                                                               primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                                               secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                                               secondary-icon-field="icon" secondary-icon-type="category" secondary-color-field="color"
                                                               secondary-hidden-field="hidden"
                                                               :disabled="loading || registering || !hasVisibleCategories"
                                                               :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                                               :show-selection-primary-text="true"
                                                               :custom-selection-primary-text="getTransactionPrimaryCategoryName(receiptCategoryId, categoryItems)"
                                                               :custom-selection-secondary-text="getTransactionSecondaryCategoryName(receiptCategoryId, categoryItems)"
                                                               :label="tt('Category')" :placeholder="tt('Category')"
                                                               :items="categoryItems"
                                                               v-model="receiptCategoryId">
                                            </two-column-select>
                                        </div>
                                    </template>
                                </v-tooltip>
                            </v-col>
                            <v-col cols="12">
                                <v-textarea type="text" persistent-placeholder rows="2"
                                            :disabled="loading || registering"
                                            :label="tt('Description')"
                                            :placeholder="tt('Your transaction description (optional)')"
                                            v-model="receiptComment" />
                            </v-col>
                        </template>

                        <template v-if="mode === 'separate'">
                            <v-col cols="12">
                                <v-row class="align-center" dense :key="row.id" v-for="row in rows">
                                    <v-col cols="1" class="d-flex justify-center">
                                        <v-checkbox density="compact" hide-details
                                                    :disabled="loading || registering || row.saveStatus === 'saved'"
                                                    :model-value="row.include"
                                                    @update:model-value="setRowInclude(row.id, !!$event)" />
                                    </v-col>
                                    <v-col cols="11" md="4">
                                        <v-text-field type="text" density="compact" hide-details
                                                      :disabled="loading || registering || row.saveStatus === 'saved'"
                                                      :model-value="row.comment"
                                                      @update:model-value="updateRowComment(row.id, $event)"
                                                      v-if="!row.isAdjustment" />
                                        <div class="d-flex align-center text-medium-emphasis" v-else>
                                            <v-icon class="me-1" size="18" :icon="mdiScaleBalance" />
                                            <span class="text-truncate">{{ getAdjustmentDisplayName(row.adjustmentKind) }}</span>
                                        </div>
                                    </v-col>
                                    <v-col cols="5" md="3" offset="1" offset-md="0">
                                        <amount-input density="compact"
                                                      :currency="accountCurrency"
                                                      :show-currency="true"
                                                      :readonly="row.isAdjustment"
                                                      :disabled="loading || registering || row.saveStatus === 'saved'"
                                                      :model-value="row.amount"
                                                      @update:model-value="updateRowAmount(row.id, $event)" />
                                    </v-col>
                                    <v-col cols="5" md="3">
                                        <two-column-select primary-key-field="id" primary-value-field="id" primary-title-field="name"
                                                           primary-icon-field="icon" primary-icon-type="category" primary-color-field="color"
                                                           primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                                           secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                                           secondary-icon-field="icon" secondary-icon-type="category" secondary-color-field="color"
                                                           secondary-hidden-field="hidden"
                                                           density="compact"
                                                           :disabled="loading || registering || row.saveStatus === 'saved' || !hasVisibleCategories"
                                                           :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                                           :custom-selection-secondary-text="getTransactionSecondaryCategoryName(row.categoryId, categoryItems) || tt('Category')"
                                                           :placeholder="tt('Category')"
                                                           :items="categoryItems"
                                                           :model-value="row.categoryId"
                                                           @update:model-value="onRowCategoryChanged(row.id, $event)">
                                        </two-column-select>
                                    </v-col>
                                    <v-col cols="1" class="d-flex justify-center">
                                        <v-progress-circular indeterminate size="20" v-if="row.saveStatus === 'saving'" />
                                        <v-icon color="success" :icon="mdiCheckCircleOutline" v-else-if="row.saveStatus === 'saved'" />
                                        <v-icon color="error" :icon="mdiAlertCircleOutline" v-else-if="row.saveStatus === 'failed'" />
                                    </v-col>
                                </v-row>
                            </v-col>
                            <v-col cols="12">
                                <div class="d-flex align-center flex-wrap gap-2">
                                    <span class="text-body-2">{{ tt('Items Total') }}: <strong>{{ displayIncludedTotal }}</strong></span>
                                    <span class="text-body-2 text-medium-emphasis">{{ tt('Receipt Total') }}: {{ displayReceiptTotal }}</span>
                                    <v-chip color="warning" size="small" v-if="hasSelectionMismatch">
                                        {{ tt('The total of selected items differs from the receipt total') }}
                                    </v-chip>
                                    <v-btn color="secondary" variant="tonal" size="small"
                                           :disabled="loading || registering"
                                           @click="distributeDifference"
                                           v-if="canDistributeDifference">{{ tt('Distribute Difference to Items') }}</v-btn>
                                </div>
                            </v-col>
                        </template>
                    </v-row>
                </v-form>
            </v-card-text>
            <v-card-text class="overflow-y-visible">
                <div class="w-100 d-flex justify-center gap-4">
                    <v-btn :disabled="loading || !canConfirm" @click="confirmReceipt">
                        {{ tt('Add') }}
                        <v-progress-circular indeterminate size="22" class="ms-2" v-if="registering"></v-progress-circular>
                    </v-btn>
                    <v-btn color="warning" variant="tonal"
                           :disabled="loading || !canRetryFailedTransactions"
                           @click="retry" v-if="hasFailedTransactions">{{ tt('Retry Failed Items') }}</v-btn>
                    <v-btn color="secondary" variant="tonal" :disabled="registering" @click="cancel">{{ tt('Cancel') }}</v-btn>
                </div>
            </v-card-text>
        </v-card>
    </v-dialog>

    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import MapView from '@/components/common/MapView.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, computed, useTemplateRef, watch, nextTick } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useReceiptRecognitionConfirmBase } from '@/views/base/transactions/ReceiptRecognitionConfirmBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';

import { CategoryType } from '@/core/category.ts';
import { TransactionType } from '@/core/transaction.ts';

import { Account, type CategorizedAccountWithDisplayBalance } from '@/models/account.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';
import type {
    PendingReceiptRecognitionResult,
    ReceiptRegistrationResult
} from '@/models/custom_receipt_recognition.ts';

import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';
import {
    getTransactionPrimaryCategoryName,
    getTransactionSecondaryCategoryName
} from '@/lib/category.ts';
import { getMapProvider } from '@/lib/server_settings.ts';

import {
    mdiMapMarkerOutline,
    mdiScaleBalance,
    mdiCheckCircleOutline,
    mdiAlertCircleOutline
} from '@mdi/js';

interface ReceiptRecognitionConfirmResponse {
    message: string;
}

type MapViewType = InstanceType<typeof MapView>;
type SnackBarType = InstanceType<typeof SnackBar>;

const {
    tt,
    formatDateTimeToLongDateTime,
    formatAmountToLocalizedNumeralsWithCurrency,
    getCategorizedAccountsWithDisplayBalance
} = useI18n();

const {
    rows,
    receiptTotal,
    mode,
    attachGeoLocation,
    accountId,
    type,
    time,
    receiptCategoryId,
    receiptComment,
    recognizedGeoLocation,
    resolvedLocation,
    hasResolvedLocation,
    includedTotal,
    hasSelectionMismatch,
    canDistributeDifference,
    registering,
    savedRowCount,
    hasFailedTransactions,
    canRetryFailedTransactions,
    canConfirm,
    init,
    updateRowAmount,
    updateRowCategory,
    updateRowComment,
    setRowInclude,
    updateReceiptTotal,
    distributeDifference,
    getAdjustmentDisplayName,
    confirm,
    retryFailedTransactions,
    getRegistrationResultMessage
} = useReceiptRecognitionConfirmBase();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();

const map = useTemplateRef<MapViewType>('map');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

let resolveFunc: ((response?: ReceiptRecognitionConfirmResponse) => void) | null = null;
let rejectFunc: ((reason?: unknown) => void) | null = null;

const showState = ref<boolean>(false);
const loading = ref<boolean>(false);
const imageSrc = ref<string | undefined>(undefined);

const allVisibleCategorizedAccounts = computed<CategorizedAccountWithDisplayBalance[]>(() => getCategorizedAccountsWithDisplayBalance(accountsStore.allVisiblePlainAccounts, settingsStore.appSettings.showAccountBalance, settingsStore.appSettings.accountCategoryOrders));

const sourceAccountName = computed<string>(() => {
    if (accountId.value) {
        return Account.findAccountNameById(accountsStore.allPlainAccounts, accountId.value) || '';
    } else {
        return tt('None');
    }
});

const accountCurrency = computed<string>(() => {
    const account = accountsStore.allAccountsMap[accountId.value];

    if (account) {
        return account.currency;
    }

    return userStore.currentUserDefaultCurrency;
});

const categoryItems = computed<TransactionCategory[]>(() => {
    const allCategories = transactionCategoriesStore.allTransactionCategories;

    if (type.value === TransactionType.Income) {
        return allCategories[CategoryType.Income] || [];
    } else if (type.value === TransactionType.Transfer) {
        return allCategories[CategoryType.Transfer] || [];
    } else {
        return allCategories[CategoryType.Expense] || [];
    }
});

const hasVisibleCategories = computed<boolean>(() => {
    if (type.value === TransactionType.Income) {
        return transactionCategoriesStore.hasVisibleIncomeCategories;
    } else if (type.value === TransactionType.Transfer) {
        return transactionCategoriesStore.hasVisibleTransferCategories;
    } else {
        return transactionCategoriesStore.hasVisibleExpenseCategories;
    }
});

// the resolved-location label and the attach-geolocation toggle must stay visible without a map
// provider (coordinates are attached regardless), only the map preview depends on the provider
const mapProviderAvailable = computed<boolean>(() => !!getMapProvider());
const displayTime = computed<string>(() => time.value ? formatDateTimeToLongDateTime(parseDateTimeFromUnixTime(time.value)) : '');
const displayIncludedTotal = computed<string>(() => formatAmountToLocalizedNumeralsWithCurrency(includedTotal.value, accountCurrency.value));
const displayReceiptTotal = computed<string>(() => formatAmountToLocalizedNumeralsWithCurrency(receiptTotal.value, accountCurrency.value));

function open(result: PendingReceiptRecognitionResult): Promise<ReceiptRecognitionConfirmResponse | undefined> {
    showState.value = true;
    loading.value = true;

    releaseImageSrc();
    imageSrc.value = URL.createObjectURL(result.imageFile);

    init(result);

    Promise.all([
        accountsStore.loadAllAccounts({ force: false }),
        transactionCategoriesStore.loadAllCategories({ force: false })
    ]).then(() => {
        loading.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });

    return new Promise((resolve, reject) => {
        resolveFunc = resolve;
        rejectFunc = reject;
    });
}

function close(): void {
    showState.value = false;
    releaseImageSrc();
}

function releaseImageSrc(): void {
    if (imageSrc.value) {
        URL.revokeObjectURL(imageSrc.value);
        imageSrc.value = undefined;
    }
}

function onRegistrationFinished(result: ReceiptRegistrationResult): void {
    if (result.allSaved) {
        resolveFunc?.({ message: getRegistrationResultMessage(result) });
        close();
    } else {
        snackbar.value?.showError(getRegistrationResultMessage(result));
    }
}

function confirmReceipt(): void {
    confirm().then(onRegistrationFinished).catch(error => {
        if (error) {
            snackbar.value?.showError(error);
        }
    });
}

function retry(): void {
    retryFailedTransactions().then(onRegistrationFinished).catch(error => {
        if (error) {
            snackbar.value?.showError(error);
        }
    });
}

function cancel(): void {
    if (registering.value) {
        return;
    }

    if (savedRowCount.value > 0) {
        // some transactions have already been saved, so the caller still needs to reload the list
        resolveFunc?.(undefined);
    } else {
        rejectFunc?.();
    }

    close();
}

function onRowCategoryChanged(id: string, categoryId: unknown): void {
    updateRowCategory(id, categoryId as string);
}

watch(showState, (newValue) => {
    if (newValue) {
        nextTick(() => {
            map.value?.initMapView();
        });
    }
});

defineExpose({
    open
});
</script>

<style>
.receipt-recognition-confirm-map-view {
    height: 200px;
}

.receipt-recognition-confirm-mode .v-btn {
    text-transform: none;
}
</style>
