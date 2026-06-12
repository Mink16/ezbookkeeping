import { describe, expect, test } from 'vitest';

import { TransactionType } from '@/core/transaction.ts';

import type { RecognizedReceiptDetailsResponse } from '@/models/custom_receipt_recognition.ts';

import {
    distributeAmountProportionally,
    calculateAdjustment,
    shouldShowReceiptConfirmation,
    getDefaultAttachGeoLocation,
    isHighConfidenceLocation
} from '@/lib/custom_receipt_recognition.ts';

function sum(values: number[]): number {
    return values.reduce((total, value) => total + value, 0);
}

describe('distributeAmountProportionally', () => {
    const testCases: {
        name: string;
        delta: number;
        weights: number[];
        expected: number[];
    }[] = [
        {
            name: 'zero delta distributes nothing',
            delta: 0,
            weights: [100, 200, 300],
            expected: [0, 0, 0]
        },
        {
            name: 'exact proportional split without remainder',
            delta: 600,
            weights: [100, 200, 300],
            expected: [100, 200, 300]
        },
        {
            name: 'remainder goes to rows with largest fractional part',
            // shares: floor(10*1/6)=1 rem 4, floor(10*2/6)=3 rem 2, floor(10*3/6)=5 rem 0
            // remaining 1 unit goes to row 0 (largest remainder)
            delta: 10,
            weights: [1, 2, 3],
            expected: [2, 3, 5]
        },
        {
            name: 'one minimum currency unit goes to the largest remainder row',
            // shares: floor(1*w/W)=0 for all; remainders are w_i, so the largest weight wins
            delta: 1,
            weights: [100, 250, 150],
            expected: [0, 1, 0]
        },
        {
            name: 'all rows with the same weight, remainder assigned from the first row',
            // floor(5*1/3)=1 each (3 distributed), remaining 2 to rows 0 and 1
            delta: 5,
            weights: [100, 100, 100],
            expected: [2, 2, 1]
        },
        {
            name: 'negative delta mirrors the positive distribution',
            delta: -10,
            weights: [1, 2, 3],
            expected: [-2, -3, -5]
        },
        {
            name: 'negative delta with same weights',
            delta: -5,
            weights: [100, 100, 100],
            expected: [-2, -2, -1]
        },
        {
            name: 'rows with zero weight never receive a share',
            delta: 10,
            weights: [100, 0, 100],
            expected: [5, 0, 5]
        },
        {
            name: 'rows with negative weight never receive a share',
            // negative rows (discount lines) are excluded from both the weight sum and the distribution
            delta: 10,
            weights: [100, -50, 100],
            expected: [5, 0, 5]
        },
        {
            name: 'single positive row receives the whole delta',
            delta: 123,
            weights: [999],
            expected: [123]
        },
        {
            name: 'no positive weight returns all zeros',
            delta: 10,
            weights: [0, -100],
            expected: [0, 0]
        },
        {
            name: 'empty weights returns empty array',
            delta: 10,
            weights: [],
            expected: []
        },
        {
            name: 'non-integer delta distributes nothing',
            delta: 10.5,
            weights: [100, 200],
            expected: [0, 0]
        },
        {
            name: 'typical receipt tax distribution (8% on x100 amounts)',
            // 8% tax of 1,000 yen (=100000) is 8000; weights are item prices
            delta: 8000,
            weights: [39800, 29800, 30400],
            expected: [3184, 2384, 2432]
        }
    ];

    testCases.forEach(testCase => {
        test(testCase.name, () => {
            const actual = distributeAmountProportionally(testCase.delta, testCase.weights);
            expect(actual).toStrictEqual(testCase.expected);
        });
    });

    test('sum of shares always strictly equals delta when a positive weight exists', () => {
        const sumInvariantCases: { delta: number, weights: number[] }[] = [
            { delta: 1, weights: [1] },
            { delta: 1, weights: [3, 3, 3] },
            { delta: 7, weights: [1, 1, 1] },
            { delta: 97, weights: [13, 17, 19, 23] },
            { delta: 1000, weights: [333, 333, 334] },
            { delta: 999983, weights: [1, 999999] },
            { delta: -97, weights: [13, 17, 19, 23] },
            { delta: -1, weights: [100, 100] },
            { delta: 1234567, weights: [98765, 43210, 11111, 0, -5] },
            { delta: 3, weights: [1000000, 1, 1] }
        ];

        for (const { delta, weights } of sumInvariantCases) {
            const shares = distributeAmountProportionally(delta, weights);
            expect(sum(shares), `delta=${delta} weights=${JSON.stringify(weights)}`).toBe(delta);
        }
    });

    test('rows with non-positive weight always receive zero', () => {
        const shares = distributeAmountProportionally(101, [50, 0, -10, 25]);
        expect(shares[1]).toBe(0);
        expect(shares[2]).toBe(0);
        expect(sum(shares)).toBe(101);
    });
});

describe('calculateAdjustment', () => {
    const testCases: {
        name: string;
        total: number;
        amounts: number[];
        expected: number;
    }[] = [
        { name: 'zero difference', total: 600, amounts: [100, 200, 300], expected: 0 },
        { name: 'positive difference (tax-exclusive receipt)', total: 1080, amounts: [500, 500], expected: 80 },
        { name: 'negative difference', total: 900, amounts: [500, 500], expected: -100 },
        { name: 'discount line as negative amount', total: 450, amounts: [500, -50], expected: 0 },
        { name: 'empty items', total: 1234, amounts: [], expected: 1234 },
        { name: 'zero total', total: 0, amounts: [100, -100], expected: 0 }
    ];

    testCases.forEach(testCase => {
        test(testCase.name, () => {
            expect(calculateAdjustment(testCase.total, testCase.amounts)).toBe(testCase.expected);
        });
    });

    test('adjustment restores the sum invariant', () => {
        const total = 108000;
        const amounts = [39800, 29800, 30400];
        const adjustment = calculateAdjustment(total, amounts);
        expect(sum(amounts) + adjustment).toBe(total);
    });
});

describe('shouldShowReceiptConfirmation', () => {
    function buildResponse(items?: { amount: number }[], type: number = TransactionType.Expense): RecognizedReceiptDetailsResponse {
        return {
            type: type,
            sourceAmount: 1000,
            items: items
        } as RecognizedReceiptDetailsResponse;
    }

    test('returns false when items is missing', () => {
        expect(shouldShowReceiptConfirmation(buildResponse(undefined))).toBe(false);
    });

    test('returns false when items is empty', () => {
        expect(shouldShowReceiptConfirmation(buildResponse([]))).toBe(false);
    });

    test('returns false when items has a single item (legacy single transaction flow)', () => {
        expect(shouldShowReceiptConfirmation(buildResponse([{ amount: 1000 }]))).toBe(false);
    });

    test('returns true when an expense receipt has two or more items', () => {
        expect(shouldShowReceiptConfirmation(buildResponse([{ amount: 600 }, { amount: 400 }]))).toBe(true);
    });

    test('returns false for an income receipt even with two or more items (legacy flow)', () => {
        expect(shouldShowReceiptConfirmation(buildResponse([{ amount: 600 }, { amount: 400 }], TransactionType.Income))).toBe(false);
    });

    test('returns false for a transfer receipt even with two or more items (no destination account in the confirmation flow)', () => {
        expect(shouldShowReceiptConfirmation(buildResponse([{ amount: 600 }, { amount: 400 }], TransactionType.Transfer))).toBe(false);
    });
});

describe('getDefaultAttachGeoLocation / isHighConfidenceLocation', () => {
    test('high confidence location with coordinate defaults to on', () => {
        const response = {
            type: 2,
            geoLocation: { latitude: 35.0, longitude: 139.0 },
            location: { displayName: 'somewhere', confidence: 'high' }
        } as RecognizedReceiptDetailsResponse;

        expect(getDefaultAttachGeoLocation(response)).toBe(true);
    });

    test('low confidence location defaults to off', () => {
        const response = {
            type: 2,
            geoLocation: { latitude: 35.0, longitude: 139.0 },
            location: { displayName: 'somewhere', confidence: 'low' }
        } as RecognizedReceiptDetailsResponse;

        expect(getDefaultAttachGeoLocation(response)).toBe(false);
    });

    test('missing coordinate defaults to off even with high confidence label', () => {
        const response = {
            type: 2,
            location: { displayName: 'somewhere', confidence: 'high' }
        } as RecognizedReceiptDetailsResponse;

        expect(getDefaultAttachGeoLocation(response)).toBe(false);
    });

    test('missing location defaults to off', () => {
        const response = {
            type: 2,
            geoLocation: { latitude: 35.0, longitude: 139.0 }
        } as RecognizedReceiptDetailsResponse;

        expect(getDefaultAttachGeoLocation(response)).toBe(false);
        expect(isHighConfidenceLocation(undefined)).toBe(false);
    });
});
