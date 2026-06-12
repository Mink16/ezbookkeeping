<template>
    <f7-page @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': registering }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt('Confirm Recognition Result')"></f7-nav-title>
        </f7-navbar>

        <f7-list form strong inset dividers class="margin-vertical" :class="{ 'disabled': loading }">
            <f7-list-item media-item v-if="receiptImageSrc || receiptComment || displayTransactionTime">
                <template #media>
                    <img class="receipt-recognition-confirm-thumbnail" alt="receipt" :src="receiptImageSrc" v-if="receiptImageSrc" />
                </template>
                <template #title>
                    <span>{{ receiptComment || tt('None') }}</span>
                </template>
                <template #footer>
                    <span>{{ displayTransactionTime }}</span>
                </template>
            </f7-list-item>

            <f7-list-item
                class="list-item-with-header-and-title"
                link="#" no-chevron
                :class="{ 'disabled': !allVisibleCategorizedAccounts.length || registering }"
                :header="tt('Account')"
                :title="sourceAccountName"
                @click="showAccountSheet = true"
            >
                <two-column-list-item-selection-sheet primary-key-field="id" primary-value-field="category"
                                                      primary-title-field="name" primary-footer-field="displayBalance"
                                                      primary-icon-field="icon" primary-icon-type="account"
                                                      primary-sub-items-field="accounts"
                                                      :primary-title-i18n="true"
                                                      secondary-key-field="id" secondary-value-field="id"
                                                      secondary-title-field="name" secondary-footer-field="displayBalance"
                                                      secondary-icon-field="icon" secondary-icon-type="account" secondary-color-field="color"
                                                      :enable-filter="true" :filter-placeholder="tt('Find account')" :filter-no-items-text="tt('No available account')"
                                                      :items="allVisibleCategorizedAccounts"
                                                      v-model:show="showAccountSheet"
                                                      v-model="accountId">
                </two-column-list-item-selection-sheet>
            </f7-list-item>

            <f7-list-item
                class="list-item-with-header-and-title"
                link="#" no-chevron
                :class="{ 'disabled': registering }"
                :header="tt('Receipt Total')"
                :title="getDisplayAmount(receiptTotal)"
                @click="showReceiptTotalSheet = true"
            >
                <number-pad-sheet :min-value="TRANSACTION_MIN_AMOUNT"
                                  :max-value="TRANSACTION_MAX_AMOUNT"
                                  :currency="sourceAccountCurrency"
                                  v-model:show="showReceiptTotalSheet"
                                  v-model="editableReceiptTotal"
                ></number-pad-sheet>
            </f7-list-item>

            <f7-list-item
                class="list-item-with-header-and-title list-item-title-hide-overflow"
                link="#" no-chevron
                :class="{ 'disabled': registering }"
                :header="tt('Category')"
                :title="getCategoryDisplayName(receiptCategoryId)"
                @click="showReceiptCategorySheet = true"
                v-if="mode === 'combined'"
            >
                <tree-view-selection-sheet primary-key-field="id" primary-title-field="name"
                                           primary-icon-field="icon" primary-icon-type="category" primary-color-field="color"
                                           primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                           secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                           secondary-icon-field="icon" secondary-icon-type="category" secondary-color-field="color"
                                           secondary-hidden-field="hidden"
                                           :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                           :items="allTransactionCategories"
                                           v-model:show="showReceiptCategorySheet"
                                           v-model="receiptCategoryId">
                </tree-view-selection-sheet>
            </f7-list-item>

            <f7-list-input
                type="textarea"
                style="height: auto"
                :class="{ 'disabled': registering }"
                :label="tt('Description')"
                :placeholder="tt('Your transaction description (optional)')"
                v-textarea-auto-size
                v-model:value="receiptComment"
                v-if="mode === 'combined'"
            ></f7-list-input>
        </f7-list>

        <f7-list strong inset dividers class="margin-vertical" :class="{ 'disabled': loading }" v-if="hasResolvedLocation">
            <f7-list-item
                class="list-item-with-header-and-title list-item-title-hide-overflow"
                :header="tt('Resolved Location')"
                :title="resolvedLocation?.displayName"
            ></f7-list-item>
            <f7-list-item :title="tt('Attach Geolocation')">
                <template #after>
                    <f7-toggle :checked="attachGeoLocation" :disabled="registering"
                               @toggle:change="attachGeoLocation = $event"></f7-toggle>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-block class="no-margin-top margin-bottom receipt-recognition-confirm-map" v-if="hasResolvedLocation && mapProviderAvailable">
            <map-view ref="map" height="200px"
                      :enable-zoom-control="true"
                      :geo-location="recognizedGeoLocation ?? undefined">
            </map-view>
        </f7-block>

        <f7-block class="no-margin-top margin-bottom" :class="{ 'disabled': loading || registering || savedRowCount > 0 }">
            <f7-segmented strong round>
                <f7-button round small :text="tt('Register as One Transaction')" :active="mode === 'combined'"
                           @click="mode = 'combined'"></f7-button>
                <f7-button round small :text="tt('Register Items Separately')" :active="mode === 'separate'"
                           @click="mode = 'separate'"></f7-button>
            </f7-segmented>
        </f7-block>

        <f7-list media-list strong inset dividers class="margin-vertical" :class="{ 'disabled': loading }" v-if="mode === 'separate'">
            <f7-list-item link="#" no-chevron
                          :key="row.id"
                          :class="{ 'disabled': registering }"
                          v-for="row in rows"
                          @click="onRowClick(row)">
                <template #media>
                    <!-- a plain icon toggle is used instead of f7-checkbox because a native
                         checkbox inside a link list item gets its click swallowed by the
                         framework7 link handling and the include state silently desyncs -->
                    <f7-link class="receipt-recognition-confirm-row-include" :class="{ 'disabled': registering || row.saveStatus === 'saved' }"
                             @click.stop="onRowIncludeChanged(row)">
                        <f7-icon :f7="row.include ? 'checkmark_circle_fill' : 'circle'"
                                 :class="row.include ? 'text-color-primary' : 'text-color-gray'"></f7-icon>
                    </f7-link>
                </template>
                <template #title>
                    <div class="display-flex align-items-center">
                        <f7-icon class="receipt-recognition-confirm-adjustment-icon" :f7="row.adjustmentKind === 'tax' ? 'percent' : 'plus_slash_minus'" v-if="row.isAdjustment"></f7-icon>
                        <span>{{ getRowDisplayComment(row) }}</span>
                    </div>
                </template>
                <template #footer>
                    <span>{{ getCategoryDisplayName(row.categoryId) }}</span>
                </template>
                <template #after>
                    <span>{{ getDisplayAmount(row.amount) }}</span>
                    <f7-preloader class="receipt-recognition-confirm-row-status" :size="16" v-if="row.saveStatus === 'saving'"></f7-preloader>
                    <f7-icon class="receipt-recognition-confirm-row-status text-color-green" f7="checkmark_alt" v-else-if="row.saveStatus === 'saved'"></f7-icon>
                    <f7-icon class="receipt-recognition-confirm-row-status text-color-red" f7="exclamationmark_circle" v-else-if="row.saveStatus === 'failed'"></f7-icon>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-list strong inset dividers class="margin-vertical" :class="{ 'disabled': loading }" v-if="mode === 'separate'">
            <f7-list-item
                class="list-item-with-header-and-title"
                :header="tt('Items Total')"
                :title="getDisplayAmount(includedTotal)"
            ></f7-list-item>
            <f7-list-button :class="{ 'disabled': registering }" @click="distributeDifference" v-if="canDistributeDifference">{{ tt('Distribute Difference to Items') }}</f7-list-button>
        </f7-list>

        <f7-block class="no-margin-top margin-bottom receipt-recognition-confirm-mismatch" v-if="mode === 'separate' && hasSelectionMismatch">
            {{ tt('The total of selected items differs from the receipt total') }}
        </f7-block>

        <f7-list strong inset dividers class="margin-vertical" :class="{ 'disabled': loading }">
            <f7-list-button :class="{ 'disabled': !canConfirm }" @click="save">{{ tt('Save') }}</f7-list-button>
            <f7-list-button :class="{ 'disabled': !canRetryFailedTransactions }" @click="retry" v-if="hasFailedTransactions">{{ tt('Retry Failed Items') }}</f7-list-button>
        </f7-list>

        <f7-actions close-by-outside-click close-on-escape :opened="showRowActionSheet" @actions:closed="showRowActionSheet = false">
            <f7-actions-group>
                <f7-actions-button @click="showRowAmountSheet = true" v-if="!selectedRow?.isAdjustment">{{ tt('Amount') }}</f7-actions-button>
                <f7-actions-button @click="showRowCategorySheet = true">{{ tt('Category') }}</f7-actions-button>
                <f7-actions-button @click="editSelectedRowComment" v-if="!selectedRow?.isAdjustment">{{ tt('Description') }}</f7-actions-button>
            </f7-actions-group>
            <f7-actions-group>
                <f7-actions-button bold close>{{ tt('Cancel') }}</f7-actions-button>
            </f7-actions-group>
        </f7-actions>

        <number-pad-sheet :min-value="TRANSACTION_MIN_AMOUNT"
                          :max-value="TRANSACTION_MAX_AMOUNT"
                          :currency="sourceAccountCurrency"
                          v-model:show="showRowAmountSheet"
                          v-model="selectedRowAmount"
        ></number-pad-sheet>

        <tree-view-selection-sheet primary-key-field="id" primary-title-field="name"
                                   primary-icon-field="icon" primary-icon-type="category" primary-color-field="color"
                                   primary-hidden-field="hidden" primary-sub-items-field="subCategories"
                                   secondary-key-field="id" secondary-value-field="id" secondary-title-field="name"
                                   secondary-icon-field="icon" secondary-icon-type="category" secondary-color-field="color"
                                   secondary-hidden-field="hidden"
                                   :enable-filter="true" :filter-placeholder="tt('Find category')" :filter-no-items-text="tt('No available category')"
                                   :items="allTransactionCategories"
                                   v-model:show="showRowCategorySheet"
                                   v-model="selectedRowCategoryId">
        </tree-view-selection-sheet>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, useTemplateRef } from 'vue';
import type { Router } from 'framework7/types';

import MapView from '@/components/common/MapView.vue';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';

import { useReceiptRecognitionConfirmBase, type ReceiptRecognitionConfirmRow } from '@/views/base/transactions/ReceiptRecognitionConfirmBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';

import { CategoryType } from '@/core/category.ts';
import { TransactionType } from '@/core/transaction.ts';
import { TRANSACTION_MIN_AMOUNT, TRANSACTION_MAX_AMOUNT } from '@/consts/transaction.ts';

import type { CategorizedAccountWithDisplayBalance } from '@/models/account.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';

import { parseDateTimeFromUnixTimeWithBrowserTimezone } from '@/lib/datetime.ts';
import { getMapProvider } from '@/lib/server_settings.ts';

type MapViewType = InstanceType<typeof MapView>;

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const {
    tt,
    formatAmountToLocalizedNumeralsWithCurrency,
    formatDateTimeToLongDateTime,
    getCategorizedAccountsWithDisplayBalance
} = useI18n();
const { showPrompt, showToast, routeBackOnError } = useI18nUIComponents();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();

const {
    receiptImageFile,
    rows,
    receiptTotal,
    mode,
    attachGeoLocation,
    accountId,
    type,
    receiptCategoryId,
    receiptComment,
    time,
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
    initFromPendingResult,
    updateRowAmount,
    updateRowCategory,
    updateRowComment,
    setRowInclude,
    updateReceiptTotal,
    distributeDifference,
    getRowDisplayComment,
    confirm,
    retryFailedTransactions,
    getRegistrationResultMessage
} = useReceiptRecognitionConfirmBase();

const map = useTemplateRef<MapViewType>('map');

const mapProviderAvailable: boolean = !!getMapProvider();

const loading = ref<boolean>(false);
const loadingError = ref<unknown>(null);
const receiptImageSrc = ref<string | undefined>(undefined);
const selectedRowId = ref<string | null>(null);
const showAccountSheet = ref<boolean>(false);
const showReceiptTotalSheet = ref<boolean>(false);
const showReceiptCategorySheet = ref<boolean>(false);
const showRowActionSheet = ref<boolean>(false);
const showRowAmountSheet = ref<boolean>(false);
const showRowCategorySheet = ref<boolean>(false);

const allVisibleCategorizedAccounts = computed<CategorizedAccountWithDisplayBalance[]>(() => getCategorizedAccountsWithDisplayBalance(accountsStore.allVisiblePlainAccounts, settingsStore.appSettings.showAccountBalance, settingsStore.appSettings.accountCategoryOrders));

const categoryType = computed<number>(() => {
    if (type.value === TransactionType.Income) {
        return CategoryType.Income;
    } else if (type.value === TransactionType.Transfer) {
        return CategoryType.Transfer;
    }

    return CategoryType.Expense;
});

const allTransactionCategories = computed<TransactionCategory[]>(() => transactionCategoriesStore.allTransactionCategories[categoryType.value] || []);

const sourceAccountName = computed<string>(() => accountsStore.allAccountsMap[accountId.value]?.name ?? tt('None'));
const sourceAccountCurrency = computed<string>(() => accountsStore.allAccountsMap[accountId.value]?.currency ?? userStore.currentUserDefaultCurrency);

const displayTransactionTime = computed<string>(() => time.value ? formatDateTimeToLongDateTime(parseDateTimeFromUnixTimeWithBrowserTimezone(time.value)) : '');

const selectedRow = computed<ReceiptRecognitionConfirmRow | null>(() => rows.value.find(row => row.id === selectedRowId.value) ?? null);

// row amount / receipt total edits must go through the invariant engine
// (updateRowAmount / updateReceiptTotal) so the adjustment row keeps following
const selectedRowAmount = computed<number>({
    get: () => selectedRow.value?.amount ?? 0,
    set: (value: number) => {
        if (selectedRowId.value) {
            updateRowAmount(selectedRowId.value, value);
        }
    }
});

const selectedRowCategoryId = computed<string>({
    get: () => selectedRow.value?.categoryId ?? '',
    set: (value: string) => {
        if (selectedRowId.value) {
            updateRowCategory(selectedRowId.value, value);
        }
    }
});

const editableReceiptTotal = computed<number>({
    get: () => receiptTotal.value,
    set: (value: number) => updateReceiptTotal(value)
});

function getDisplayAmount(amount: number): string {
    return formatAmountToLocalizedNumeralsWithCurrency(amount, sourceAccountCurrency.value);
}

function getCategoryDisplayName(categoryId: string): string {
    if (!categoryId) {
        return tt('None');
    }

    return transactionCategoriesStore.allTransactionCategoriesMap[categoryId]?.name ?? tt('None');
}

function onRowClick(row: ReceiptRecognitionConfirmRow): void {
    // already saved rows are immutable (same as the desktop dialog): editing them
    // would break the registered total without changing the saved transactions
    if (registering.value || row.saveStatus === 'saved') {
        return;
    }

    selectedRowId.value = row.id;
    showRowActionSheet.value = true;
}

function onRowIncludeChanged(row: ReceiptRecognitionConfirmRow): void {
    setRowInclude(row.id, !row.include);
}

function editSelectedRowComment(): void {
    const row = selectedRow.value;

    if (!row || row.isAdjustment) {
        return;
    }

    const rowId = row.id;

    showPrompt('Description', row.comment, (value: string) => {
        updateRowComment(rowId, value);
    });
}

function save(): void {
    confirm().then(result => {
        showToast(getRegistrationResultMessage(result));

        if (result.allSaved) {
            props.f7router.back();
        }
    }).catch(error => {
        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function retry(): void {
    retryFailedTransactions().then(result => {
        showToast(getRegistrationResultMessage(result));

        if (result.allSaved) {
            props.f7router.back();
        }
    }).catch(error => {
        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function onPageAfterIn(): void {
    map.value?.initMapView();
}

function init(): void {
    if (!initFromPendingResult()) {
        loadingError.value = 'Parameter Invalid';
        return;
    }

    if (receiptImageFile.value) {
        receiptImageSrc.value = URL.createObjectURL(receiptImageFile.value);
    }

    loading.value = true;

    Promise.all([
        accountsStore.loadAllAccounts({ force: false }),
        transactionCategoriesStore.loadAllCategories({ force: false })
    ]).then(() => {
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

onUnmounted(() => {
    if (receiptImageSrc.value) {
        URL.revokeObjectURL(receiptImageSrc.value);
        receiptImageSrc.value = undefined;
    }
});

routeBackOnError(props.f7router, loadingError);
init();
</script>

<style>
.receipt-recognition-confirm-thumbnail {
    width: 40px;
    height: 56px;
    object-fit: cover;
    border-radius: 4px;
}

.receipt-recognition-confirm-adjustment-icon {
    font-size: 16px;
    margin-inline-end: 4px;
}

.receipt-recognition-confirm-row-status {
    margin-inline-start: 6px;

    &.f7-icons {
        font-size: 18px;
    }
}

.receipt-recognition-confirm-map .map-view-container {
    width: 100%;
    border-radius: var(--f7-list-inset-border-radius);
    overflow: hidden;
}

.receipt-recognition-confirm-mismatch {
    color: var(--f7-color-orange);
    font-size: var(--ebk-large-footer-font-size);
}
</style>
