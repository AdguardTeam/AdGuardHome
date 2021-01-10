import parser from './parser';
import { isTextNode, NODE } from './nodes';

/**
 * Compares two AST (abstract syntax tree) structures,
 * view tests for examples
 * @param baseAst
 * @param targetAst
 * @returns {boolean}
 */
const areAstStructuresSame = (baseAst: NODE[], targetAst: NODE[]) => {
    const textNodeFilter = (node: NODE) => {
        return !isTextNode(node);
    };

    const filteredBaseAst = baseAst.filter(textNodeFilter);

    const filteredTargetAst = targetAst.filter(textNodeFilter);

    // if AST structures have different lengths, they are not equal
    if (filteredBaseAst.length !== filteredTargetAst.length) {
        return false;
    }

    for (let i = 0; i < filteredBaseAst.length; i += 1) {
        const baseNode = filteredBaseAst[i];

        const targetNode = filteredTargetAst.find((node) => {
            return node.type === baseNode.type && node.value === baseNode.value;
        });

        if (!targetNode) {
            return false;
        }

        if (targetNode.children && baseNode.children) {
            const areChildrenSame = areAstStructuresSame(baseNode.children, targetNode.children);
            if (!areChildrenSame) {
                return false;
            }
        }
    }

    return true;
};

/**
 * Validates translation against base string by AST (abstract syntax tree) structure
 * @param baseStr
 * @param targetStr
 * @returns {boolean}
 */
export const isTargetStrValid = (baseStr: string, targetStr: string) => {
    const baseAst = parser(baseStr);
    const targetAst = parser(targetStr);

    const result = areAstStructuresSame(baseAst, targetAst);

    return result;
};
