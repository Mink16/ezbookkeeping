package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReconcileRecognizedReceiptItems(t *testing.T) {
	tests := []struct {
		name                   string
		transactionType        TransactionType
		totalAmount            int64
		itemAmounts            []int64
		expectedTotalAmount    int64
		expectedAdjustment     bool
		expectedAdjustmentSize int64
		expectedAdjustmentKind string
	}{
		{
			name:                "DeltaZeroReturnsItemsUnchanged",
			transactionType:     TRANSACTION_TYPE_EXPENSE,
			totalAmount:         150000,
			itemAmounts:         []int64{100000, 50000},
			expectedTotalAmount: 150000,
			expectedAdjustment:  false,
		},
		{
			name:                   "ExclusiveTax8Percent",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            108000, // 1080 yen paid
			itemAmounts:            []int64{60000, 40000},
			expectedTotalAmount:    108000,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 8000, // 80 yen = 8% of 1000 yen
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_TAX,
		},
		{
			name:                   "ExclusiveTax10Percent",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            110000, // 1100 yen paid
			itemAmounts:            []int64{60000, 40000},
			expectedTotalAmount:    110000,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 10000, // 100 yen = 10% of 1000 yen
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_TAX,
		},
		{
			name:                   "ExclusiveTaxMixed8And10Percent",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            109000, // 1090 yen paid: 500 yen at 8% (40) + 500 yen at 10% (50)
			itemAmounts:            []int64{50000, 50000},
			expectedTotalAmount:    109000,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 9000,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_TAX,
		},
		{
			name:                   "ExclusiveTaxWithNegativeDiscountItem",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            43200, // 432 yen paid: (500 - 100) yen + 8% tax (32 yen)
			itemAmounts:            []int64{50000, -10000},
			expectedTotalAmount:    43200,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 3200,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_TAX,
		},
		{
			name:                "NegativeDiscountItemDeltaZero",
			transactionType:     TRANSACTION_TYPE_EXPENSE,
			totalAmount:         40000,
			itemAmounts:         []int64{50000, -10000},
			expectedTotalAmount: 40000,
			expectedAdjustment:  false,
		},
		{
			name:                "MissingTotalAmountTrustsItemSum",
			transactionType:     TRANSACTION_TYPE_EXPENSE,
			totalAmount:         0,
			itemAmounts:         []int64{50000, 30000},
			expectedTotalAmount: 80000,
			expectedAdjustment:  false,
		},
		{
			name:                   "RecognitionDifferenceTooLargeForTax",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            150000,
			itemAmounts:            []int64{60000, 40000},
			expectedTotalAmount:    150000,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 50000, // 500 yen, far beyond 10% + 2 yen
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_UNKNOWN,
		},
		{
			name:                   "RecognitionDifferenceNegativeDelta",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            90000,
			itemAmounts:            []int64{60000, 40000},
			expectedTotalAmount:    90000,
			expectedAdjustment:     true,
			expectedAdjustmentSize: -10000,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_UNKNOWN,
		},
		{
			name:                   "TaxLowerBoundExactlyMinus2Yen",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            107800, // delta = floor(100000 * 8%) - 200 = 7800
			itemAmounts:            []int64{100000},
			expectedTotalAmount:    107800,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 7800,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_TAX,
		},
		{
			name:                   "TaxJustBelowLowerBound",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            107799, // delta = 7799, one unit below the lower bound
			itemAmounts:            []int64{100000},
			expectedTotalAmount:    107799,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 7799,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_UNKNOWN,
		},
		{
			name:                   "TaxUpperBoundExactlyPlus2Yen",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            110200, // delta = ceil(100000 * 10%) + 200 = 10200
			itemAmounts:            []int64{100000},
			expectedTotalAmount:    110200,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 10200,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_TAX,
		},
		{
			name:                   "TaxJustAboveUpperBound",
			transactionType:        TRANSACTION_TYPE_EXPENSE,
			totalAmount:            110201, // delta = 10201, one unit above the upper bound
			itemAmounts:            []int64{100000},
			expectedTotalAmount:    110201,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 10201,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_UNKNOWN,
		},
		{
			name:                   "IncomeTypeNeverLabeledAsTax",
			transactionType:        TRANSACTION_TYPE_INCOME,
			totalAmount:            108000, // delta would be 8% tax for an expense
			itemAmounts:            []int64{60000, 40000},
			expectedTotalAmount:    108000,
			expectedAdjustment:     true,
			expectedAdjustmentSize: 8000,
			expectedAdjustmentKind: RECOGNIZED_RECEIPT_ADJUSTMENT_KIND_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]*RecognizedReceiptItemResponse, 0, len(tt.itemAmounts))

			for i := 0; i < len(tt.itemAmounts); i++ {
				items = append(items, &RecognizedReceiptItemResponse{Amount: tt.itemAmounts[i]})
			}

			actualTotalAmount, actualItems := ReconcileRecognizedReceiptItems(tt.transactionType, tt.totalAmount, items)

			assert.Equal(t, tt.expectedTotalAmount, actualTotalAmount)

			var actualItemAmountSum int64

			for i := 0; i < len(actualItems); i++ {
				actualItemAmountSum += actualItems[i].Amount
			}

			assert.Equal(t, actualTotalAmount, actualItemAmountSum, "sum invariant must hold")

			if tt.expectedAdjustment {
				assert.Equal(t, len(tt.itemAmounts)+1, len(actualItems))

				adjustmentItem := actualItems[len(actualItems)-1]
				assert.True(t, adjustmentItem.IsAdjustment)
				assert.Equal(t, tt.expectedAdjustmentSize, adjustmentItem.Amount)
				assert.Equal(t, tt.expectedAdjustmentKind, adjustmentItem.AdjustmentKind)
				assert.Empty(t, adjustmentItem.Comment)

				for i := 0; i < len(actualItems)-1; i++ {
					assert.False(t, actualItems[i].IsAdjustment)
					assert.Equal(t, tt.itemAmounts[i], actualItems[i].Amount)
				}
			} else {
				assert.Equal(t, len(tt.itemAmounts), len(actualItems))

				for i := 0; i < len(actualItems); i++ {
					assert.False(t, actualItems[i].IsAdjustment)
				}
			}
		})
	}
}

func TestRecognizedReceiptItemResponseJsonMarshal_ZeroAmountIsAlwaysSerialized(t *testing.T) {
	// zero is a meaningful item amount (free items, fully discounted lines); the frontend
	// requires the amount field to be present, so it must never be dropped by omitempty
	item := &RecognizedReceiptItemResponse{
		Amount:  0,
		Comment: "レジ袋",
	}

	data, err := json.Marshal(item)

	assert.Nil(t, err)
	assert.Contains(t, string(data), "\"amount\":0")
}
