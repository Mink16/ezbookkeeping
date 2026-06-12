import { describe, expect, test } from 'vitest';

import { ALL_LLM_PROMPT_PLACEHOLDERS, getAllLlmPromptPlaceholderHints, getMissingLlmPromptPlaceholders } from '@/lib/llm_prompt.ts';

const CONTENT_WITH_ALL_PLACEHOLDERS = 'Time: {{.CurrentDateTime}}\n' +
    'Expense: {{.AllExpenseCategoryNames}}\n' +
    'Income: {{.AllIncomeCategoryNames}}\n' +
    'Transfer: {{.AllTransferCategoryNames}}\n' +
    'Accounts: {{.AllAccountNames}}\n' +
    'Tags: {{.AllTagNames}}';

describe('getMissingLlmPromptPlaceholders', () => {
    test('returns empty array when all placeholders exist', () => {
        expect(getMissingLlmPromptPlaceholders(CONTENT_WITH_ALL_PLACEHOLDERS)).toStrictEqual([]);
    });

    test('returns all placeholders for content without any placeholder', () => {
        expect(getMissingLlmPromptPlaceholders('You are a financial assistant.')).toStrictEqual([
            '{{.CurrentDateTime}}',
            '{{.AllExpenseCategoryNames}}',
            '{{.AllIncomeCategoryNames}}',
            '{{.AllTransferCategoryNames}}',
            '{{.AllAccountNames}}',
            '{{.AllTagNames}}'
        ]);
    });

    test('returns all placeholders for empty content', () => {
        expect(getMissingLlmPromptPlaceholders('').length).toBe(ALL_LLM_PROMPT_PLACEHOLDERS.length);
    });

    test('returns only missing placeholders', () => {
        const content = 'Time: {{.CurrentDateTime}}\nAccounts: {{.AllAccountNames}}';
        expect(getMissingLlmPromptPlaceholders(content)).toStrictEqual([
            '{{.AllExpenseCategoryNames}}',
            '{{.AllIncomeCategoryNames}}',
            '{{.AllTransferCategoryNames}}',
            '{{.AllTagNames}}'
        ]);
    });

    test('accepts whitespace variants inside the placeholder', () => {
        expect(getMissingLlmPromptPlaceholders(CONTENT_WITH_ALL_PLACEHOLDERS.replaceAll('{{.', '{{ .'))).toStrictEqual([]);
        expect(getMissingLlmPromptPlaceholders('{{.CurrentDateTime }}')).not.toContain('{{.CurrentDateTime}}');
    });

    test('does not treat prefixed placeholder names as matches', () => {
        expect(getMissingLlmPromptPlaceholders('{{.AllTagNamesExtra}}')).toContain('{{.AllTagNames}}');
    });

    test('does not report duplicated placeholders as missing', () => {
        const content = '{{.CurrentDateTime}} and again {{.CurrentDateTime}}';
        expect(getMissingLlmPromptPlaceholders(content)).not.toContain('{{.CurrentDateTime}}');
    });
});

describe('getAllLlmPromptPlaceholderHints', () => {
    test('returns one hint with a description key for every placeholder', () => {
        const hints = getAllLlmPromptPlaceholderHints();

        expect(hints.length).toBe(ALL_LLM_PROMPT_PLACEHOLDERS.length);

        for (const [index, placeholderName] of ALL_LLM_PROMPT_PLACEHOLDERS.entries()) {
            expect(hints[index]?.placeholder).toBe('{{.' + placeholderName + '}}');
            expect(hints[index]?.descriptionKey).toBeTruthy();
        }
    });

    test('hint placeholders satisfy the missing placeholder check', () => {
        const content = getAllLlmPromptPlaceholderHints().map(hint => hint.placeholder).join('\n');
        expect(getMissingLlmPromptPlaceholders(content)).toStrictEqual([]);
    });
});
