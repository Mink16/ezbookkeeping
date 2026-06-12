import { ref, computed, watch } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useCustomReceiptRecognitionStore } from '@/stores/customReceiptRecognition.ts';

import { itemAndIndex } from '@/core/base.ts';
import type { Coordinate } from '@/core/coordinate.ts';
import { KnownFileType } from '@/core/file.ts';
import { ImageUploadQualityType } from '@/core/image.ts';
import { TransactionType } from '@/core/transaction.ts';

import type {
    RecognizedReceiptDetailsResponse,
    RecognizedReceiptLocationResponse,
    RecognizedReceiptAdjustmentKind,
    PendingReceiptRecognitionResult,
    ReceiptTransactionSaveStatus,
    ReceiptRegistrationMode,
    ReceiptTransactionCandidate,
    ReceiptRegistrationCommonOptions,
    ReceiptRegistrationResult
} from '@/models/custom_receipt_recognition.ts';

import {
    distributeAmountProportionally,
    calculateAdjustment,
    getDefaultAttachGeoLocation
} from '@/lib/custom_receipt_recognition.ts';

import {
    getTimezoneOffsetMinutes,
    getSameDateTimeWithCurrentTimezone,
    parseDateTimeFromUnixTimeWithBrowserTimezone,
    getCurrentUnixTime
} from '@/lib/datetime.ts';

import { compressJpgImageByQuality } from '@/lib/ui/common.ts';

import logger from '@/lib/logger.ts';

export const COMBINED_TRANSACTION_CANDIDATE_ID: string = 'receipt-combined-transaction';

export interface ReceiptRecognitionConfirmRow {
    id: string;
    include: boolean;
    amount: number; // amount (x100), negative for discount lines
    categoryId: string;
    comment: string;
    isAdjustment: boolean;
    adjustmentKind?: RecognizedReceiptAdjustmentKind;
    saveStatus: ReceiptTransactionSaveStatus;
}

export function useReceiptRecognitionConfirmBase() {
    const { tt } = useI18n();

    const settingsStore = useSettingsStore();
    const customReceiptRecognitionStore = useCustomReceiptRecognitionStore();

    const recognizedResponse = ref<RecognizedReceiptDetailsResponse | null>(null);
    const receiptImageFile = ref<File | null>(null);

    const rows = ref<ReceiptRecognitionConfirmRow[]>([]);
    const receiptTotal = ref<number>(0);
    const mode = ref<ReceiptRegistrationMode>('combined');
    const attachGeoLocation = ref<boolean>(false);
    const accountId = ref<string>('');
    const type = ref<number>(TransactionType.Expense);
    const time = ref<number>(0);
    const tagIds = ref<string[]>([]);
    const receiptCategoryId = ref<string>('');
    const receiptComment = ref<string>('');

    let adjustmentRowSequence = 0;
    let lastCandidates: ReceiptTransactionCandidate[] | null = null;
    let lastOptions: ReceiptRegistrationCommonOptions | null = null;

    const imageUploadQualityType = computed<ImageUploadQualityType>(() => ImageUploadQualityType.valueOf(settingsStore.appSettings.transactionPictureQuality) ?? ImageUploadQualityType.Default);

    const recognizedGeoLocation = computed<Coordinate | null>(() => recognizedResponse.value?.geoLocation ?? null);
    const resolvedLocation = computed<RecognizedReceiptLocationResponse | null>(() => recognizedResponse.value?.location ?? null);
    const hasResolvedLocation = computed<boolean>(() => !!recognizedGeoLocation.value && !!resolvedLocation.value);

    const includedRows = computed<ReceiptRecognitionConfirmRow[]>(() => rows.value.filter(row => row.include));
    const includedNormalTotal = computed<number>(() => rows.value.reduce((total, row) => row.include && !row.isAdjustment ? total + row.amount : total, 0));
    const includedTotal = computed<number>(() => rows.value.reduce((total, row) => row.include ? total + row.amount : total, 0));
    const adjustmentRow = computed<ReceiptRecognitionConfirmRow | null>(() => rows.value.find(row => row.isAdjustment) ?? null);
    const hasAdjustmentRow = computed<boolean>(() => !!adjustmentRow.value);
    const adjustmentAmount = computed<number>(() => adjustmentRow.value?.amount ?? 0);
    const excludedRowCount = computed<number>(() => rows.value.reduce((count, row) => row.include ? count : count + 1, 0));
    const hasSelectionMismatch = computed<boolean>(() => includedTotal.value !== receiptTotal.value);
    // already saved rows must not be modified anymore (their transactions are in the database),
    // so they are excluded from the distribution targets and a saved adjustment row cannot be distributed
    const canDistributeDifference = computed<boolean>(() => !!adjustmentRow.value && adjustmentRow.value.saveStatus !== 'saved' && rows.value.some(row => row.include && !row.isAdjustment && row.saveStatus !== 'saved' && row.amount > 0));

    const registering = computed<boolean>(() => customReceiptRecognitionStore.registering);
    const combinedSaveStatus = computed<ReceiptTransactionSaveStatus>(() => customReceiptRecognitionStore.getTransactionSaveStatus(COMBINED_TRANSACTION_CANDIDATE_ID));
    const savedRowCount = computed<number>(() => rows.value.reduce((count, row) => row.saveStatus === 'saved' ? count + 1 : count, 0));
    const failedRowCount = computed<number>(() => rows.value.reduce((count, row) => row.saveStatus === 'failed' ? count + 1 : count, 0));
    const hasFailedTransactions = computed<boolean>(() => {
        if (mode.value === 'combined') {
            return combinedSaveStatus.value === 'failed';
        }

        return failedRowCount.value > 0;
    });
    const canRetryFailedTransactions = computed<boolean>(() => !registering.value && hasFailedTransactions.value && !!lastCandidates && !!lastOptions);

    const canConfirm = computed<boolean>(() => {
        if (registering.value || !accountId.value) {
            return false;
        }

        if (mode.value === 'combined') {
            // once a row transaction has been saved in separate mode, registering the
            // combined receipt total on top of it would double count the receipt
            if (savedRowCount.value > 0) {
                return false;
            }

            return !!receiptCategoryId.value;
        }

        if (includedRows.value.length < 1) {
            return false;
        }

        for (const row of includedRows.value) {
            if (!row.categoryId) {
                return false;
            }
        }

        return true;
    });

    function getDefaultTransactionTime(): number {
        return getSameDateTimeWithCurrentTimezone(parseDateTimeFromUnixTimeWithBrowserTimezone(getCurrentUnixTime())).getUnixTime();
    }

    function init(result: PendingReceiptRecognitionResult): void {
        const response = result.response;

        recognizedResponse.value = response;
        receiptImageFile.value = result.imageFile;

        const items = response.items ?? [];
        const newRows: ReceiptRecognitionConfirmRow[] = [];

        for (const [item, index] of itemAndIndex(items)) {
            newRows.push({
                id: `receipt-item-${index}`,
                include: true,
                amount: item.amount ?? 0, // defensive: an undefined amount would poison every total with NaN
                categoryId: item.categoryId ?? '',
                comment: item.comment ?? '',
                isAdjustment: !!item.isAdjustment,
                adjustmentKind: item.adjustmentKind,
                saveStatus: 'pending'
            });
        }

        rows.value = newRows;
        receiptTotal.value = response.sourceAmount ?? includedNormalTotal.value + adjustmentAmount.value;
        mode.value = 'combined';
        attachGeoLocation.value = getDefaultAttachGeoLocation(response);
        accountId.value = response.sourceAccountId ?? '';
        type.value = response.type;
        time.value = response.time ?? getDefaultTransactionTime();
        tagIds.value = response.tagIds ? response.tagIds.slice() : [];
        receiptCategoryId.value = response.categoryId ?? '';
        receiptComment.value = response.comment ?? '';

        adjustmentRowSequence = 0;
        lastCandidates = null;
        lastOptions = null;

        customReceiptRecognitionStore.resetRegistrationState();
    }

    function initFromPendingResult(): boolean {
        const result = customReceiptRecognitionStore.consumePendingResult();

        if (!result) {
            return false;
        }

        init(result);
        return true;
    }

    function findRow(id: string): ReceiptRecognitionConfirmRow | null {
        return rows.value.find(row => row.id === id) ?? null;
    }

    // invariant engine (spec §4.4 layer 3): the adjustment row follows amount /
    // total edits so that Σ(included rows) === receiptTotal keeps holding
    function recalculateAdjustment(): void {
        const delta = calculateAdjustment(receiptTotal.value, rows.value.filter(row => row.include && !row.isAdjustment).map(row => row.amount));
        const currentAdjustmentRow = rows.value.find(row => row.isAdjustment);

        if (delta === 0) {
            if (currentAdjustmentRow) {
                rows.value = rows.value.filter(row => !row.isAdjustment);
            }
        } else if (currentAdjustmentRow) {
            currentAdjustmentRow.amount = delta;
            currentAdjustmentRow.include = true;
        } else {
            rows.value.push({
                id: `receipt-adjustment-${adjustmentRowSequence++}`,
                include: true,
                amount: delta,
                categoryId: '',
                comment: '',
                isAdjustment: true,
                adjustmentKind: 'unknown',
                saveStatus: 'pending'
            });
        }
    }

    // already saved rows are immutable: their transactions are in the database, so editing or
    // excluding them would silently break the registered total or allow duplicate registration
    function updateRowAmount(id: string, amount: number): void {
        const row = findRow(id);

        if (!row || row.isAdjustment || row.saveStatus === 'saved') {
            return;
        }

        row.amount = amount;
        recalculateAdjustment();
    }

    function updateRowCategory(id: string, categoryId: string): void {
        const row = findRow(id);

        if (row && row.saveStatus !== 'saved') {
            row.categoryId = categoryId;
        }
    }

    function updateRowComment(id: string, comment: string): void {
        const row = findRow(id);

        if (row && !row.isAdjustment && row.saveStatus !== 'saved') {
            row.comment = comment;
        }
    }

    // intentionally does NOT trigger the adjustment recalculation: excluding a
    // row means the user wants to register less than the receipt total, which
    // is surfaced via the hasSelectionMismatch badge instead (spec §4.4)
    function setRowInclude(id: string, include: boolean): void {
        const row = findRow(id);

        if (row && row.saveStatus !== 'saved') {
            row.include = include;
        }
    }

    function updateReceiptTotal(total: number): void {
        receiptTotal.value = total;
        recalculateAdjustment();
    }

    // distributes the adjustment row amount to the included positive normal
    // rows with the largest remainder method, then removes the adjustment row;
    // the sum invariant strictly holds afterwards (integer arithmetic only).
    // already saved rows are never modified (their transactions are in the
    // database), so the difference only goes to rows that will still be saved
    function distributeDifference(): void {
        const currentAdjustmentRow = rows.value.find(row => row.isAdjustment);

        if (!currentAdjustmentRow || currentAdjustmentRow.saveStatus === 'saved') {
            return;
        }

        const targetRows = rows.value.filter(row => row.include && !row.isAdjustment && row.saveStatus !== 'saved');
        const weights = targetRows.map(row => row.amount > 0 ? row.amount : 0);

        if (!weights.some(weight => weight > 0)) {
            return;
        }

        const shares = distributeAmountProportionally(currentAdjustmentRow.amount, weights);

        for (const [row, index] of itemAndIndex(targetRows)) {
            row.amount += shares[index] ?? 0;
        }

        rows.value = rows.value.filter(row => !row.isAdjustment);
    }

    function getAdjustmentDisplayName(adjustmentKind?: RecognizedReceiptAdjustmentKind): string {
        if (adjustmentKind === 'tax') {
            return tt('Consumption Tax');
        }

        return tt('Adjustment');
    }

    function getRowDisplayComment(row: ReceiptRecognitionConfirmRow): string {
        if (row.comment) {
            return row.comment;
        }

        if (row.isAdjustment) {
            return getAdjustmentDisplayName(row.adjustmentKind);
        }

        return '';
    }

    function buildCandidates(): ReceiptTransactionCandidate[] {
        if (mode.value === 'combined') {
            return [{
                id: COMBINED_TRANSACTION_CANDIDATE_ID,
                amount: receiptTotal.value,
                categoryId: receiptCategoryId.value,
                comment: receiptComment.value,
                tagIds: tagIds.value.slice()
            }];
        }

        return includedRows.value.map(row => ({
            id: row.id,
            amount: row.amount,
            categoryId: row.categoryId,
            comment: getRowDisplayComment(row),
            tagIds: tagIds.value.slice()
        }));
    }

    async function buildCommonOptions(): Promise<ReceiptRegistrationCommonOptions> {
        const timeZone = settingsStore.appSettings.timeZone;
        const geoLocation = attachGeoLocation.value && recognizedGeoLocation.value ? {
            latitude: recognizedGeoLocation.value.latitude,
            longitude: recognizedGeoLocation.value.longitude
        } : null;

        let compressedImageFile: File | undefined = undefined;

        if (settingsStore.appSettings.autoUploadTransactionPictureForAIRecognition && receiptImageFile.value) {
            try {
                const blob = await compressJpgImageByQuality(receiptImageFile.value, imageUploadQualityType.value);
                compressedImageFile = KnownFileType.JPG.createFileFromBlob(blob, 'image');
            } catch (error) {
                // the receipt picture is best-effort; registration proceeds without it
                logger.error('failed to compress receipt picture for recognized transactions', error);
            }
        }

        return {
            type: type.value,
            time: time.value,
            timeZone: timeZone,
            utcOffset: getTimezoneOffsetMinutes(time.value, timeZone),
            sourceAccountId: accountId.value,
            geoLocation: geoLocation,
            receiptImageFile: compressedImageFile
        };
    }

    function confirm(): Promise<ReceiptRegistrationResult> {
        if (!canConfirm.value) {
            return Promise.reject({ message: 'An error occurred' });
        }

        if (mode.value === 'separate' && excludedRowCount.value === 0 && hasSelectionMismatch.value) {
            // must not happen by construction (spec §4.4): the adjustment row
            // keeps the invariant whenever no row is excluded
            logger.error('receipt recognition sum invariant is broken without excluded rows, this is a bug');
            return Promise.reject({ message: 'An error occurred' });
        }

        return buildCommonOptions().then(options => {
            const candidates = buildCandidates();

            lastCandidates = candidates;
            lastOptions = options;

            return customReceiptRecognitionStore.registerRecognizedTransactions({ candidates, options });
        });
    }

    function retryFailedTransactions(): Promise<ReceiptRegistrationResult> {
        if (!lastCandidates || !lastOptions || registering.value) {
            return Promise.reject({ message: 'An error occurred' });
        }

        return customReceiptRecognitionStore.registerRecognizedTransactions({
            candidates: lastCandidates,
            options: lastOptions
        });
    }

    function getRegistrationResultMessage(result: ReceiptRegistrationResult): string {
        return result.allSaved ? 'All transactions have been saved' : 'Unable to save some transactions';
    }

    // keep row save statuses in sync with the store while registering
    watch(() => customReceiptRecognitionStore.transactionSaveStatuses, () => {
        for (const row of rows.value) {
            row.saveStatus = customReceiptRecognitionStore.getTransactionSaveStatus(row.id);
        }
    }, { deep: true });

    return {
        // constants
        COMBINED_TRANSACTION_CANDIDATE_ID,
        // states
        recognizedResponse,
        receiptImageFile,
        rows,
        receiptTotal,
        mode,
        attachGeoLocation,
        accountId,
        type,
        time,
        tagIds,
        receiptCategoryId,
        receiptComment,
        // computed states
        recognizedGeoLocation,
        resolvedLocation,
        hasResolvedLocation,
        includedRows,
        includedNormalTotal,
        includedTotal,
        adjustmentRow,
        hasAdjustmentRow,
        adjustmentAmount,
        excludedRowCount,
        hasSelectionMismatch,
        canDistributeDifference,
        registering,
        combinedSaveStatus,
        savedRowCount,
        failedRowCount,
        hasFailedTransactions,
        canRetryFailedTransactions,
        canConfirm,
        // functions
        init,
        initFromPendingResult,
        recalculateAdjustment,
        updateRowAmount,
        updateRowCategory,
        updateRowComment,
        setRowInclude,
        updateReceiptTotal,
        distributeDifference,
        getAdjustmentDisplayName,
        getRowDisplayComment,
        confirm,
        retryFailedTransactions,
        getRegistrationResultMessage
    };
}
