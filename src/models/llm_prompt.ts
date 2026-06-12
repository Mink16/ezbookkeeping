export const DEFAULT_LLM_PROMPT_ID: string = '0';

export class LlmPrompt {
    public id: string;
    public name: string;
    public content: string;
    public isActive: boolean;

    private constructor(id: string, name: string, content: string, isActive: boolean) {
        this.id = id;
        this.name = name;
        this.content = content;
        this.isActive = isActive;
    }

    public toCreateRequest(): LlmPromptCreateRequest {
        return {
            name: this.name,
            content: this.content
        };
    }

    public toModifyRequest(): LlmPromptModifyRequest {
        return {
            id: this.id,
            name: this.name,
            content: this.content
        };
    }

    public clone(): LlmPrompt {
        return new LlmPrompt(this.id, this.name, this.content, this.isActive);
    }

    public static of(promptResponse: LlmPromptInfoResponse): LlmPrompt {
        return new LlmPrompt(promptResponse.id, promptResponse.name, promptResponse.content || '', promptResponse.isActive);
    }

    public static ofMulti(promptResponses: LlmPromptInfoResponse[]): LlmPrompt[] {
        const prompts: LlmPrompt[] = [];

        for (const promptResponse of promptResponses) {
            prompts.push(LlmPrompt.of(promptResponse));
        }

        return prompts;
    }

    public static createNewPrompt(name?: string, content?: string): LlmPrompt {
        return new LlmPrompt('', name || '', content || '', false);
    }
}

export interface LlmPromptCreateRequest {
    readonly name: string;
    readonly content: string;
}

export interface LlmPromptModifyRequest {
    readonly id: string;
    readonly name: string;
    readonly content: string;
}

export interface LlmPromptDeleteRequest {
    readonly id: string;
}

export interface LlmPromptSetActiveRequest {
    readonly id: string;
}

export interface LlmPromptPreviewRequest {
    readonly content: string;
}

export interface LlmPromptInfoResponse {
    readonly id: string;
    readonly name: string;
    readonly content?: string;
    readonly isActive: boolean;
    readonly createdTime: number;
    readonly updatedTime: number;
}

export interface LlmPromptListResponse {
    readonly defaultContent: string;
    readonly prompts: LlmPromptInfoResponse[];
}

export interface LlmPromptPreviewResponse {
    readonly renderedContent: string;
}
