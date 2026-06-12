export const ALL_LLM_PROMPT_PLACEHOLDERS: string[] = [
    'CurrentDateTime',
    'AllExpenseCategoryNames',
    'AllIncomeCategoryNames',
    'AllTransferCategoryNames',
    'AllAccountNames',
    'AllTagNames'
];

export interface LlmPromptPlaceholderHint {
    readonly placeholder: string;
    readonly descriptionKey: string;
}

const ALL_LLM_PROMPT_PLACEHOLDER_DESCRIPTION_KEYS: Record<string, string> = {
    'CurrentDateTime': 'The current date and time',
    'AllExpenseCategoryNames': 'All visible expense category names (secondary categories only)',
    'AllIncomeCategoryNames': 'All visible income category names (secondary categories only)',
    'AllTransferCategoryNames': 'All visible transfer category names (secondary categories only)',
    'AllAccountNames': 'All visible account names',
    'AllTagNames': 'All visible tag names'
};

export function getAllLlmPromptPlaceholderHints(): LlmPromptPlaceholderHint[] {
    return ALL_LLM_PROMPT_PLACEHOLDERS.map(placeholderName => ({
        placeholder: '{{.' + placeholderName + '}}',
        descriptionKey: ALL_LLM_PROMPT_PLACEHOLDER_DESCRIPTION_KEYS[placeholderName] as string
    }));
}

export function getMissingLlmPromptPlaceholders(content: string): string[] {
    const missingPlaceholders: string[] = [];

    for (const placeholderName of ALL_LLM_PROMPT_PLACEHOLDERS) {
        const placeholderRegex = new RegExp('\\{\\{\\s*\\.' + placeholderName + '\\s*\\}\\}');

        if (!placeholderRegex.test(content)) {
            missingPlaceholders.push('{{.' + placeholderName + '}}');
        }
    }

    return missingPlaceholders;
}
