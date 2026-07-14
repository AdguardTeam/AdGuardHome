/**
 * Checks if versions are equal.
 * Please note, that this method strips the "v" prefix.
 *
 * @param left {string} - left version
 * @param right {string} - right version
 * @return {boolean} true if versions are equal
 */
export const areEqualVersions = (left: any, right: any) => {
    if (!left || !right) {
        return false;
    }

    const leftVersion = left.replace(/^v/, '');
    const rightVersion = right.replace(/^v/, '');
    return leftVersion === rightVersion;
};
