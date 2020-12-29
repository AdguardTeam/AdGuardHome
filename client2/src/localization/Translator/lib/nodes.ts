export enum NODE_TYPES {
    PLACEHOLDER = 'placeholder',
    TEXT = 'text',
    TAG = 'tag',
    VOID_TAG = 'void_tag',
}

export interface NODE {
    type: NODE_TYPES;
    value: string | keyof HTMLElementTagNameMap;
    children?: NODE[];
}

export const isTextNode = (node: NODE) => {
    return node?.type === NODE_TYPES.TEXT;
};

export const isTagNode = (node: NODE) => {
    return node?.type === NODE_TYPES.TAG;
};

export const isPlaceholderNode = (node: NODE) => {
    return node?.type === NODE_TYPES.PLACEHOLDER;
};

export const isVoidTagNode = (node: NODE) => {
    return node?.type === NODE_TYPES.VOID_TAG;
};

export const placeholderNode = (value: string) => {
    return { type: NODE_TYPES.PLACEHOLDER, value };
};

export const textNode = (str: string) => {
    return { type: NODE_TYPES.TEXT, value: str };
};

export const tagNode = (tagName: keyof HTMLElementTagNameMap, children: NODE[]) => {
    const value = tagName.trim();
    return { type: NODE_TYPES.TAG, value, children };
};

export const voidTagNode = (tagName: keyof HTMLElementTagNameMap) => {
    const value = tagName.trim();
    return { type: NODE_TYPES.VOID_TAG, value };
};

export const isNode = (checked: any) => {
    return !!checked?.type;
};
