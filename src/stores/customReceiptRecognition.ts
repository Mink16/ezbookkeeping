import { ref } from 'vue';
import { defineStore } from 'pinia';

import { useUserStore } from './user.ts';
import { useTransactionsStore } from './transaction.ts';

import { TransactionType } from '@/core/transaction.ts';

import type {
    RecognizedReceiptDetailsResponse,
    PendingReceiptRecognitionResult,
    ReceiptTransactionSaveStatus,
    ReceiptTransactionCandidate,
    ReceiptRegistrationCommonOptions,
    ReceiptRegistrationResult
} from '@/models/custom_receipt_recognition.ts';
import type { TransactionPictureInfoBasicResponse } from '@/models/transaction_picture_info.ts';
import { Transaction } from '@/models/transaction.ts';

import { generateRandomUUID } from '@/lib/misc.ts';
import logger from '@/lib/logger.ts';
import services from '@/lib/services.ts';

export const useCustomReceiptRecognitionStore = defineStore('customReceiptRecognition', () => {
    const userStore = useUserStore();
    const transactionsStore = useTransactionsStore();

    // handoff payload for the mobile page navigation (URL query cannot carry arrays)
    const pendingResult = ref<PendingReceiptRecognitionResult | null>(null);

    // per-row save statuses, keyed by candidate id, updated live while registering
    const transactionSaveStatuses = ref<Record<string, ReceiptTransactionSaveStatus>>({});
    const registering = ref<boolean>(false);

    // the receipt picture is uploaded at most once and attached to the first successfully saved transaction only
    let uploadedPictureInfo: TransactionPictureInfoBasicResponse | null = null;
    let pictureAttached: boolean = false;

    function setPendingResult(result: PendingReceiptRecognitionResult | null): void {
        pendingResult.value = result;
    }

    function consumePendingResult(): PendingReceiptRecognitionResult | null {
        const result = pendingResult.value;
        pendingResult.value = null;
        return result;
    }

    function clearPendingResult(): void {
        pendingResult.value = null;
    }

    function resetRegistrationState(): void {
        transactionSaveStatuses.value = {};
        registering.value = false;
        uploadedPictureInfo = null;
        pictureAttached = false;
    }

    function getTransactionSaveStatus(id: string): ReceiptTransactionSaveStatus {
        return transactionSaveStatuses.value[id] ?? 'pending';
    }

    function recognizeReceiptImageDetails({ imageFile, cancelableUuid }: { imageFile: File, cancelableUuid?: string }): Promise<RecognizedReceiptDetailsResponse> {
        return new Promise((resolve, reject) => {
            services.recognizeReceiptImageDetails({ imageFile, cancelableUuid }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to recognize image' });
                    return;
                }

                resolve(data.result);
            }).catch(error => {
                if (error.canceled) {
                    reject(error);
                }

                logger.error('failed to recognize image', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to recognize image' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function cancelRecognizeReceiptImageDetails(cancelableUuid: string): void {
        services.cancelRequest(cancelableUuid);
    }

    function createTransactionFromCandidate(candidate: ReceiptTransactionCandidate, options: ReceiptRegistrationCommonOptions): Transaction {
        const transaction = Transaction.createNewTransaction(options.type, options.time, options.timeZone, options.utcOffset);

        transaction.sourceAccountId = options.sourceAccountId;
        transaction.sourceAmount = candidate.amount;

        if (options.type === TransactionType.Transfer) {
            transaction.destinationAmount = candidate.amount;
        }

        transaction.setCategoryId(candidate.categoryId);
        transaction.tagIds = candidate.tagIds.slice();
        transaction.comment = candidate.comment;

        if (options.geoLocation) {
            transaction.setGeoLocation(options.geoLocation);
        }

        return transaction;
    }

    async function registerRecognizedTransactions({ candidates, options }: { candidates: ReceiptTransactionCandidate[], options: ReceiptRegistrationCommonOptions }): Promise<ReceiptRegistrationResult> {
        if (registering.value) {
            return Promise.reject({ message: 'An error occurred' });
        }

        registering.value = true;

        const newStatuses: Record<string, ReceiptTransactionSaveStatus> = {};

        for (const candidate of candidates) {
            newStatuses[candidate.id] = transactionSaveStatuses.value[candidate.id] === 'saved' ? 'saved' : 'pending';
        }

        transactionSaveStatuses.value = newStatuses;

        try {
            if (options.receiptImageFile && !uploadedPictureInfo && !pictureAttached) {
                try {
                    uploadedPictureInfo = await transactionsStore.uploadTransactionPicture({
                        pictureFile: options.receiptImageFile
                    });
                } catch (error) {
                    // the receipt picture is best-effort; transactions are still registered without it
                    logger.error('failed to upload receipt picture for recognized transactions', error);
                }
            }

            let savedCount = 0;
            let failedCount = 0;

            for (const candidate of candidates) {
                if (transactionSaveStatuses.value[candidate.id] === 'saved') {
                    savedCount++;
                    continue;
                }

                transactionSaveStatuses.value[candidate.id] = 'saving';

                const transaction = createTransactionFromCandidate(candidate, options);
                const attachPicture = !!uploadedPictureInfo && !pictureAttached;

                if (attachPicture && uploadedPictureInfo) {
                    transaction.addPicture(uploadedPictureInfo);
                }

                try {
                    await transactionsStore.saveTransaction({
                        transaction: transaction,
                        defaultCurrency: userStore.currentUserDefaultCurrency,
                        isEdit: false,
                        clientSessionId: generateRandomUUID()
                    });

                    transactionSaveStatuses.value[candidate.id] = 'saved';
                    savedCount++;

                    if (attachPicture) {
                        pictureAttached = true;
                    }
                } catch (error) {
                    logger.error('failed to save recognized receipt transaction', error);
                    transactionSaveStatuses.value[candidate.id] = 'failed';
                    failedCount++;
                }
            }

            return {
                savedCount: savedCount,
                failedCount: failedCount,
                allSaved: failedCount === 0
            };
        } finally {
            registering.value = false;
        }
    }

    return {
        // states
        pendingResult,
        transactionSaveStatuses,
        registering,
        // functions
        setPendingResult,
        consumePendingResult,
        clearPendingResult,
        resetRegistrationState,
        getTransactionSaveStatus,
        recognizeReceiptImageDetails,
        cancelRecognizeReceiptImageDetails,
        registerRecognizedTransactions
    };
});
