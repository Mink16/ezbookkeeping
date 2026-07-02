import type { Coordinate } from '@/core/coordinate.ts';

import type { RecognizedTransactionResponse } from './large_language_model.ts';
import type { TransactionGeoLocationResponse } from './transaction.ts';

export type RecognizedReceiptAdjustmentKind = 'tax' | 'unknown';

export type RecognizedReceiptLocationConfidence = 'high' | 'low';

export interface RecognizedReceiptItemResponse {
    readonly categoryId?: string;
    readonly amount: number; // amount (x100), negative for discount lines
    readonly comment?: string; // item name
    readonly isAdjustment?: boolean;
    readonly adjustmentKind?: RecognizedReceiptAdjustmentKind;
}

export interface RecognizedReceiptLocationResponse {
    readonly displayName: string;
    readonly confidence: RecognizedReceiptLocationConfidence;
    readonly provider?: string;
}

export interface RecognizedReceiptDetailsResponse extends RecognizedTransactionResponse {
    readonly items?: RecognizedReceiptItemResponse[];
    readonly geoLocation?: TransactionGeoLocationResponse;
    readonly location?: RecognizedReceiptLocationResponse;
}

export interface PendingReceiptRecognitionResult {
    readonly response: RecognizedReceiptDetailsResponse;
    readonly imageFile: File;
}

export type ReceiptTransactionSaveStatus = 'pending' | 'saving' | 'saved' | 'failed';

export type ReceiptRegistrationMode = 'combined' | 'separate';

export interface ReceiptTransactionCandidate {
    readonly id: string; // stable row id used for per-row save status tracking
    readonly amount: number; // amount (x100)
    readonly categoryId: string;
    readonly comment: string;
    readonly tagIds: string[];
}

export interface ReceiptRegistrationCommonOptions {
    readonly type: number; // TransactionType
    readonly time: number; // unix time
    readonly timeZone: string;
    readonly utcOffset: number; // minutes
    readonly sourceAccountId: string;
    readonly geoLocation: Coordinate | null;
    readonly receiptImageFile?: File; // already compressed, attached to the first successfully saved transaction only
}

export interface ReceiptRegistrationResult {
    readonly savedCount: number;
    readonly failedCount: number;
    readonly allSaved: boolean;
}
