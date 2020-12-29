import {
    isTextNode,
    isTagNode,
    isPlaceholderNode,
    isVoidTagNode,
    NODE,
} from './nodes';

/**
 * Checks if target is function
 * @param target
 * @returns {boolean}
 */
const isFunction = (target: any) => {
    return typeof target === 'function';
};

type FormatingFunc<T = string> = (chunks: string) => T;
export type AllowedValues<T> = Record<string, number | string | FormatingFunc<T>>;

/**
 * This function accepts an AST (abstract syntax tree) which is a result
 * of the parser function call, and converts tree nodes into array of strings replacing node
 * values with provided values.
 * Values is a map with functions or strings, where each key is related to placeholder value
 * or tag value
 * e.g.
 * string "text <tag>tag text</tag> %placeholder%" is parsed into next AST
 *
 *      [
 *          { type: 'text', value: 'text ' },
 *          {
 *              type: 'tag',
 *              value: 'tag',
 *              children: [{ type: 'text', value: 'tag text' }],
 *          },
 *          { type: 'text', value: ' ' },
 *          { type: 'placeholder', value: 'placeholder' }
 *      ];
 *
 * this AST after format and next values
 *
 *      {
 *          // here used template strings, but it can be react components as well
 *          tag: (chunks) => `<b>${chunks}</b>`,
 *          placeholder: 'placeholder text'
 *      }
 *
 * will return next array
 *
 * [ 'text ', '<b>tag text</b>', ' ', 'placeholder text' ]
 *
 * as you can see, <tag> was replaced by <b>, and placeholder was replaced by placeholder text
 *
 * @param ast - AST (abstract syntax tree)
 * @param values
 * @returns {[]}
 */
const format = <T = any | string>(ast: NODE[], values: AllowedValues<T>) => {
    const result: (string | T)[] = [];
    let i = 0;
    while (i < ast.length) {
        const currentNode = ast[i];
        // if current node is text node, there is nothing to change, append value to the result
        if (isTextNode(currentNode)) {
            result.push(currentNode.value);
        } else if (isTagNode(currentNode)) {
            const children = [...format(currentNode.children ? currentNode.children : [], values)].join('');
            const value = values[currentNode.value];
            if (typeof value === 'string' || typeof value === 'number' || typeof value === 'function') {
                if (isFunction(value)) {
                    result.push((value as FormatingFunc<T>)(children));
                } else {
                    result.push(value.toString());
                }
            } else {
                throw new Error(`Value ${currentNode.value} wasn't provided`);
            }
        } else if (isVoidTagNode(currentNode)) {
            const value = values[currentNode.value];
            if (typeof value === 'string' || typeof value === 'number') {
                result.push(value.toString());
            } else {
                throw new Error(`Value ${currentNode.value} wasn't provided`);
            }
        } else if (isPlaceholderNode(currentNode)) {
            const value = values[currentNode.value];
            if (typeof value === 'string' || typeof value === 'number') {
                result.push(value.toString());
            } else {
                throw new Error(`Value ${currentNode.value} wasn't provided`);
            }
        }
        i += 1;
    }

    return result;
};

export default format;
