export const ALL_LLM_PROMPT_PLACEHOLDERS: string[] = [
    'CurrentDateTime',
    'AllExpenseCategoryNames',
    'AllIncomeCategoryNames',
    'AllTransferCategoryNames',
    'AllAccountNames',
    'AllTagNames'
];

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
