import { TransactionType } from '@/core/transaction.ts';

import type {
    RecognizedReceiptDetailsResponse,
    RecognizedReceiptLocationResponse
} from '@/models/custom_receipt_recognition.ts';

/**
 * Distributes the specified delta (integer amount, x100) to the specified rows
 * proportionally to their weights using the largest remainder method.
 *
 * - All calculation is done with integer arithmetic, so the sum of the returned
 *   shares is exactly equal to delta (no rounding error), as long as at least
 *   one weight is positive.
 * - Rows with non-positive weights never receive any share.
 * - Negative delta is supported (the sign is separated and the same procedure
 *   is applied to the absolute value).
 * - If there is no positive weight, an all-zero array is returned (the caller
 *   should not rely on the sum invariant in that case).
 */
export function distributeAmountProportionally(delta: number, weights: number[]): number[] {
    const shares: number[] = new Array<number>(weights.length).fill(0);

    if (weights.length < 1 || delta === 0 || !Number.isInteger(delta)) {
        return shares;
    }

    let totalWeight = 0;

    for (const weight of weights) {
        if (Number.isInteger(weight) && weight > 0) {
            totalWeight += weight;
        }
    }

    if (totalWeight <= 0) {
        return shares;
    }

    const sign = delta < 0 ? -1 : 1;
    const absDelta = Math.abs(delta);

    const remainders: { index: number, remainder: number }[] = [];
    let distributed = 0;

    for (let i = 0; i < weights.length; i++) {
        const weight = weights[i] as number;

        if (!Number.isInteger(weight) || weight <= 0) {
            continue;
        }

        const scaled = absDelta * weight;
        const share = Math.floor(scaled / totalWeight);

        shares[i] = share;
        distributed += share;
        remainders.push({ index: i, remainder: scaled % totalWeight });
    }

    let remaining = absDelta - distributed; // 0 <= remaining < count of positive-weight rows

    remainders.sort((a, b) => {
        if (a.remainder !== b.remainder) {
            return b.remainder - a.remainder; // larger remainder first
        }

        return a.index - b.index; // stable: earlier row first on ties
    });

    for (let i = 0; i < remainders.length && remaining > 0; i++) {
        const item = remainders[i] as { index: number, remainder: number };
        shares[item.index] = (shares[item.index] as number) + 1;
        remaining--;
    }

    if (sign < 0) {
        for (let i = 0; i < shares.length; i++) {
            shares[i] = -(shares[i] as number);
        }
    }

    return shares;
}

/**
 * Returns the adjustment amount required to make the included item amounts sum
 * up to the specified total (total - sum of item amounts).
 */
export function calculateAdjustment(total: number, includedItemAmounts: number[]): number {
    let sum = 0;

    for (const amount of includedItemAmounts) {
        sum += amount;
    }

    return total - sum;
}

/**
 * Returns whether the recognition confirmation screen should be shown for the
 * specified response. The confirmation flow only supports expense receipts:
 * income / transfer results (a transfer needs a destination account, which the
 * confirmation screen does not handle) and responses with less than two items
 * fall back to the legacy single-transaction flow (edit page prefill).
 */
export function shouldShowReceiptConfirmation(response: RecognizedReceiptDetailsResponse): boolean {
    return response.type === TransactionType.Expense && !!response.items && response.items.length >= 2;
}

/**
 * Returns the default state of the "attach geolocation" toggle: enabled only
 * when a coordinate was resolved with high confidence.
 */
export function getDefaultAttachGeoLocation(response: RecognizedReceiptDetailsResponse): boolean {
    return !!response.geoLocation && isHighConfidenceLocation(response.location);
}

export function isHighConfidenceLocation(location?: RecognizedReceiptLocationResponse): boolean {
    return !!location && location.confidence === 'high';
}
